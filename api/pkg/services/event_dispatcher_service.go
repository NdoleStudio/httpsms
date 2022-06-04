package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/NdoleStudio/http-sms-manager/pkg/events"
	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"
)

// EventDispatcher dispatches a new event
type EventDispatcher struct {
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.EventRepository
	listeners  map[string][]events.EventListener
}

// NewEventDispatcher creates a new EventDispatcher
func NewEventDispatcher(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.EventRepository,
) (dispatcher *EventDispatcher) {
	return &EventDispatcher{
		logger:     logger,
		tracer:     tracer,
		listeners:  make(map[string][]events.EventListener),
		repository: repository,
	}
}

// Dispatch dispatches a new event
func (dispatcher *EventDispatcher) Dispatch(ctx context.Context, event cloudevents.Event) error {
	ctx, span := dispatcher.tracer.Start(ctx)
	defer span.End()

	if err := dispatcher.repository.Save(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot save event with ID [%s] and type [%s]", event.ID(), event.Type())
		return dispatcher.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	dispatcher.publish(ctx, event)
	return nil
}

// Subscribe a listener to an event
func (dispatcher *EventDispatcher) Subscribe(eventType string, listener events.EventListener) {
	if _, ok := dispatcher.listeners[eventType]; !ok {
		dispatcher.listeners[eventType] = []events.EventListener{}
	}

	// remove duplicates
	for _, existing := range dispatcher.listeners[eventType] {
		if fmt.Sprintf("%T", existing) == fmt.Sprintf("%T", listener) {
			return
		}
	}

	dispatcher.listeners[eventType] = append(dispatcher.listeners[eventType], listener)
}

func (dispatcher *EventDispatcher) publish(ctx context.Context, event cloudevents.Event) {
	ctx, span := dispatcher.tracer.Start(ctx)
	defer span.End()

	ctxLogger := dispatcher.tracer.CtxLogger(dispatcher.logger, span)

	subscribers, ok := dispatcher.listeners[event.Type()]
	if !ok {
		ctxLogger.Info(fmt.Sprintf("no listener is configured for event type [%s]", event.Type()))
		return
	}

	var wg sync.WaitGroup
	for _, sub := range subscribers {
		wg.Add(1)
		go func(ctx context.Context, sub events.EventListener) {
			if err := sub(ctx, event); err != nil {
				msg := fmt.Sprintf("subscriber [%T] cannot handle event [%s]", sub, event.Type())
				ctxLogger.Error(stacktrace.Propagate(err, msg))
			}
			wg.Done()
		}(ctx, sub)
	}

	wg.Wait()
}
