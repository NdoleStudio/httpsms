package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
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
	notificationID := uuid.New()
	ttl := 5 * time.Minute
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
		assert.Equal(t, "5m0s", payload.Message.Android.TTL)

		return response(http.StatusNoContent, http.NoBody), nil
	}))

	result, err := sender.Send(context.Background(), "https://adapter.example.com/notify", notification)

	require.NoError(t, err)
	assert.Equal(t, "http/"+notificationID.String(), result)
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

func TestHTTPNotificationSenderConfiguresSecureHTTPClient(t *testing.T) {
	policy := newHTTPNotificationPolicy()
	sender := NewHTTPNotificationSender(nil, nil, &http.Client{Timeout: time.Minute}, policy)

	transport, ok := sender.client.Transport.(*http.Transport)
	require.True(t, ok)
	assert.Zero(t, sender.client.Timeout)
	assert.Nil(t, transport.Proxy)
	assert.NotNil(t, transport.DialContext)
	assert.NotNil(t, sender.client.CheckRedirect)
	require.NotNil(t, transport.TLSClientConfig)
	assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestHTTPNotificationSenderClearsCustomTLSDialersAndServerName(t *testing.T) {
	sender := NewHTTPNotificationSender(nil, nil, &http.Client{
		Transport: &http.Transport{
			DialTLS: func(string, string) (net.Conn, error) {
				return nil, errors.New("unsafe TLS dialer must not be used")
			},
			DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
				return nil, errors.New("unsafe TLS context dialer must not be used")
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         "attacker.example.com",
			},
		},
	}, newHTTPNotificationPolicy())

	transport, ok := sender.client.Transport.(*http.Transport)

	require.True(t, ok)
	assert.Nil(t, transport.DialTLS)
	assert.Nil(t, transport.DialTLSContext)
	require.NotNil(t, transport.DialContext)
	require.NotNil(t, transport.TLSClientConfig)
	assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
	assert.Empty(t, transport.TLSClientConfig.ServerName)
}

func TestHTTPNotificationSenderRejectsOpaqueTransportAndUsesPolicyDialer(t *testing.T) {
	opaque := &wrappedNotificationRoundTripper{}
	policy := NewNotificationEndpointPolicy(&staticHostResolver{
		addresses: map[string][]netip.Addr{
			"private.example.com": {netip.MustParseAddr("127.0.0.1")},
		},
	}, nil)
	sender := NewHTTPNotificationSender(nil, nil, &http.Client{
		Transport: opaque,
	}, policy)

	transport, ok := sender.client.Transport.(*http.Transport)
	require.True(t, ok)

	_, err := transport.DialContext(context.Background(), "tcp", "private.example.com:443")

	require.Error(t, err)
	assert.Zero(t, opaque.calls)
}

func TestHTTPNotificationSenderTrustsServiceCreatedWrapperWithSecuredParent(t *testing.T) {
	policy := NewNotificationEndpointPolicy(&staticHostResolver{
		addresses: map[string][]netip.Addr{
			"private.example.com": {netip.MustParseAddr("127.0.0.1")},
		},
	}, nil)
	unsafeDialCalls := 0
	var parent http.RoundTripper
	transport := NewNotificationHTTPTransport(
		policy,
		&http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(context.Context, string, string) (net.Conn, error) {
				unsafeDialCalls++
				return nil, errors.New("unsafe dialer must not be used")
			},
			DialTLS: func(string, string) (net.Conn, error) {
				return nil, errors.New("unsafe TLS dialer must not be used")
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS10,
				ServerName:         "attacker.example.com",
			},
		},
		&net.Dialer{Timeout: time.Second},
		func(securedParent http.RoundTripper) http.RoundTripper {
			parent = securedParent
			return &wrappedNotificationRoundTripper{}
		},
	)

	sender := NewHTTPNotificationSender(nil, nil, &http.Client{Transport: transport}, policy)

	assert.Same(t, transport, sender.client.Transport)
	securedParent, ok := parent.(*http.Transport)
	require.True(t, ok)
	assert.Nil(t, securedParent.Proxy)
	assert.NotNil(t, securedParent.DialContext)
	assert.Nil(t, securedParent.DialTLS)
	assert.Nil(t, securedParent.DialTLSContext)
	require.NotNil(t, securedParent.TLSClientConfig)
	assert.False(t, securedParent.TLSClientConfig.InsecureSkipVerify)
	assert.Empty(t, securedParent.TLSClientConfig.ServerName)
	assert.Equal(t, uint16(tls.VersionTLS12), securedParent.TLSClientConfig.MinVersion)

	_, err := securedParent.DialContext(context.Background(), "tcp", "private.example.com:443")

	require.Error(t, err)
	assert.Zero(t, unsafeDialCalls)
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

type wrappedNotificationRoundTripper struct {
	calls int
}

func (transport *wrappedNotificationRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	transport.calls++
	return nil, errors.New("must not be used")
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
	return &HTTPNotificationSender{
		logger:     logger,
		client:     &http.Client{Transport: transport},
		policy:     newHTTPNotificationPolicy(),
		attempts:   3,
		timeout:    5 * time.Second,
		retryDelay: func(context.Context, time.Duration) error { return nil },
	}
}

func newHTTPNotificationPolicy() *NotificationEndpointPolicy {
	return NewNotificationEndpointPolicy(&staticHostResolver{
		addresses: map[string][]netip.Addr{
			"adapter.example.com": {netip.MustParseAddr("8.8.8.8")},
		},
	}, nil)
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
