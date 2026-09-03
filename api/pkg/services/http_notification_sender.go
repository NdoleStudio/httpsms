package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/types/known/durationpb"
)

const maxNotificationResponseDiscardBytes = 4 * 1024

type notificationHTTPTransport struct {
	secured *http.Transport
	policy  *NotificationEndpointPolicy
}

func (transport *notificationHTTPTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport.secured.RoundTrip(request)
}

type httpNotificationRequest struct {
	Message httpNotificationMessage `json:"message"`
}

type httpNotificationMessage struct {
	Token   string                  `json:"token"`
	Data    map[string]string       `json:"data,omitempty"`
	Android httpNotificationAndroid `json:"android,omitempty"`
}

type httpNotificationAndroid struct {
	Priority string `json:"priority,omitempty"`
	TTL      string `json:"ttl,omitempty"`
}

// HTTPNotificationSender sends FCM-compatible gateway notifications to HTTPS adapters.
type HTTPNotificationSender struct {
	logger          telemetry.Logger
	tracer          telemetry.Tracer
	client          *http.Client
	policy          *NotificationEndpointPolicy
	attempts        uint
	timeout         time.Duration
	retryDelay      func(context.Context, time.Duration) error
	attemptRecorder notificationHTTPAttemptRecorder
}

// NewHTTPNotificationSender creates an SSRF-safe HTTP notification sender.
func NewHTTPNotificationSender(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *http.Client,
	policy *NotificationEndpointPolicy,
) *HTTPNotificationSender {
	return &HTTPNotificationSender{
		logger:          logger,
		tracer:          tracer,
		client:          newNotificationHTTPClient(client, policy),
		policy:          policy,
		attempts:        3,
		timeout:         5 * time.Second,
		attemptRecorder: newNotificationHTTPAttemptRecorder(tracer),
		retryDelay: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

// NewNotificationHTTPTransport creates a transport that always routes through a policy-secured parent.
func NewNotificationHTTPTransport(
	policy *NotificationEndpointPolicy,
	transport *http.Transport,
	dialer *net.Dialer,
) http.RoundTripper {
	if policy == nil {
		panic("notification endpoint policy is required")
	}
	secured := secureNotificationHTTPTransport(transport, policy, dialer)

	return &notificationHTTPTransport{
		secured: secured,
		policy:  policy,
	}
}

// Send delivers a notification to an HTTPS adapter. A successful response only accepts wake-up delivery.
func (sender *HTTPNotificationSender) Send(
	ctx context.Context,
	destination string,
	notification GatewayNotification,
) (string, error) {
	endpoint, err := url.Parse(destination)
	if err != nil {
		return "", sender.notificationError("", "cannot parse notification endpoint")
	}
	hostname := endpoint.Hostname()
	if sender.policy == nil {
		return "", sender.notificationError(hostname, "notification endpoint policy is required")
	}

	payload := httpNotificationRequest{
		Message: httpNotificationMessage{
			Token: destination,
			Data:  notification.Data,
			Android: httpNotificationAndroid{
				Priority: notification.Priority,
			},
		},
	}
	if notification.TTL != nil {
		payload.Message.Android.TTL = formatProtobufDuration(*notification.TTL)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", sender.notificationError(hostname, "cannot encode notification")
	}

	if sender.attempts == 0 {
		return "", sender.notificationError(hostname, "notification sender has no attempts configured")
	}

	for attempt := uint(1); attempt <= sender.attempts; attempt++ {
		requestCtx, cancel := context.WithTimeout(ctx, sender.timeout)
		attemptCtx := requestCtx
		finishAttempt := func(int, error) {}
		if sender.attemptRecorder != nil {
			attemptCtx, finishAttempt = sender.attemptRecorder.Start(attemptCtx, attempt)
		}

		_, requestErr := sender.policy.Validate(attemptCtx, endpoint)
		statusCode := 0
		if requestErr == nil {
			var request *http.Request
			request, requestErr = http.NewRequestWithContext(
				attemptCtx,
				http.MethodPost,
				endpoint.String(),
				bytes.NewReader(body),
			)
			if requestErr != nil {
				finishAttempt(statusCode, requestErr)
				cancel()
				return "", sender.notificationError(hostname, "cannot create notification request")
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-httpSMS-Notification-ID", notification.NotificationID.String())
			statusCode, requestErr = sender.sendAttempt(request)
		}
		finishAttempt(statusCode, requestErr)
		cancel()

		if requestErr == nil {
			return "http/" + notification.NotificationID.String(), nil
		}
		if ctx.Err() != nil {
			return "", sender.notificationError(hostname, "notification request cancelled")
		}
		if attempt == sender.attempts || !isRetryableNotificationError(requestErr) {
			return "", sender.notificationError(hostname, "notification request failed")
		}
		if sender.retryDelay(ctx, notificationRetryDelay(attempt)) != nil {
			return "", sender.notificationError(hostname, "notification retry cancelled")
		}
	}

	return "", sender.notificationError(hostname, "notification request failed")
}

func (sender *HTTPNotificationSender) sendAttempt(request *http.Request) (int, error) {
	otel.GetTextMapPropagator().Inject(request.Context(), propagation.HeaderCarrier(request.Header))

	response, err := sender.client.Do(request)
	if err != nil {
		return 0, err
	}
	if response.Body != nil {
		_, _ = io.CopyN(io.Discard, response.Body, maxNotificationResponseDiscardBytes)
		_ = response.Body.Close()
	}
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return response.StatusCode, nil
	}
	if isRetryableNotificationStatus(response.StatusCode) {
		err = retryableNotificationStatusError{statusCode: response.StatusCode}
		return response.StatusCode, err
	}
	err = terminalNotificationStatusError{statusCode: response.StatusCode}
	return response.StatusCode, err
}

func (sender *HTTPNotificationSender) notificationError(hostname string, message string) error {
	if hostname == "" {
		hostname = "unknown"
	}
	err := stacktrace.Propagatef(stacktrace.NewErrorf("%s", message), "cannot send notification to [%s]", hostname)
	if sender.logger != nil {
		sender.logger.Error(err)
	}
	return err
}

func newNotificationHTTPClient(client *http.Client, policy *NotificationEndpointPolicy) *http.Client {
	if client == nil {
		client = &http.Client{}
	}

	configured := *client
	configured.Timeout = 0
	configured.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	if trusted, ok := configured.Transport.(*notificationHTTPTransport); ok &&
		policy != nil &&
		trusted.policy == policy &&
		trusted.secured != nil {
		return &configured
	}

	transport, ok := configured.Transport.(*http.Transport)
	if !ok {
		transport = http.DefaultTransport.(*http.Transport)
	}
	configured.Transport = secureNotificationHTTPTransport(transport, policy, &net.Dialer{})

	return &configured
}

func secureNotificationHTTPTransport(
	transport *http.Transport,
	policy *NotificationEndpointPolicy,
	dialer *net.Dialer,
) *http.Transport {
	if transport == nil {
		transport = http.DefaultTransport.(*http.Transport)
	}
	transport = transport.Clone()
	transport.Proxy = nil
	transport.DialTLS = nil
	transport.DialTLSContext = nil
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.InsecureSkipVerify = false
	transport.TLSClientConfig.ServerName = ""
	if transport.TLSClientConfig.MinVersion < tls.VersionTLS12 {
		transport.TLSClientConfig.MinVersion = tls.VersionTLS12
	}
	if transport.TLSClientConfig.MaxVersion != 0 && transport.TLSClientConfig.MaxVersion < tls.VersionTLS12 {
		transport.TLSClientConfig.MaxVersion = tls.VersionTLS12
	}
	if policy != nil {
		if dialer == nil {
			dialer = &net.Dialer{}
		}
		configuredDialer := *dialer
		transport.DialContext = policy.DialContext(&configuredDialer)
	} else {
		transport.DialContext = func(context.Context, string, string) (net.Conn, error) {
			return nil, stacktrace.NewError("notification endpoint policy is required")
		}
	}

	return transport
}

func isRetryableNotificationStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		(statusCode >= http.StatusInternalServerError && statusCode < 600)
}

func notificationRetryDelay(attempt uint) time.Duration {
	return time.Duration(1<<(attempt-1)) * 250 * time.Millisecond
}

type retryableNotificationStatusError struct {
	statusCode int
}

func (error retryableNotificationStatusError) Error() string {
	return http.StatusText(error.statusCode)
}

type terminalNotificationStatusError struct {
	statusCode int
}

func (error terminalNotificationStatusError) Error() string {
	return http.StatusText(error.statusCode)
}

func isRetryableNotificationError(err error) bool {
	if err == nil || isTerminalNotificationStatusError(err) || isNotificationEndpointPolicyViolation(err) {
		return false
	}
	return true
}

func isTerminalNotificationStatusError(err error) bool {
	var statusError terminalNotificationStatusError
	return errors.As(err, &statusError)
}

func formatProtobufDuration(value time.Duration) string {
	duration := durationpb.New(value)
	seconds := duration.Seconds
	nanoseconds := int64(duration.Nanos)
	sign := ""
	if seconds < 0 || nanoseconds < 0 {
		sign = "-"
		seconds = -seconds
		nanoseconds = -nanoseconds
	}

	result := sign + strconv.FormatInt(seconds, 10)
	if nanoseconds == 0 {
		return result + "s"
	}

	fraction := fmt.Sprintf("%09d", nanoseconds)
	switch {
	case nanoseconds%1_000_000 == 0:
		fraction = fraction[:3]
	case nanoseconds%1_000 == 0:
		fraction = fraction[:6]
	}

	return result + "." + fraction + "s"
}

type notificationHTTPAttemptRecorder interface {
	Start(context.Context, uint) (context.Context, func(int, error))
}

type otelNotificationHTTPAttemptRecorder struct {
	tracer          telemetry.Tracer
	attemptCounter  metric.Int64Counter
	durationSeconds metric.Float64Histogram
}

func newNotificationHTTPAttemptRecorder(tracer telemetry.Tracer) notificationHTTPAttemptRecorder {
	if tracer == nil {
		return nil
	}

	meter := otel.GetMeterProvider().Meter("github.com/NdoleStudio/httpsms/pkg/services")
	attemptCounter, _ := meter.Int64Counter("httpsms.notification.http.attempts")
	durationSeconds, _ := meter.Float64Histogram("httpsms.notification.http.attempt.duration")

	return &otelNotificationHTTPAttemptRecorder{
		tracer:          tracer,
		attemptCounter:  attemptCounter,
		durationSeconds: durationSeconds,
	}
}

func (recorder *otelNotificationHTTPAttemptRecorder) Start(
	ctx context.Context,
	attempt uint,
) (context.Context, func(int, error)) {
	ctx, span := recorder.tracer.Start(ctx, "phone_notification_http")
	span.SetAttributes(
		attribute.String("notification.transport", "http"),
		attribute.Int("notification.attempt", int(attempt)),
	)
	startedAt := time.Now()

	return ctx, func(statusCode int, err error) {
		statusClass := notificationHTTPStatusClass(statusCode, err)
		attributes := []attribute.KeyValue{
			attribute.String("notification.transport", "http"),
			attribute.Int("notification.attempt", int(attempt)),
			attribute.String("notification.status_class", statusClass),
		}
		span.SetAttributes(attributes...)
		if err != nil {
			span.SetStatus(codes.Error, "notification HTTP attempt failed")
		} else {
			span.SetStatus(codes.Ok, "")
		}

		options := metric.WithAttributes(attributes...)
		if recorder.attemptCounter != nil {
			recorder.attemptCounter.Add(ctx, 1, options)
		}
		if recorder.durationSeconds != nil {
			recorder.durationSeconds.Record(ctx, time.Since(startedAt).Seconds(), options)
		}
		span.End()
	}
}

func notificationHTTPStatusClass(statusCode int, err error) string {
	if err != nil && statusCode == 0 {
		return "transport_error"
	}
	if statusCode < 100 {
		return "unknown"
	}
	return fmt.Sprintf("%dxx", statusCode/100)
}
