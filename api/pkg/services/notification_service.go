package services

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"
	cloudevents "github.com/cloudevents/sdk-go/v2"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// NotificationService sends out notifications to mobile phones
type NotificationService struct {
	logger                      telemetry.Logger
	tracer                      telemetry.Tracer
	phoneNotificationRepository repositories.PhoneNotificationRepository
	phoneRepository             repositories.PhoneRepository
	messagingClient             *messaging.Client
	eventDispatcher             *EventDispatcher
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	messagingClient *messaging.Client,
	phoneRepository repositories.PhoneRepository,
	phoneNotificationRepository repositories.PhoneNotificationRepository,
	dispatcher *EventDispatcher,
) (s *NotificationService) {
	return &NotificationService{
		logger:                      logger.WithService(fmt.Sprintf("%T", s)),
		tracer:                      tracer,
		messagingClient:             messagingClient,
		phoneNotificationRepository: phoneNotificationRepository,
		phoneRepository:             phoneRepository,
		eventDispatcher:             dispatcher,
	}
}

// NotificationSendParams are parameters for sending a notification
type NotificationSendParams struct {
	UserID              entities.UserID
	PhoneID             uuid.UUID
	PhoneNotificationID uuid.UUID
	MessageID           uuid.UUID
}

// Send sends a message when a message is sent
func (service *NotificationService) Send(ctx context.Context, params *NotificationSendParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	phone, err := service.phoneRepository.LoadByID(ctx, params.PhoneID)
	if err != nil {
		service.updateStatus(ctx, params.PhoneNotificationID, entities.PhoneNotificationStatusFailed)
		msg := fmt.Sprintf("cannot load phone with userID [%s] and phoneID [%s]", params.UserID, params.PhoneID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if phone.FcmToken == nil {
		service.updateStatus(ctx, params.PhoneNotificationID, entities.PhoneNotificationStatusFailed)
		msg := fmt.Sprintf("phone with id [%s] has no FCM token", phone.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.NewError(msg))
	}

	result, err := service.messagingClient.Send(ctx, &messaging.Message{
		Data: map[string]string{
			"KEY_MESSAGE_ID": params.MessageID.String(),
		},
		Token: *phone.FcmToken,
	})
	if err != nil {
		service.updateStatus(ctx, params.PhoneNotificationID, entities.PhoneNotificationStatusFailed)
		msg := fmt.Sprintf("cannot send notification for message [%s] to phone [%s]", params.MessageID, phone.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("sent notification [%s] for message [%s] to phone [%s]", result, params.MessageID, phone.ID))
	service.updateStatus(ctx, params.PhoneNotificationID, entities.PhoneNotificationStatusSent)
	return nil
}

// NotificationScheduleParams are parameters for sending a notification
type NotificationScheduleParams struct {
	UserID    entities.UserID
	Owner     string
	Source    string
	MessageID uuid.UUID
}

// Schedule a notification to be sent to a phone
func (service *NotificationService) Schedule(ctx context.Context, params *NotificationScheduleParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	phone, err := service.phoneRepository.Load(ctx, params.UserID, params.Owner)
	if err != nil {
		msg := fmt.Sprintf("cannot load phone with userID [%s] and phone [%s]", params.UserID, params.Owner)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	notification := &entities.PhoneNotification{
		ID:          uuid.New(),
		MessageID:   params.MessageID,
		UserID:      params.UserID,
		PhoneID:     phone.ID,
		Status:      entities.PhoneNotificationStatusPending,
		ScheduledAt: time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err = service.phoneNotificationRepository.Schedule(ctx, phone.MessagesPerMinute, notification); err != nil {
		msg := fmt.Sprintf("cannot schedule notification for message [%s] to phone [%s]", params.MessageID, phone.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	event, err := service.createEvent(params.Source, notification)
	if err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot create cloud event for notification [%s]", notification.ID))
	}

	if err = service.eventDispatcher.DispatchWithTimeout(ctx, event, notification.ScheduledAt.Sub(time.Now())); err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot dispatch event [%s] for notification [%s]", event.Type(), notification.ID))
	}

	ctxLogger.Info(fmt.Sprintf("message with id [%s] notification scheduled for [%s] with id [%s]", params.MessageID, notification.ScheduledAt, notification.ID))
	return nil
}

func (service *NotificationService) createEvent(source string, notification *entities.PhoneNotification) (cloudevents.Event, error) {
	event := cloudevents.NewEvent()

	event.SetSource(source)
	event.SetType(events.EventTypeMessageNotificationScheduled)
	event.SetTime(time.Now().UTC())
	event.SetID(uuid.New().String())

	payload := events.MessageNotificationScheduledPayload{
		MessageID:      notification.MessageID,
		UserID:         notification.UserID,
		PhoneID:        notification.PhoneID,
		ScheduledAt:    notification.ScheduledAt,
		NotificationID: notification.ID,
	}

	if err := event.SetData(cloudevents.ApplicationJSON, payload); err != nil {
		msg := fmt.Sprintf("cannot encode %T [%#+v] as JSON", payload, payload)
		return event, stacktrace.Propagate(err, msg)
	}

	return event, nil
}

func (service *NotificationService) updateStatus(ctx context.Context, notificationID uuid.UUID, status entities.PhoneNotificationStatus) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	err := service.phoneNotificationRepository.UpdateStatus(ctx, notificationID, status)
	if err != nil {
		msg := fmt.Sprintf("cannot update status of notificaiton with id [%s] to [%s]", notificationID, status)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return
	}

	ctxLogger.Info(fmt.Sprintf("updated status of notificaiton with id [%s] to [%s]", notificationID, status))
}
