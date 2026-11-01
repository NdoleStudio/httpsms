package services

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/NdoleStudio/stacktrace"
	"github.com/carlmjohnson/requests"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
)

type emulatorPushQueue struct {
	config     PushQueueConfig
	client     *http.Client
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	namedTasks sync.Map
}

// EmulatorPushQueue creates a new googlePushQueue
func EmulatorPushQueue(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *http.Client,
	config PushQueueConfig,
) PushQueue {
	return &emulatorPushQueue{
		tracer: tracer,
		logger: logger.WithService(fmt.Sprintf("%T", emulatorPushQueue{})),
		client: client,
		config: config,
	}
}

// Enqueue a task to the queue
func (queue *emulatorPushQueue) Enqueue(ctx context.Context, task *PushQueueTask, timeout time.Duration) (queueID string, err error) {
	ctx, span, ctxLogger := queue.tracer.StartWithLogger(ctx, queue.logger)
	defer span.End()

	queueID = uuid.New().String()
	if task.ID != "" {
		queueID = task.ID
		if _, exists := queue.namedTasks.LoadOrStore(queueID, struct{}{}); exists {
			return queueID, nil
		}
		// Mirror Cloud Tasks' named-task retention in this non-durable local emulator.
		time.AfterFunc(24*time.Hour, func() { queue.namedTasks.Delete(queueID) })
	}

	time.AfterFunc(timeout, queue.push(*task, queueID))

	ctxLogger.Info(fmt.Sprintf(
		"task added to [%s] queue with ID [%s] and scheduled at [%s]",
		queue.config.Name,
		queueID,
		time.Now().UTC().Add(timeout),
	))

	return queueID, nil
}

func (queue *emulatorPushQueue) push(task PushQueueTask, queueID string) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := requests.
			URL(task.URL).
			Client(queue.client).
			Method(task.Method).
			BodyBytes(task.Body)

		// add headers
		for key, value := range task.Headers {
			request.Header(key, value)
		}

		// add content type
		request.Header("Content-Type", "application/json")

		if err := request.Fetch(ctx); err != nil {
			queue.logger.Error(stacktrace.Propagatef(err, "cannot send http request to [%s] for queue task [%s]", task.URL, queueID))
			return
		}

		queue.logger.Info(fmt.Sprintf("queue task [%s] sent to URL [%s]", queueID, task.URL))
	}
}
