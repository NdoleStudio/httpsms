package validators

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// TurnstileTokenValidator validates the token used to validate captchas from cloudflare
type TurnstileTokenValidator struct {
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	secretKey  string
	httpClient *http.Client
}

type turnstileVerifyResponse struct {
	Success     bool      `json:"success"`
	ChallengeTs time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []any     `json:"error-codes"`
	Action      string    `json:"action"`
	Cdata       string    `json:"cdata"`
	Metadata    struct {
		EphemeralID string `json:"ephemeral_id"`
	} `json:"metadata"`
}

// NewTurnstileTokenValidator creates a new TurnstileTokenValidator
func NewTurnstileTokenValidator(logger telemetry.Logger, tracer telemetry.Tracer, secretKey string, httpClient *http.Client) *TurnstileTokenValidator {
	return &TurnstileTokenValidator{
		logger.WithService(fmt.Sprintf("%T", &TurnstileTokenValidator{})),
		tracer,
		secretKey,
		httpClient,
	}
}

// ValidateToken validates the cloudflare turnstile token
// https://developers.cloudflare.com/turnstile/get-started/server-side-validation/
func (v *TurnstileTokenValidator) ValidateToken(ctx context.Context, ipAddress, token string) bool {
	ctx, span, ctxLogger := v.tracer.StartWithLogger(ctx, v.logger)
	defer span.End()

	payload, err := json.Marshal(map[string]string{
		"secret":   v.secretKey,
		"response": token,
		"remoteip": ipAddress,
	})
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "failed to marshal payload"))
		return false
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", bytes.NewBuffer(payload))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "failed to create http request request"))
		return false
	}

	request.Header.Set("Content-Type", "application/json")
	response, err := v.httpClient.Do(request)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("failed to send http request to [%s]", request.URL.String())))
		return false
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "failed to read response body from cloudflare turnstile"))
		return false
	}

	ctxLogger.Info(fmt.Sprintf("successfully validated token with cloudflare with response [%s]", body))

	data := new(turnstileVerifyResponse)
	if err = json.Unmarshal(body, data); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "failed to unmarshal response from cloudflare turnstile"))
		return false
	}

	return data.Success
}
