package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// EmulatorFCMClient sends FCM messages to the phone emulator via HTTP.
type EmulatorFCMClient struct {
	httpClient *http.Client
	endpoint   string
	logger     telemetry.Logger
}

// NewEmulatorFCMClient creates a new EmulatorFCMClient.
func NewEmulatorFCMClient(httpClient *http.Client, endpoint string, logger telemetry.Logger) *EmulatorFCMClient {
	return &EmulatorFCMClient{
		httpClient: httpClient,
		endpoint:   endpoint,
		logger:     logger,
	}
}

// emulatorFCMRequest is the payload sent to the emulator's FCM endpoint.
type emulatorFCMRequest struct {
	Message *emulatorFCMMessage `json:"message"`
}

type emulatorFCMMessage struct {
	Token   string            `json:"token"`
	Data    map[string]string `json:"data,omitempty"`
	Android *emulatorAndroid  `json:"android,omitempty"`
}

type emulatorAndroid struct {
	Priority string `json:"priority,omitempty"`
}

// emulatorFCMResponse is the response from the emulator.
type emulatorFCMResponse struct {
	Name string `json:"name"`
}

// Send sends a message to the emulator's FCM endpoint.
func (c *EmulatorFCMClient) Send(ctx context.Context, message *messaging.Message) (string, error) {
	payload := &emulatorFCMRequest{
		Message: &emulatorFCMMessage{
			Token: message.Token,
			Data:  message.Data,
		},
	}
	if message.Android != nil {
		payload.Message.Android = &emulatorAndroid{
			Priority: message.Android.Priority,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", stacktrace.Propagate(err, "cannot marshal FCM request for emulator")
	}

	url := fmt.Sprintf("%s/v1/projects/httpsms-test/messages:send", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", stacktrace.Propagate(err, "cannot create HTTP request for emulator FCM")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", stacktrace.Propagate(err, fmt.Sprintf("cannot send FCM to emulator at [%s]", url))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", stacktrace.Propagate(err, "cannot read emulator FCM response body")
	}

	if resp.StatusCode != http.StatusOK {
		return "", stacktrace.NewError("emulator FCM returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result emulatorFCMResponse
	if err = json.Unmarshal(respBody, &result); err != nil {
		return "", stacktrace.Propagate(err, "cannot decode emulator FCM response")
	}

	c.logger.Info(fmt.Sprintf("emulator FCM sent successfully: %s", result.Name))
	return result.Name, nil
}
