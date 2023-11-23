package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/avast/retry-go"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type googlePushQueue struct {
	queueConfig PushQueueConfig
	logger      telemetry.Logger
	tracer      telemetry.Tracer
	client      *cloudtasks.Client
}

// NewGooglePushQueue creates a new googlePushQueue
func NewGooglePushQueue(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *cloudtasks.Client,
	queueConfig PushQueueConfig,
) PushQueue {
	return &googlePushQueue{
		client:      client,
		tracer:      tracer,
		logger:      logger,
		queueConfig: queueConfig,
	}
}

// Enqueue a task to the queue
func (queue *googlePushQueue) Enqueue(ctx context.Context, task *PushQueueTask, timeout time.Duration) (queueID string, err error) {
	err = retry.Do(func() error {
		queueID, err = queue.enqueueImpl(ctx, task, timeout)
		return err
	}, retry.Attempts(3))
	return queueID, err
}

// enqueueImpl a task to the queue
func (queue *googlePushQueue) enqueueImpl(ctx context.Context, task *PushQueueTask, timeout time.Duration) (queueID string, err error) {
	ctx, span := queue.tracer.Start(ctx)
	defer span.End()

	ctxLogger := queue.tracer.CtxLogger(queue.logger, span)

	headers := map[string]string{"Content-Type": "application/json"}
	for key, value := range task.Headers {
		headers[key] = value
	}

	// Build the Task payload.
	req := &cloudtaskspb.CreateTaskRequest{
		Parent: queue.queueConfig.Name,
		Task: &cloudtaskspb.Task{
			MessageType: &cloudtaskspb.Task_HttpRequest{
				HttpRequest: &cloudtaskspb.HttpRequest{
					Headers:    headers,
					HttpMethod: queue.httpMethodToProtoHTTPMethod(task.Method),
					Url:        task.URL,
				},
			},
			ScheduleTime: timestamppb.New(time.Now().UTC().Add(timeout)),
		},
	}

	// Add a payload message if one is present.
	req.Task.GetHttpRequest().Body = task.Body

	requestCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	queueTask, err := queue.client.CreateTask(requestCtx, req)
	if err != nil {
		msg := fmt.Sprintf("cannot schedule task [%s] to URL [%s]", string(task.Body), task.URL)
		return queueID, queue.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf(
		"item added to [%s] queue with id [%s] and schedule [%s]",
		queue.queueConfig.Name,
		queueTask.Name,
		queueTask.GetScheduleTime().AsTime(),
	))

	return queueTask.Name, nil
}

func (queue *googlePushQueue) httpMethodToProtoHTTPMethod(httpMethod string) cloudtaskspb.HttpMethod {
	method, ok := map[string]cloudtaskspb.HttpMethod{
		http.MethodGet:  cloudtaskspb.HttpMethod_GET,
		http.MethodPost: cloudtaskspb.HttpMethod_POST,
		http.MethodPut:  cloudtaskspb.HttpMethod_PUT,
	}[httpMethod]

	if !ok {
		return cloudtaskspb.HttpMethod_POST
	}

	return method
}
