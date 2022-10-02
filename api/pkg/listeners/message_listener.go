package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"
)

// MessageListener handles cloud events which need to update entities.Message
type MessageListener struct {
	listener
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.MessageService
}

// NewMessageListener creates a new instance of MessageListener
func NewMessageListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.MessageService,
	repository repositories.EventListenerLogRepository,
) (l *MessageListener, routes map[string]events.EventListener) {
	l = &MessageListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
		listener: listener{
			repository: repository,
		},
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessagePhoneSending:          l.OnMessagePhoneSending,
		events.EventTypeMessagePhoneSent:             l.OnMessagePhoneSent,
		events.EventTypeMessagePhoneDelivered:        l.OnMessagePhoneDelivered,
		events.EventTypeMessageSendFailed:            l.OnMessagePhoneFailed,
		events.EventTypeMessageNotificationSent:      l.onMessageNotificationSent,
		events.EventTypeMessageNotificationFailed:    l.onMessageNotificationFailed,
		events.EventTypeMessageSendExpiredCheck:      l.onMessageSendExpiredCheck,
		events.EventTypeMessageSendExpired:           l.onMessageSendExpired,
		events.EventTypeMessageNotificationScheduled: l.onMessageNotificationScheduled,
	}
}

// OnMessagePhoneSending handles the events.EventTypeMessagePhoneSending event
func (listener *MessageListener) OnMessagePhoneSending(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handled, err := listener.repository.Has(ctx, event.ID(), listener.signature(event))
	if err != nil {
		msg := fmt.Sprintf("cannot verify if event [%s] has been handled by [%T]", event.ID(), listener.signature(event))
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), listener.signature(event)))
		return nil
	}

	var payload events.MessagePhoneSendingPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	handleParams := services.HandleMessageParams{
		ID:        payload.ID,
		UserID:    payload.UserID,
		Timestamp: event.Time(),
		Source:    event.Source(),
	}

	if err = listener.service.HandleMessageSending(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle sending for message with ID [%s] for event with ID [%s]", handleParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

// OnMessagePhoneSent handles the events.EventTypeMessagePhoneSent event
func (listener *MessageListener) OnMessagePhoneSent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handled, err := listener.repository.Has(ctx, event.ID(), listener.signature(event))
	if err != nil {
		msg := fmt.Sprintf("cannot verify if event [%s] has been handled by [%T]", event.ID(), listener.signature(event))
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), listener.signature(event)))
		return nil
	}

	var payload events.MessagePhoneSentPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	handleParams := services.HandleMessageParams{
		ID:        payload.ID,
		UserID:    payload.UserID,
		Source:    event.Source(),
		Timestamp: payload.Timestamp,
	}

	if err = listener.service.HandleMessageSent(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle [%s] for message with ID [%s] for event with ID [%s]", event.Type(), handleParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

// OnMessagePhoneDelivered handles the events.EventTypeMessagePhoneDelivered event
func (listener *MessageListener) OnMessagePhoneDelivered(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handled, err := listener.repository.Has(ctx, event.ID(), listener.signature(event))
	if err != nil {
		msg := fmt.Sprintf("cannot verify if event [%s] has been handled by [%T]", event.ID(), listener.signature(event))
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), listener.signature(event)))
		return nil
	}

	var payload events.MessagePhoneDeliveredPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	handleParams := services.HandleMessageParams{
		ID:        payload.ID,
		UserID:    payload.UserID,
		Timestamp: payload.Timestamp,
	}

	if err = listener.service.HandleMessageDelivered(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle [%s] for message with ID [%s] for event with ID [%s]", event.Type(), handleParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

// OnMessagePhoneFailed handles the events.EventTypeMessageSendFailed event
func (listener *MessageListener) OnMessagePhoneFailed(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handled, err := listener.repository.Has(ctx, event.ID(), listener.signature(event))
	if err != nil {
		msg := fmt.Sprintf("cannot verify if event [%s] has been handled by [%T]", event.ID(), listener.signature(event))
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), listener.signature(event)))
		return nil
	}

	var payload events.MessageSendFailedPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	handleParams := services.HandleMessageFailedParams{
		ID:           payload.ID,
		UserID:       payload.UserID,
		ErrorMessage: payload.ErrorMessage,
		Timestamp:    payload.Timestamp,
	}

	if err = listener.service.HandleMessageFailed(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle [%s] for message with ID [%s] for event with ID [%s]", event.Type(), handleParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

// onMessageNotificationFailed handles the events.EventTypeMessageNotificationFailed event
func (listener *MessageListener) onMessageNotificationFailed(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageNotificationFailedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	message, err := listener.service.GetMessage(ctx, payload.UserID, payload.MessageID)
	if err != nil {
		msg := fmt.Sprintf("cannot load message with id [%s] and user id [%s]", payload.MessageID, payload.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	storeParams := services.MessageStoreEventParams{
		MessageID:    payload.MessageID,
		EventName:    entities.MessageEventNameFailed,
		Timestamp:    payload.NotificationFailedAt,
		ErrorMessage: &payload.ErrorMessage,
		Source:       event.Source(),
	}
	if _, err = listener.service.StoreEvent(ctx, message, storeParams); err != nil {
		msg := fmt.Sprintf("cannot store message event [%s] for message with ID [%s]", storeParams.EventName, storeParams.MessageID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onMessageNotificationSent handles the events.EventTypeMessageNotificationSent event
func (listener *MessageListener) onMessageNotificationSent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageNotificationSentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	checkParams := services.MessageScheduleExpirationParams{
		MessageID:                 payload.MessageID,
		UserID:                    payload.UserID,
		NotificationSentAt:        payload.NotificationSentAt,
		PhoneID:                   payload.PhoneID,
		Source:                    event.Source(),
		MessageExpirationDuration: payload.MessageExpirationDuration,
	}
	if err := listener.service.ScheduleExpirationCheck(ctx, checkParams); err != nil {
		msg := fmt.Sprintf("cannot exchedule expiration check for  ID [%s] and userID [%s]", checkParams.MessageID, checkParams.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	handleParams := services.HandleMessageParams{
		ID:        payload.MessageID,
		UserID:    payload.UserID,
		Source:    event.Source(),
		Timestamp: payload.NotificationSentAt,
	}
	if err := listener.service.HandleMessageNotificationSent(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle event [%s] for message [%s] and userID [%s]", event.Type(), checkParams.MessageID, checkParams.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onMessageSendExpiredCheck handles the events.EventTypeMessageSendExpiredCheck event
func (listener *MessageListener) onMessageSendExpiredCheck(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageSendExpiredCheckPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	checkParams := services.MessageCheckExpired{
		MessageID: payload.MessageID,
		UserID:    payload.UserID,
		Source:    event.Source(),
	}
	if err := listener.service.CheckExpired(ctx, checkParams); err != nil {
		msg := fmt.Sprintf("cannot check expiration for  ID [%s] and userID [%s]", checkParams.MessageID, checkParams.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onMessageSendExpired handles the events.EventTypeMessageSendExpired event
func (listener *MessageListener) onMessageSendExpired(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageSendExpiredPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	expiredParams := services.HandleMessageParams{
		ID:        payload.MessageID,
		UserID:    payload.UserID,
		Source:    event.Source(),
		Timestamp: payload.Timestamp,
	}
	if err := listener.service.HandleMessageExpired(ctx, expiredParams); err != nil {
		msg := fmt.Sprintf("cannot handle event [%s] for ID [%s] and userID [%s]", event.Type(), expiredParams.ID, expiredParams.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onMessageNotificationScheduled handles the events.EventTypeMessageSendExpired event
func (listener *MessageListener) onMessageNotificationScheduled(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageNotificationScheduledPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	expiredParams := services.HandleMessageParams{
		ID:        payload.MessageID,
		UserID:    payload.UserID,
		Source:    event.Source(),
		Timestamp: payload.ScheduledAt,
	}
	if err := listener.service.HandleMessageNotificationScheduled(ctx, expiredParams); err != nil {
		msg := fmt.Sprintf("cannot handle event [%s] for ID [%s] and userID [%s]", event.Type(), expiredParams.ID, expiredParams.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (listener *MessageListener) signature(event cloudevents.Event) string {
	return listener.handlerSignature(listener, event)
}
