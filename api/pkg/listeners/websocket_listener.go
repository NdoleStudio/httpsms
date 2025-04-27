package listeners

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/pusher/pusher-http-go/v5"
)

// WebsocketListener handles cloud events that send a websocket event to the frontend
type WebsocketListener struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	client *pusher.Client
}

// NewWebsocketListener creates a new instance of WebsocketListener
func NewWebsocketListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *pusher.Client,
) (l *WebsocketListener, routes map[string]events.EventListener) {
	l = &WebsocketListener{
		logger: logger.WithService(fmt.Sprintf("%T", l)),
		tracer: tracer,
		client: client,
	}

	return l, map[string]events.EventListener{
		events.EventTypePhoneUpdated:      l.onPhoneUpdated,
		events.EventTypeMessagePhoneSent:  l.onMessagePhoneSent,
		events.EventTypeMessageSendFailed: l.onMessagePhoneFailed,
	}
}

// onMessagePhoneSent handles the events.EventTypeMessagePhoneSent event
func (listener *WebsocketListener) onMessagePhoneSent(ctx context.Context, event cloudevents.Event) error {
	ctx, span, _ := listener.tracer.StartWithLogger(ctx, listener.logger)
	defer span.End()

	var payload events.MessagePhoneSentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.client.Trigger(payload.UserID.String(), event.Type(), event.ID()); err != nil {
		msg := fmt.Sprintf("cannot trigger websocket [%s] event with ID [%s] for user with ID [%s]", event.Type(), event.ID(), payload.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onMessagePhoneFailed handles the events.EventTypeMessageSendFailed event
func (listener *WebsocketListener) onMessagePhoneFailed(ctx context.Context, event cloudevents.Event) error {
	ctx, span, _ := listener.tracer.StartWithLogger(ctx, listener.logger)
	defer span.End()

	var payload events.MessageSendFailedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.client.Trigger(payload.UserID.String(), event.Type(), event.ID()); err != nil {
		msg := fmt.Sprintf("cannot trigger websocket [%s] event with ID [%s] for user with ID [%s]", event.Type(), event.ID(), payload.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onPhoneUpdated handles the events.EventTypePhoneUpdated event
func (listener *WebsocketListener) onPhoneUpdated(ctx context.Context, event cloudevents.Event) error {
	ctx, span, _ := listener.tracer.StartWithLogger(ctx, listener.logger)
	defer span.End()

	var payload events.PhoneUpdatedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.client.Trigger(payload.UserID.String(), event.Type(), event.ID()); err != nil {
		msg := fmt.Sprintf("cannot trigger websocket [%s] event with ID [%s] for user with ID [%s]", event.Type(), event.ID(), payload.UserID)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
