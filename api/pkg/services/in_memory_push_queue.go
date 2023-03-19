package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

type inMemoryPushQueue struct {
	getEventDispatcher func() *EventDispatcher
	queueConfig        PushQueueConfig
	logger             telemetry.Logger
	tracer             telemetry.Tracer
}

// NewGooglePushQueue creates a new googlePushQueue
func NewInMemoryPushQueue(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	queueConfig PushQueueConfig,
	getEventDispatcher func() *EventDispatcher,
) PushQueue {
	return &inMemoryPushQueue{
		tracer:             tracer,
		logger:             logger,
		queueConfig:        queueConfig,
		getEventDispatcher: getEventDispatcher,
	}
}

// Enqueue a task to the queue
func (queue *inMemoryPushQueue) Enqueue(ctx context.Context, task *PushQueueTask, timeout time.Duration) (queueID string, err error) {
	ctx, span := queue.tracer.Start(ctx)
	ctxLogger := queue.tracer.CtxLogger(queue.logger, span)
	queueID = uuid.New().String()

	go func() {
		time.Sleep(timeout)
		var event cloudevents.Event
		json.Unmarshal(task.Body, &event)
		queue.getEventDispatcher().DispatchSync(ctx, event)
	}()

	ctxLogger.Info(fmt.Sprintf(
		"item added to [%s] queue with id [%s] and schedule [%s]",
		queue.queueConfig.Name,
		queueID,
		time.Now().UTC().Add(timeout),
	))

	return queueID, nil
}
