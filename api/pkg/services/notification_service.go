package services

import (
	"context"
	"errors"
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
	Source              string
	ScheduledAt         time.Time
	MessageID           uuid.UUID
}

// Send sends a message when a message is sent
func (service *NotificationService) Send(ctx context.Context, params *NotificationSendParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	phone, err := service.phoneRepository.LoadByID(ctx, params.PhoneID)
	if err != nil {
		msg := fmt.Sprintf("cannot load phone with userID [%s] and phoneID [%s]", params.UserID, params.PhoneID)
		return service.handleNotificationFailed(ctx, errors.New(msg), params)
	}

	if phone.FcmToken == nil {
		msg := fmt.Sprintf("phone with id [%s] has no FCM token", phone.ID)
		return service.handleNotificationFailed(ctx, errors.New(msg), params)
	}

	result, err := service.messagingClient.Send(ctx, &messaging.Message{
		Data: map[string]string{
			"KEY_MESSAGE_ID": params.MessageID.String(),
		},
		Token: *phone.FcmToken,
	})
	if err != nil {
		return service.handleNotificationFailed(ctx, err, params)
	}
	return service.handleNotificationSent(ctx, result, params)
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

func (service *NotificationService) handleNotificationFailed(ctx context.Context, err error, params *NotificationSendParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	msg := fmt.Sprintf("cannot send notification for message [%s] to phone [%s]", params.MessageID, params.PhoneNotificationID)
	ctxLogger.Warn(stacktrace.Propagate(err, msg))

	event, err := service.createMessageNotificationFailedEvent(params.Source, err.Error(), params)
	if err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot create [%s] event for notification [%s]", events.EventTypeMessageNotificationFailed, params.PhoneNotificationID))
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot dispatch event [%s] for notification [%s]", event.Type(), params.PhoneNotificationID))
	}

	service.updateStatus(ctx, params.PhoneNotificationID, entities.PhoneNotificationStatusFailed)
	return nil
}

func (service *NotificationService) handleNotificationSent(ctx context.Context, result string, params *NotificationSendParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	ctxLogger.Info(fmt.Sprintf("sent notification [%s] for message [%s] to phone [%s]", result, params.MessageID, params.PhoneID))

	event, err := service.createMessageNotificationSentEvent(params.Source, result, params)
	if err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot create [%s] event for notification [%s]", events.EventTypeMessageNotificationSent, params.PhoneNotificationID))
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot dispatch event [%s] for notification [%s]", event.Type(), params.PhoneNotificationID))
	}

	service.updateStatus(ctx, params.PhoneNotificationID, entities.PhoneNotificationStatusSent)
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

func (service *NotificationService) createMessageNotificationSentEvent(source string, fcmMessageID string, params *NotificationSendParams) (cloudevents.Event, error) {
	event := cloudevents.NewEvent()

	event.SetSource(source)
	event.SetType(events.EventTypeMessageNotificationSent)
	event.SetTime(time.Now().UTC())
	event.SetID(uuid.New().String())

	payload := events.MessageNotificationSentPayload{
		MessageID:          params.MessageID,
		UserID:             params.UserID,
		PhoneID:            params.PhoneID,
		ScheduledAt:        params.ScheduledAt,
		FcmMessageID:       fcmMessageID,
		NotificationSentAt: time.Now().UTC(),
		NotificationID:     params.PhoneNotificationID,
	}

	if err := event.SetData(cloudevents.ApplicationJSON, payload); err != nil {
		msg := fmt.Sprintf("cannot encode %T [%#+v] as JSON", payload, payload)
		return event, stacktrace.Propagate(err, msg)
	}

	return event, nil
}

func (service *NotificationService) createMessageNotificationFailedEvent(source string, errorMessage string, params *NotificationSendParams) (cloudevents.Event, error) {
	event := cloudevents.NewEvent()

	event.SetSource(source)
	event.SetType(events.EventTypeMessageNotificationFailed)
	event.SetTime(time.Now().UTC())
	event.SetID(uuid.New().String())

	payload := events.MessageNotificationFailedPayload{
		MessageID:            params.MessageID,
		UserID:               params.UserID,
		PhoneID:              params.PhoneID,
		ErrorMessage:         errorMessage,
		NotificationFailedAt: time.Now().UTC(),
		NotificationID:       params.PhoneNotificationID,
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
	}

	ctxLogger.Info(fmt.Sprintf("updated status of notificaiton with id [%s] to [%s]", notificationID, status))
}
