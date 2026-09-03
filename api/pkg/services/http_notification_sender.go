package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/avast/retry-go/v5"
	"github.com/google/uuid"
)

const (
	maxNotificationResponseDiscardBytes = 4 * 1024
	notificationHTTPAttempts            = 3
	notificationHTTPTimeout             = 5 * time.Second
	notificationHTTPRetryDelay          = 250 * time.Millisecond
)

// HTTPNotificationSender sends FCM-compatible gateway notifications to HTTPS adapters.
type HTTPNotificationSender struct {
	logger  telemetry.Logger
	client  *http.Client
	retrier *retry.Retrier
	timeout time.Duration
}

// NewHTTPNotificationSender creates an HTTP notification sender.
func NewHTTPNotificationSender(
	logger telemetry.Logger,
	client *http.Client,
) *HTTPNotificationSender {
	return newHTTPNotificationSenderWithRetrier(
		logger,
		client,
		newHTTPNotificationRetrier(notificationHTTPRetryDelay),
	)
}

func newHTTPNotificationSenderWithRetrier(
	logger telemetry.Logger,
	client *http.Client,
	retrier *retry.Retrier,
) *HTTPNotificationSender {
	return &HTTPNotificationSender{
		logger:  logger,
		client:  client,
		retrier: retrier,
		timeout: notificationHTTPTimeout,
	}
}

// Send delivers a notification to an HTTPS adapter. A successful response only accepts wake-up delivery.
func (sender *HTTPNotificationSender) Send(
	ctx context.Context,
	message *messaging.Message,
	notificationID uuid.UUID,
) (string, error) {
	if message == nil {
		return "", sender.notificationError("", "notification message is nil")
	}

	endpoint, err := url.Parse(message.Token)
	if err != nil {
		return "", sender.notificationError("", "cannot parse notification endpoint")
	}
	hostname := endpoint.Hostname()

	body, err := encodeHTTPNotificationPayload(message)
	if err != nil {
		return "", sender.notificationError(hostname, "cannot encode notification")
	}

	err = sender.retrier.Do(func() error {
		return sender.deliver(ctx, endpoint, body, notificationID.String())
	})
	if err == nil {
		return "http/" + notificationID.String(), nil
	}
	if ctx.Err() != nil {
		return "", sender.notificationError(hostname, "notification request cancelled")
	}

	return "", sender.notificationError(hostname, "notification request failed")
}

func encodeHTTPNotificationPayload(message *messaging.Message) ([]byte, error) {
	return json.Marshal(map[string]any{
		"message": message,
	})
}

func (sender *HTTPNotificationSender) deliver(
	ctx context.Context,
	endpoint *url.URL,
	body []byte,
	notificationID string,
) error {
	if err := ctx.Err(); err != nil {
		return terminalNotificationRequestError{cause: err}
	}

	attemptCtx, cancel := context.WithTimeout(ctx, sender.timeout)
	defer cancel()

	request, err := createHTTPNotificationRequest(attemptCtx, endpoint, body, notificationID)
	if err != nil {
		return terminalNotificationRequestError{cause: err}
	}

	err = sender.sendAttempt(request)
	if ctx.Err() != nil {
		return terminalNotificationRequestError{cause: ctx.Err()}
	}
	return err
}

func createHTTPNotificationRequest(
	ctx context.Context,
	endpoint *url.URL,
	body []byte,
	notificationID string,
) (*http.Request, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-httpSMS-Notification-ID", notificationID)
	return request, nil
}

func (sender *HTTPNotificationSender) sendAttempt(request *http.Request) error {
	response, err := sender.client.Do(request)
	if err != nil {
		return err
	}
	if response.Body != nil {
		_, _ = io.CopyN(io.Discard, response.Body, maxNotificationResponseDiscardBytes)
		_ = response.Body.Close()
	}
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	if isRetryableNotificationStatus(response.StatusCode) {
		return retryableNotificationStatusError{statusCode: response.StatusCode}
	}
	return terminalNotificationStatusError{statusCode: response.StatusCode}
}

func newHTTPNotificationRetrier(delay time.Duration) *retry.Retrier {
	return retry.New(
		retry.Attempts(notificationHTTPAttempts),
		retry.Delay(delay),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.RetryIf(isRetryableNotificationError),
	)
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
