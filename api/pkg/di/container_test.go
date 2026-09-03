package di

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNotificationHTTPClientUsesOTelRoundTripperWithoutRetries(t *testing.T) {
	t.Setenv("ENV", "local")
	client := NewLiteContainer().NotificationHTTPClient()

	assert.Zero(t, client.Timeout)
	transport, ok := client.Transport.(*notificationTelemetryRoundTripper)
	require.True(t, ok)
	assert.Equal(t, "*otelroundtripper.otelRoundTripper", reflect.TypeOf(transport.telemetry).String())
	assert.Nil(t, client.CheckRedirect)
}

func TestPhoneNotificationDispatcherInjectsNotificationHTTPClient(t *testing.T) {
	t.Setenv("ENV", "local")
	t.Setenv("FCM_ENDPOINT", "http://localhost")

	dispatcher := NewLiteContainer().PhoneNotificationDispatcher()
	httpSender := reflect.ValueOf(dispatcher).Elem().FieldByName("httpSender").Elem().Elem()
	client := httpSender.FieldByName("client").Elem()
	transport := client.FieldByName("Transport").Elem()

	assert.Equal(t, "*di.notificationTelemetryRoundTripper", transport.Type().String())
	attemptRecorder := httpSender.FieldByName("attemptRecorder").Elem()
	assert.Equal(t, "*services.otelNotificationHTTPAttemptRecorder", attemptRecorder.Type().String())
}

func TestNotificationHTTPRoundTripperSanitizesTelemetryURLOnly(t *testing.T) {
	t.Setenv("ENV", "local")
	previousMeterProvider := otel.GetMeterProvider()
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(meterProvider)
	t.Cleanup(func() {
		otel.SetMeterProvider(previousMeterProvider)
		require.NoError(t, meterProvider.Shutdown(context.Background()))
	})

	var parentRequest *http.Request
	var parentBody string
	parent := notificationRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		parentRequest = request
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		parentBody = string(body)
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       http.NoBody,
			Header:     make(http.Header),
			Request:    request,
		}, nil
	})
	client := &http.Client{
		Transport: NewLiteContainer().notificationHTTPRoundTripper(parent),
	}
	endpoint := &url.URL{
		Scheme:   "https",
		User:     url.UserPassword("adapter-user", "adapter-password"),
		Host:     "adapter.example.com:8443",
		Path:     "/secret/path",
		RawQuery: "token=customer-secret",
		Fragment: "private-fragment",
	}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		endpoint.String(),
		bytes.NewBufferString("notification-body"),
	)
	require.NoError(t, err)
	request.Header.Set("X-Test-Header", "test-value")

	response, err := client.Do(request)

	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.NotNil(t, parentRequest)
	assert.Equal(t, endpoint.String(), parentRequest.URL.String())
	assert.Equal(t, "test-value", parentRequest.Header.Get("X-Test-Header"))
	assert.Equal(t, "notification-body", parentBody)
	username, password, hasBasicAuth := parentRequest.BasicAuth()
	assert.True(t, hasBasicAuth)
	assert.Equal(t, "adapter-user", username)
	assert.Equal(t, "adapter-password", password)

	var metrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &metrics))
	var telemetryURLs []string
	for _, scopeMetrics := range metrics.ScopeMetrics {
		for _, measured := range scopeMetrics.Metrics {
			if measured.Name != "phone_notification_http.attempts" {
				continue
			}
			sum, ok := measured.Data.(metricdata.Sum[int64])
			require.True(t, ok)
			for _, point := range sum.DataPoints {
				value, ok := point.Attributes.Value(attribute.Key("http.url"))
				require.True(t, ok)
				telemetryURLs = append(telemetryURLs, value.AsString())
			}
		}
	}
	assert.Equal(t, []string{"https://adapter.example.com:8443"}, telemetryURLs)
}

type notificationRoundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip notificationRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}
