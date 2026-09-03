package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	ttl := 10 * time.Minute
	message := &messaging.Message{
		Token: "https://adapter.example.com/notify",
		Data:  map[string]string{"KEY_MESSAGE_ID": "message-1"},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			TTL:      &ttl,
		},
	}
	sender := newHTTPNotificationSender(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))

		var payload httpNotificationPayload
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		assert.Equal(t, "https://adapter.example.com/notify", request.URL.String())
		assert.Equal(t, "https://adapter.example.com/notify", payload.Message.Token)
		assert.Equal(t, map[string]string{"KEY_MESSAGE_ID": "message-1"}, payload.Message.Data)
		assert.Equal(t, "high", payload.Message.Android.Priority)
		assert.Equal(t, "600s", payload.Message.Android.TTL)

		return response(http.StatusNoContent, http.NoBody), nil
	}))

	result, err := sender.Send(context.Background(), message)

	require.NoError(t, err)
	assert.Equal(t, "http/success", result)
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
			calls := 0
			sender := newHTTPNotificationSender(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				outcome := test.outcomes[calls]
				calls++
				if outcome.err != nil {
					return nil, outcome.err
				}
				return response(outcome.statusCode, http.NoBody), nil
			}))
			result, err := sender.Send(
				context.Background(),
				&messaging.Message{Token: "https://adapter.example.com/notify"},
			)

			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.wantCalls, calls)
			if !test.wantErr {
				assert.Equal(t, "http/success", result)
			}
		})
	}
}

func TestHTTPNotificationSenderReusesRetrierAcrossSends(t *testing.T) {
	calls := 0
	sender := newHTTPNotificationSender(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		calls++
		if calls%2 == 1 {
			return response(http.StatusServiceUnavailable, http.NoBody), nil
		}
		return response(http.StatusNoContent, http.NoBody), nil
	}))

	for range 2 {
		_, err := sender.Send(
			context.Background(),
			&messaging.Message{Token: "https://adapter.example.com/notify"},
		)
		require.NoError(t, err)
	}

	assert.Equal(t, 4, calls)
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

	_, err := sender.Send(
		context.Background(),
		&messaging.Message{
			Token: "https://adapter.example.com/notify",
			Data:  map[string]string{"KEY_MESSAGE_ID": "message-1"},
		},
	)

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

	_, err := sender.Send(
		context.Background(),
		&messaging.Message{Token: "https://adapter.example.com/notify"},
	)

	require.NoError(t, err)
	assert.Equal(t, int64(4096), body.read)
	assert.True(t, body.closed)
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

	_, err := sender.Send(
		context.Background(),
		&messaging.Message{
			Token: "https://adapter.example.com/notify",
			Data:  map[string]string{"KEY_HEARTBEAT_ID": "heartbeat-1"},
			Android: &messaging.AndroidConfig{
				Priority: "high",
			},
		},
	)

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

	sender := NewHTTPNotificationSender(nil, client)

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
		&messaging.Message{Token: endpoint.String()},
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

	_, err := sender.Send(
		context.Background(),
		&messaging.Message{Token: "https://adapter.example.com/notify"},
	)

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

	_, err := sender.Send(
		ctx,
		&messaging.Message{Token: "https://adapter.example.com/notify"},
	)

	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestHTTPNotificationSenderRejectsNilMessage(t *testing.T) {
	sender := newHTTPNotificationSender(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return response(http.StatusNoContent, http.NoBody), nil
	}))

	_, err := sender.Send(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification message is nil")
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
	return newHTTPNotificationSenderWithRetrier(
		logger,
		&http.Client{Transport: transport},
		newHTTPNotificationRetrier(0),
	)
}

func response(statusCode int, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       body,
		Header:     make(http.Header),
	}
}
