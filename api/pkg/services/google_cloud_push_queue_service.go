package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
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
	ctx, span := queue.tracer.Start(ctx)
	defer span.End()

	ctxLogger := queue.tracer.CtxLogger(queue.logger, span)

	headers := map[string]string{"Content-Type": "application/json"}
	for key, value := range task.Headers {
		headers[key] = value
	}

	// Build the Task payload.
	// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#CreateTaskRequest
	req := &taskspb.CreateTaskRequest{
		Parent: queue.queueConfig.Name,
		Task: &taskspb.Task{
			// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#HttpRequest
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
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

	queueTask, err := queue.client.CreateTask(ctx, req)
	if err != nil {
		msg := fmt.Sprintf("cannot schedule task %s to URL: %s", string(task.Body), task.URL)
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

func (queue *googlePushQueue) httpMethodToProtoHTTPMethod(httpMethod string) taskspb.HttpMethod {
	method, ok := map[string]taskspb.HttpMethod{
		http.MethodGet:  taskspb.HttpMethod_GET,
		http.MethodPost: taskspb.HttpMethod_POST,
		http.MethodPut:  taskspb.HttpMethod_PUT,
	}[httpMethod]

	if !ok {
		return taskspb.HttpMethod_POST
	}

	return method
}
