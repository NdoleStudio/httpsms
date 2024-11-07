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

// PhoneNotificationService sends out notifications to mobile phones
type PhoneNotificationService struct {
	service
	logger                      telemetry.Logger
	tracer                      telemetry.Tracer
	phoneNotificationRepository repositories.PhoneNotificationRepository
	phoneRepository             repositories.PhoneRepository
	messagingClient             *messaging.Client
	eventDispatcher             *EventDispatcher
}

// NewNotificationService creates a new PhoneNotificationService
func NewNotificationService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	messagingClient *messaging.Client,
	phoneRepository repositories.PhoneRepository,
	phoneNotificationRepository repositories.PhoneNotificationRepository,
	dispatcher *EventDispatcher,
) (s *PhoneNotificationService) {
	return &PhoneNotificationService{
		logger:                      logger.WithService(fmt.Sprintf("%T", s)),
		tracer:                      tracer,
		messagingClient:             messagingClient,
		phoneNotificationRepository: phoneNotificationRepository,
		phoneRepository:             phoneRepository,
		eventDispatcher:             dispatcher,
	}
}

// DeleteAllForUser deletes all entities.PhoneNotification for an entities.UserID.
func (service *PhoneNotificationService) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.phoneNotificationRepository.DeleteAllForUser(ctx, userID); err != nil {
		msg := fmt.Sprintf("could not delete all [entities.PhoneNotification] for user with ID [%s]", userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted all [entities.PhoneNotification] for user with ID [%s]", userID))
	return nil
}

// SendHeartbeatFCM sends a heartbeat message so the phone can request a heartbeat
func (service *PhoneNotificationService) SendHeartbeatFCM(ctx context.Context, payload *events.PhoneHeartbeatMissedPayload) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	phone, err := service.phoneRepository.LoadByID(ctx, payload.UserID, payload.PhoneID)
	if err != nil {
		msg := fmt.Sprintf("cannot load phone with userID [%s] and phoneID [%s]", payload.UserID, payload.PhoneID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if phone.FcmToken == nil {
		msg := fmt.Sprintf("phone with id [%s] has no FCM token", phone.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	result, err := service.messagingClient.Send(ctx, &messaging.Message{
		Data: map[string]string{
			"KEY_HEARTBEAT_ID": time.Now().UTC().Format(time.RFC3339),
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
		Token: *phone.FcmToken,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot send heartbeat FCM to phone with id [%s] for user [%s]", phone.ID, phone.UserID)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return nil
	}

	ctxLogger.Info(fmt.Sprintf("successfully sent heartbeat FCM [%s] to phone with ID [%s] for user [%s] and monitor [%s]", result, payload.PhoneID, payload.UserID, payload.MonitorID))
	return nil
}

// PhoneNotificationSendParams are parameters for sending a notification
type PhoneNotificationSendParams struct {
	UserID              entities.UserID
	PhoneID             uuid.UUID
	PhoneNotificationID uuid.UUID
	Source              string
	ScheduledAt         time.Time
	MessageID           uuid.UUID
}

// Send sends a message when a message is sent
func (service *PhoneNotificationService) Send(ctx context.Context, params *PhoneNotificationSendParams) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	phone, err := service.phoneRepository.LoadByID(ctx, params.UserID, params.PhoneID)
	if err != nil {
		msg := fmt.Sprintf("cannot load phone with userID [%s] and phoneID [%s]", params.UserID, params.PhoneID)
		return service.handleNotificationFailed(ctx, errors.New(msg), params)
	}

	if phone.FcmToken == nil {
		msg := fmt.Sprintf("phone with id [%s] has no FCM token", phone.ID)
		return service.handleNotificationFailed(ctx, errors.New(msg), params)
	}

	ttl := phone.MessageExpirationDuration()
	result, err := service.messagingClient.Send(ctx, &messaging.Message{
		Data: map[string]string{
			"KEY_MESSAGE_ID": params.MessageID.String(),
		},
		Android: &messaging.AndroidConfig{
			Priority: "normal",
			TTL:      &ttl,
		},
		Token: *phone.FcmToken,
	})
	if err != nil {
		ctxLogger.Warn(stacktrace.Propagate(err, "cannot send FCM to phone"))
		msg := fmt.Sprintf("cannot send notification for to your phone [%s]. Reinstall the httpSMS app on your Android phone.", phone.PhoneNumber)
		return service.handleNotificationFailed(ctx, errors.New(msg), params)
	}

	return service.handleNotificationSent(ctx, phone, result, params)
}

// PhoneNotificationScheduleParams are parameters for sending a notification
type PhoneNotificationScheduleParams struct {
	UserID    entities.UserID
	Owner     string
	Source    string
	Encrypted bool
	Contact   string
	Content   string
	SIM       entities.SIM
	MessageID uuid.UUID
}

// Schedule a notification to be sent to a phone
func (service *PhoneNotificationService) Schedule(ctx context.Context, params *PhoneNotificationScheduleParams) error {
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

	if err = service.dispatchMessageNotificationScheduled(ctx, params, notification); err != nil {
		ctxLogger.Error(err)
	}

	if err = service.dispatchMessageNotificationSend(ctx, params.Source, notification); err != nil {
		return service.tracer.WrapErrorSpan(span, err)
	}

	ctxLogger.Info(fmt.Sprintf("message with id [%s] notification scheduled for [%s] with id [%s]", params.MessageID, notification.ScheduledAt, notification.ID))
	return nil
}

func (service *PhoneNotificationService) dispatchMessageNotificationSend(ctx context.Context, source string, notification *entities.PhoneNotification) error {
	event, err := service.createMessageNotificationSendEvent(source, &events.MessageNotificationSendPayload{
		MessageID:      notification.MessageID,
		UserID:         notification.UserID,
		PhoneID:        notification.PhoneID,
		ScheduledAt:    notification.ScheduledAt,
		NotificationID: notification.ID,
	})
	if err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot create [%s] event for notification [%s]", events.EventTypeMessageNotificationSend, notification.ID))
	}

	if _, err = service.eventDispatcher.DispatchWithTimeout(ctx, event, notification.ScheduledAt.Sub(time.Now())); err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot dispatch event [%s] for notification [%s]", event.Type(), notification.ID))
	}
	return nil
}

func (service *PhoneNotificationService) dispatchMessageNotificationScheduled(ctx context.Context, params *PhoneNotificationScheduleParams, notification *entities.PhoneNotification) error {
	event, err := service.createMessageNotificationScheduledEvent(params.Source, &events.MessageNotificationScheduledPayload{
		MessageID:      notification.MessageID,
		Owner:          params.Owner,
		Contact:        params.Contact,
		Encrypted:      params.Encrypted,
		Content:        params.Content,
		SIM:            params.SIM,
		UserID:         notification.UserID,
		PhoneID:        notification.PhoneID,
		ScheduledAt:    notification.ScheduledAt,
		NotificationID: notification.ID,
	})
	if err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot create [%s] event for notification [%s]", events.EventTypeMessageNotificationScheduled, notification.ID))
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot dispatch event [%s] for notification [%s]", event.Type(), notification.ID))
	}
	return nil
}

func (service *PhoneNotificationService) handleNotificationFailed(ctx context.Context, err error, params *PhoneNotificationSendParams) error {
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

func (service *PhoneNotificationService) handleNotificationSent(ctx context.Context, phone *entities.Phone, result string, params *PhoneNotificationSendParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	ctxLogger.Info(fmt.Sprintf("sent notification [%s] for message [%s] to phone [%s]", result, params.MessageID, params.PhoneID))

	event, err := service.createMessageNotificationSentEvent(params.Source, phone, result, params)
	if err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot create [%s] event for notification [%s]", events.EventTypeMessageNotificationSent, params.PhoneNotificationID))
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot dispatch event [%s] for notification [%s]", event.Type(), params.PhoneNotificationID))
	}

	service.updateStatus(ctx, params.PhoneNotificationID, entities.PhoneNotificationStatusSent)
	return nil
}

func (service *PhoneNotificationService) createMessageNotificationScheduledEvent(source string, payload *events.MessageNotificationScheduledPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessageNotificationScheduled, source, payload)
}

func (service *PhoneNotificationService) createMessageNotificationSendEvent(source string, payload *events.MessageNotificationSendPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessageNotificationSend, source, payload)
}

func (service *PhoneNotificationService) createMessageNotificationSentEvent(source string, phone *entities.Phone, fcmMessageID string, params *PhoneNotificationSendParams) (cloudevents.Event, error) {
	event := cloudevents.NewEvent()

	event.SetSource(source)
	event.SetType(events.EventTypeMessageNotificationSent)
	event.SetTime(time.Now().UTC())
	event.SetID(uuid.New().String())

	payload := events.MessageNotificationSentPayload{
		MessageID:                 params.MessageID,
		UserID:                    params.UserID,
		PhoneID:                   params.PhoneID,
		ScheduledAt:               params.ScheduledAt,
		MessageExpirationDuration: phone.MessageExpirationDuration(),
		FcmMessageID:              fcmMessageID,
		NotificationSentAt:        time.Now().UTC(),
		NotificationID:            params.PhoneNotificationID,
	}

	if err := event.SetData(cloudevents.ApplicationJSON, payload); err != nil {
		msg := fmt.Sprintf("cannot encode %T [%#+v] as JSON", payload, payload)
		return event, stacktrace.Propagate(err, msg)
	}

	return event, nil
}

func (service *PhoneNotificationService) createMessageNotificationFailedEvent(source string, errorMessage string, params *PhoneNotificationSendParams) (cloudevents.Event, error) {
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

func (service *PhoneNotificationService) updateStatus(ctx context.Context, notificationID uuid.UUID, status entities.PhoneNotificationStatus) {
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
