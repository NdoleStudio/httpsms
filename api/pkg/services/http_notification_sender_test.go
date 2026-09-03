package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

type httpNotificationPayload struct {
	Message struct {
		Token   string            `json:"token"`
		Data    map[string]string `json:"data"`
		Android struct {
			Priority string `json:"priority"`
			TTL      string `json:"ttl,omitempty"`
		} `json:"android"`
	} `json:"message"`
}

func TestHTTPNotificationSenderSendsFCMCompatiblePayload(t *testing.T) {
	notificationID := uuid.New()
	ttl := 10 * time.Minute
	notification := GatewayNotification{
		Data:           map[string]string{"KEY_MESSAGE_ID": "message-1"},
		Priority:       "high",
		TTL:            &ttl,
		NotificationID: notificationID,
	}
	sender := newHTTPNotificationSender(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
		assert.Equal(t, notification.NotificationID.String(), request.Header.Get("X-httpSMS-Notification-ID"))

		var payload httpNotificationPayload
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		assert.Equal(t, "https://adapter.example.com/notify", request.URL.String())
		assert.Equal(t, "https://adapter.example.com/notify", payload.Message.Token)
		assert.Equal(t, map[string]string{"KEY_MESSAGE_ID": "message-1"}, payload.Message.Data)
		assert.Equal(t, "high", payload.Message.Android.Priority)
		assert.Equal(t, "600s", payload.Message.Android.TTL)

		return response(http.StatusNoContent, http.NoBody), nil
	}))

	result, err := sender.Send(context.Background(), "https://adapter.example.com/notify", notification)

	require.NoError(t, err)
	assert.Equal(t, "http/"+notificationID.String(), result)
}

func TestFormatProtobufDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{name: "whole seconds", duration: 10 * time.Minute, expected: "600s"},
		{name: "milliseconds", duration: 1500 * time.Millisecond, expected: "1.500s"},
		{name: "microseconds", duration: time.Second + 234567*time.Microsecond, expected: "1.234567s"},
		{name: "nanoseconds", duration: time.Second + 234567890*time.Nanosecond, expected: "1.234567890s"},
		{name: "negative subsecond", duration: -500 * time.Millisecond, expected: "-0.500s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, formatProtobufDuration(test.duration))
		})
	}
}

func TestHTTPNotificationSenderRetriesOnlyTransientFailures(t *testing.T) {
	tests := []struct {
		name      string
		outcomes  []roundTripOutcome
		wantCalls int
		wantErr   bool
	}{
		{
			name: "network error then accepted",
			outcomes: []roundTripOutcome{
				{err: errors.New("connection reset")},
				{statusCode: http.StatusAccepted},
			},
			wantCalls: 2,
		},
		{
			name: "request timeout then success",
			outcomes: []roundTripOutcome{
				{statusCode: http.StatusRequestTimeout},
				{statusCode: http.StatusOK},
			},
			wantCalls: 2,
		},
		{
			name: "rate limited then no content",
			outcomes: []roundTripOutcome{
				{statusCode: http.StatusTooManyRequests},
				{statusCode: http.StatusNoContent},
			},
			wantCalls: 2,
		},
		{
			name: "server errors then no content",
			outcomes: []roundTripOutcome{
				{statusCode: http.StatusInternalServerError},
				{statusCode: http.StatusBadGateway},
				{statusCode: http.StatusNoContent},
			},
			wantCalls: 3,
		},
		{
			name:      "bad request fails immediately",
			outcomes:  []roundTripOutcome{{statusCode: http.StatusBadRequest}},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name: "three service unavailable responses fail",
			outcomes: []roundTripOutcome{
				{statusCode: http.StatusServiceUnavailable},
				{statusCode: http.StatusServiceUnavailable},
				{statusCode: http.StatusServiceUnavailable},
			},
			wantCalls: 3,
			wantErr:   true,
		},
		{
			name:      "redirect fails immediately",
			outcomes:  []roundTripOutcome{{statusCode: http.StatusFound}},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name:      "nonstandard 6xx response fails immediately",
			outcomes:  []roundTripOutcome{{statusCode: 600}},
			wantCalls: 1,
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var notificationIDs []string
			calls := 0
			sender := newHTTPNotificationSender(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
				notificationIDs = append(notificationIDs, request.Header.Get("X-httpSMS-Notification-ID"))
				outcome := test.outcomes[calls]
				calls++
				if outcome.err != nil {
					return nil, outcome.err
				}
				return response(outcome.statusCode, http.NoBody), nil
			}))
			notificationID := uuid.New()

			_, err := sender.Send(context.Background(), "https://adapter.example.com/notify", GatewayNotification{
				NotificationID: notificationID,
			})

			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.wantCalls, calls)
			assert.Equal(t, makeNotificationIDs(notificationID.String(), test.wantCalls), notificationIDs)
		})
	}
}

func TestHTTPNotificationSenderCreatesFreshRequestAndBodyForEveryAttempt(t *testing.T) {
	var requests []*http.Request
	var bodies [][]byte
	sender := newHTTPNotificationSender(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request)
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		bodies = append(bodies, body)
		if len(requests) < 3 {
			return response(http.StatusServiceUnavailable, http.NoBody), nil
		}
		return response(http.StatusNoContent, http.NoBody), nil
	}))

	_, err := sender.Send(context.Background(), "https://adapter.example.com/notify", GatewayNotification{
		Data:           map[string]string{"KEY_MESSAGE_ID": "message-1"},
		NotificationID: uuid.New(),
	})

	require.NoError(t, err)
	require.Len(t, requests, 3)
	assert.NotSame(t, requests[0], requests[1])
	assert.NotSame(t, requests[1], requests[2])
	require.Len(t, bodies, 3)
	assert.NotEmpty(t, bodies[0])
	assert.Equal(t, bodies[0], bodies[1])
	assert.Equal(t, bodies[1], bodies[2])
}

func TestHTTPNotificationSenderBoundsResponseBodyDiscard(t *testing.T) {
	body := &boundedReadCloser{remaining: 8192}
	sender := newHTTPNotificationSender(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return response(http.StatusNoContent, body), nil
	}))

	_, err := sender.Send(context.Background(), "https://adapter.example.com/notify", GatewayNotification{
		NotificationID: uuid.New(),
	})

	require.NoError(t, err)
	assert.Equal(t, int64(4096), body.read)
	assert.True(t, body.closed)
}

func TestHTTPNotificationSenderRedactsDestinationSecrets(t *testing.T) {
	logger := &httpNotificationRecordingLogger{}
	sender := newHTTPNotificationSenderWithLogger(t, logger, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return response(http.StatusBadRequest, io.NopCloser(bytes.NewBufferString("customer-secret"))), nil
	}))
	destination := "https://adapter.example.com/secret/path?token=customer-secret"

	_, err := sender.Send(context.Background(), destination, GatewayNotification{NotificationID: uuid.New()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "adapter.example.com")
	for _, secret := range []string{"secret/path", "customer-secret", destination} {
		assert.NotContains(t, err.Error(), secret)
		assert.NotContains(t, strings.Join(logger.errors, "\n"), secret)
	}
}

func TestHTTPNotificationSenderOmitsTTLForHeartbeat(t *testing.T) {
	sender := newHTTPNotificationSender(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var payload httpNotificationPayload
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		assert.Equal(t, "high", payload.Message.Android.Priority)
		assert.Empty(t, payload.Message.Android.TTL)
		assert.Equal(t, "heartbeat-1", payload.Message.Data["KEY_HEARTBEAT_ID"])
		return response(http.StatusNoContent, http.NoBody), nil
	}))

	_, err := sender.Send(context.Background(), "https://adapter.example.com/notify", GatewayNotification{
		Data:           map[string]string{"KEY_HEARTBEAT_ID": "heartbeat-1"},
		Priority:       "high",
		NotificationID: uuid.New(),
	})

	require.NoError(t, err)
}

func TestHTTPNotificationSenderUsesInjectedHTTPClientUnchanged(t *testing.T) {
	transport := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return response(http.StatusNoContent, http.NoBody), nil
	})
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Minute,
	}

	sender := NewHTTPNotificationSender(nil, nil, client)

	assert.Same(t, client, sender.client)
	assert.Equal(t, reflect.ValueOf(transport).Pointer(), reflect.ValueOf(sender.client.Transport).Pointer())
	assert.Equal(t, time.Minute, sender.client.Timeout)
}

func TestHTTPNotificationSenderAllowsEndpointUserInformation(t *testing.T) {
	sender := newHTTPNotificationSender(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		username, password, ok := request.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "adapter-user", username)
		assert.Equal(t, "adapter-password", password)
		return response(http.StatusNoContent, http.NoBody), nil
	}))
	endpoint := &url.URL{
		Scheme: "https",
		User:   url.UserPassword("adapter-user", "adapter-password"),
		Host:   "adapter.example.com",
		Path:   "/notify",
	}

	_, err := sender.Send(
		context.Background(),
		endpoint.String(),
		GatewayNotification{NotificationID: uuid.New()},
	)

	require.NoError(t, err)
}

func TestHTTPNotificationSenderBoundsEveryAttemptByTimeout(t *testing.T) {
	calls := 0
	sender := newHTTPNotificationSender(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		<-request.Context().Done()
		return nil, request.Context().Err()
	}))
	sender.timeout = 10 * time.Millisecond

	_, err := sender.Send(context.Background(), "https://adapter.example.com/notify", GatewayNotification{
		NotificationID: uuid.New(),
	})

	require.Error(t, err)
	assert.Equal(t, 3, calls)
}

func TestHTTPNotificationSenderStopsRetriesWhenParentContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	sender := newHTTPNotificationSender(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		cancel()
		<-request.Context().Done()
		return nil, request.Context().Err()
	}))

	_, err := sender.Send(ctx, "https://adapter.example.com/notify", GatewayNotification{
		NotificationID: uuid.New(),
	})

	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestHTTPNotificationSenderTelemetryDoesNotExportCallbackURL(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(context.Background()))
	})
	ctx, parent := provider.Tracer("test").Start(context.Background(), "parent")
	logger := &httpNotificationRecordingLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	sender := newHTTPNotificationSender(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return response(http.StatusNoContent, http.NoBody), nil
	}))
	sender.attemptRecorder = newNotificationHTTPAttemptRecorder(tracer)
	destination := "https://adapter.example.com/secret/path?token=customer-secret"

	_, err := sender.Send(ctx, destination, GatewayNotification{NotificationID: uuid.New()})
	parent.End()

	require.NoError(t, err)
	var exported string
	for _, span := range spanRecorder.Ended() {
		exported += span.Name() + span.Status().Description
		for _, attribute := range span.Attributes() {
			exported += string(attribute.Key) + attribute.Value.Emit()
		}
		for _, event := range span.Events() {
			exported += event.Name
			for _, attribute := range event.Attributes {
				exported += string(attribute.Key) + attribute.Value.Emit()
			}
		}
	}
	assert.Contains(t, exported, "notification.transporthttp")
	assert.Contains(t, exported, "notification.status_class2xx")
	for _, secret := range []string{destination, "secret/path", "customer-secret"} {
		assert.NotContains(t, exported, secret)
	}
}

type roundTripOutcome struct {
	statusCode int
	err        error
}

type boundedReadCloser struct {
	remaining int64
	read      int64
	closed    bool
}

func (reader *boundedReadCloser) Read(buffer []byte) (int, error) {
	if reader.remaining == 0 {
		return 0, io.EOF
	}
	read := int64(len(buffer))
	if read > reader.remaining {
		read = reader.remaining
	}
	reader.remaining -= read
	reader.read += read
	return int(read), nil
}

func (reader *boundedReadCloser) Close() error {
	reader.closed = true
	return nil
}

type httpNotificationRecordingLogger struct {
	errors []string
}

func (logger *httpNotificationRecordingLogger) Error(err error) {
	logger.errors = append(logger.errors, err.Error())
}

func (logger *httpNotificationRecordingLogger) WithService(string) telemetry.Logger { return logger }

func (logger *httpNotificationRecordingLogger) WithString(string, string) telemetry.Logger {
	return logger
}

func (logger *httpNotificationRecordingLogger) WithSpan(trace.SpanContext) telemetry.Logger {
	return logger
}

func (logger *httpNotificationRecordingLogger) Trace(string)                  {}
func (logger *httpNotificationRecordingLogger) Info(string)                   {}
func (logger *httpNotificationRecordingLogger) Warn(error)                    {}
func (logger *httpNotificationRecordingLogger) Debug(string)                  {}
func (logger *httpNotificationRecordingLogger) Fatal(error)                   {}
func (logger *httpNotificationRecordingLogger) Printf(string, ...interface{}) {}

func newHTTPNotificationSender(t *testing.T, transport roundTripFunc) *HTTPNotificationSender {
	t.Helper()
	return newHTTPNotificationSenderWithLogger(t, nil, transport)
}

func newHTTPNotificationSenderWithLogger(
	t *testing.T,
	logger telemetry.Logger,
	transport roundTripFunc,
) *HTTPNotificationSender {
	t.Helper()
	sender := NewHTTPNotificationSender(logger, nil, &http.Client{Transport: transport})
	sender.retryDelay = 0
	return sender
}

func response(statusCode int, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       body,
		Header:     make(http.Header),
	}
}

func makeNotificationIDs(notificationID string, length int) []string {
	notificationIDs := make([]string, length)
	for index := range notificationIDs {
		notificationIDs[index] = notificationID
	}
	return notificationIDs
}
