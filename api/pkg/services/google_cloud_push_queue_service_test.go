package services

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type heartbeatCloudTaskServer struct {
	cloudtaskspb.UnimplementedCloudTasksServer
	mu           sync.Mutex
	tasks        []*cloudtaskspb.Task
	rejectChecks bool
}

func (server *heartbeatCloudTaskServer) CreateTask(_ context.Context, request *cloudtaskspb.CreateTaskRequest) (*cloudtaskspb.Task, error) {
	server.mu.Lock()
	defer server.mu.Unlock()

	var event cloudevents.Event
	if err := json.Unmarshal(request.Task.GetHttpRequest().Body, &event); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if server.rejectChecks && event.Type() == events.EventTypePhoneHeartbeatCheck {
		return nil, status.Error(codes.Unavailable, "heartbeat queue unavailable")
	}
	for _, existing := range server.tasks {
		if existing.Name == request.Task.Name {
			return nil, status.Error(codes.AlreadyExists, "task already exists")
		}
	}
	task := proto.Clone(request.Task).(*cloudtaskspb.Task)
	if task.Name == "" {
		task.Name = request.Parent + "/tasks/" + uuid.NewString()
	}
	server.tasks = append(server.tasks, task)
	return task, nil
}

func (server *heartbeatCloudTaskServer) queuedEvents(t *testing.T, eventType string) []cloudevents.Event {
	t.Helper()
	server.mu.Lock()
	defer server.mu.Unlock()

	var result []cloudevents.Event
	for _, task := range server.tasks {
		var event cloudevents.Event
		require.NoError(t, json.Unmarshal(task.GetHttpRequest().Body, &event))
		if event.Type() == eventType {
			result = append(result, event)
		}
	}
	return result
}

func newHeartbeatCloudTaskQueue(t *testing.T, logger telemetry.Logger, tracer telemetry.Tracer) (PushQueue, *heartbeatCloudTaskServer) {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := &heartbeatCloudTaskServer{}
	rpcServer := grpc.NewServer()
	cloudtaskspb.RegisterCloudTasksServer(rpcServer, server)
	go func() { _ = rpcServer.Serve(listener) }()
	connection, err := grpc.NewClient("passthrough:///cloudtasks",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	client, err := cloudtasks.NewClient(context.Background(), option.WithGRPCConn(connection))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Close()
		_ = connection.Close()
		rpcServer.Stop()
		_ = listener.Close()
	})
	return NewGooglePushQueue(logger, tracer, client, PushQueueConfig{
		Name: "projects/test/locations/test/queues/heartbeats",
	}), server
}

func TestGooglePushQueue_DeduplicatesOfflineEventsWithoutDeduplicatingOtherEvents(t *testing.T) {
	for _, eventType := range []string{events.EventTypePhoneHeartbeatOffline, events.EventTypePhoneHeartbeatOnline} {
		t.Run(eventType, func(t *testing.T) {
			logger := &phoneNotificationLogger{}
			tracer := telemetry.NewOtelLogger("test", logger)
			queue, server := newHeartbeatCloudTaskQueue(t, logger, tracer)
			dispatcher := NewEventDispatcher(logger, tracer, nil, queue, PushQueueConfig{
				ConsumerEndpoint: "https://example.com/events",
			})
			event := cloudevents.NewEvent()
			event.SetID(uuid.NewString())
			event.SetType(eventType)
			event.SetSource("https://example.com")
			require.NoError(t, dispatcher.Dispatch(context.Background(), event))
			require.NoError(t, dispatcher.Dispatch(context.Background(), event))
			want := 2
			if eventType == events.EventTypePhoneHeartbeatOffline {
				want = 1
			}
			require.Len(t, server.queuedEvents(t, eventType), want)
		})
	}
}
