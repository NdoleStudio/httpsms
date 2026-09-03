package di

import (
	"context"
	"net/http"
	"net/url"

	"github.com/NdoleStudio/go-otelroundtripper"
	"go.opentelemetry.io/otel"
)

type notificationOriginalRequestContextKey struct{}

// notificationTelemetryRoundTripper gives telemetry a sanitized request while preserving delivery semantics.
type notificationTelemetryRoundTripper struct {
	telemetry http.RoundTripper
}

func (roundTripper *notificationTelemetryRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil {
		return roundTripper.telemetry.RoundTrip(request)
	}

	ctx := context.WithValue(request.Context(), notificationOriginalRequestContextKey{}, request)
	telemetryRequest := request.Clone(ctx)
	telemetryRequest.URL = notificationTelemetryURL(request.URL)

	return roundTripper.telemetry.RoundTrip(telemetryRequest)
}

type notificationOriginalRequestRoundTripper struct {
	parent http.RoundTripper
}

// RoundTrip restores the original request before the standard transport sends it.
func (roundTripper *notificationOriginalRequestRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	originalRequest, ok := request.Context().Value(notificationOriginalRequestContextKey{}).(*http.Request)
	if !ok {
		originalRequest = request
	}

	return roundTripper.parent.RoundTrip(originalRequest)
}

func (container *Container) notificationHTTPRoundTripper(parent http.RoundTripper) http.RoundTripper {
	originalRequestRoundTripper := &notificationOriginalRequestRoundTripper{parent: parent}

	return &notificationTelemetryRoundTripper{
		telemetry: otelroundtripper.New(
			otelroundtripper.WithName("phone_notification_http"),
			otelroundtripper.WithParent(originalRequestRoundTripper),
			otelroundtripper.WithMeter(otel.GetMeterProvider().Meter(container.projectID)),
		),
	}
}

func notificationTelemetryURL(requestURL *url.URL) *url.URL {
	return &url.URL{
		Scheme: requestURL.Scheme,
		Host:   requestURL.Host,
	}
}
