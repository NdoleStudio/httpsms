package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
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
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	client     *http.Client
	policy     *NotificationEndpointPolicy
	attempts   uint
	timeout    time.Duration
	retryDelay func(context.Context, time.Duration) error
}

// NewHTTPNotificationSender creates an SSRF-safe HTTP notification sender.
func NewHTTPNotificationSender(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *http.Client,
	policy *NotificationEndpointPolicy,
) *HTTPNotificationSender {
	return &HTTPNotificationSender{
		logger:   logger,
		tracer:   tracer,
		client:   newNotificationHTTPClient(client, policy),
		policy:   policy,
		attempts: 3,
		timeout:  5 * time.Second,
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
	if _, err = sender.policy.Validate(ctx, endpoint); err != nil {
		return "", sender.notificationError(hostname, "cannot validate notification endpoint")
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
		payload.Message.Android.TTL = notification.TTL.String()
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
		request, requestErr := http.NewRequestWithContext(
			requestCtx,
			http.MethodPost,
			endpoint.String(),
			bytes.NewReader(body),
		)
		if requestErr == nil {
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-httpSMS-Notification-ID", notification.NotificationID.String())
			requestErr = sender.sendAttempt(request)
		}
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

	transport, ok := configured.Transport.(*http.Transport)
	if !ok {
		transport = http.DefaultTransport.(*http.Transport)
	}
	transport = transport.Clone()
	transport.Proxy = nil
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.InsecureSkipVerify = false
	if policy != nil {
		transport.DialContext = policy.DialContext(&net.Dialer{})
	}
	configured.Transport = transport

	return &configured
}

func isRetryableNotificationStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
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
	_, isRetryableStatus := err.(retryableNotificationStatusError)
	return !isTerminalNotificationStatusError(err) && (isRetryableStatus || err != nil)
}

func isTerminalNotificationStatusError(err error) bool {
	_, ok := err.(terminalNotificationStatusError)
	return ok
}
