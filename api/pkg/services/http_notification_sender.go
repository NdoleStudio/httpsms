package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/avast/retry-go/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/types/known/durationpb"
)

const maxNotificationResponseDiscardBytes = 4 * 1024

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
	attempts        uint
	timeout         time.Duration
	retryDelay      time.Duration
	attemptRecorder notificationHTTPAttemptRecorder
}

// NewHTTPNotificationSender creates an HTTP notification sender.
func NewHTTPNotificationSender(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *http.Client,
) *HTTPNotificationSender {
	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPNotificationSender{
		logger:          logger,
		tracer:          tracer,
		client:          client,
		attempts:        3,
		timeout:         5 * time.Second,
		retryDelay:      250 * time.Millisecond,
		attemptRecorder: newNotificationHTTPAttemptRecorder(tracer),
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

	attempt := uint(0)
	err = retry.New(
		retry.Attempts(sender.attempts),
		retry.Delay(sender.retryDelay),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
		retry.RetryIf(isRetryableNotificationError),
	).Do(func() error {
		attempt++
		requestCtx, cancel := context.WithTimeout(ctx, sender.timeout)
		attemptCtx := requestCtx
		finishAttempt := func(int, error) {}
		if sender.attemptRecorder != nil {
			attemptCtx, finishAttempt = sender.attemptRecorder.Start(attemptCtx, attempt)
		}

		request, requestErr := http.NewRequestWithContext(
			attemptCtx,
			http.MethodPost,
			endpoint.String(),
			bytes.NewReader(body),
		)
		if requestErr != nil {
			finishAttempt(0, requestErr)
			cancel()
			return terminalNotificationRequestError{cause: requestErr}
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-httpSMS-Notification-ID", notification.NotificationID.String())

		statusCode, requestErr := sender.sendAttempt(request)
		finishAttempt(statusCode, requestErr)
		cancel()

		if ctx.Err() != nil {
			return terminalNotificationRequestError{cause: ctx.Err()}
		}
		return requestErr
	})
	if err == nil {
		return "http/" + notification.NotificationID.String(), nil
	}
	if ctx.Err() != nil {
		return "", sender.notificationError(hostname, "notification request cancelled")
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

func isRetryableNotificationStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		(statusCode >= http.StatusInternalServerError && statusCode < 600)
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

type terminalNotificationRequestError struct {
	cause error
}

func (notificationError terminalNotificationRequestError) Error() string {
	return notificationError.cause.Error()
}

func (notificationError terminalNotificationRequestError) Unwrap() error {
	return notificationError.cause
}

func isRetryableNotificationError(err error) bool {
	if err == nil || isTerminalNotificationError(err) {
		return false
	}
	return true
}

func isTerminalNotificationError(err error) bool {
	var statusError terminalNotificationStatusError
	if errors.As(err, &statusError) {
		return true
	}

	var requestError terminalNotificationRequestError
	return errors.As(err, &requestError)
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
