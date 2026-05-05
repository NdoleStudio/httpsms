package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	httpsms "github.com/NdoleStudio/httpsms-go"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/wiremock/go-wiremock"
	wmJournal "github.com/wiremock/go-wiremock/journal"
)

const (
	apiBaseURL         = "http://localhost:8000"
	wiremockURL        = "http://localhost:8080"
	wiremockWebhookURL = "http://wiremock.local:8080" // reachable from API container, passes URL validation (needs a dot)
	userAPIKey         = "test-user-api-key"
)

type testPhone struct {
	PhoneNumber string
	PhoneAPIKey string
	FcmToken    string
}

func newAPIClient() *httpsms.Client {
	return httpsms.New(
		httpsms.WithBaseURL(apiBaseURL),
		httpsms.WithAPIKey(userAPIKey),
	)
}

func newPhoneClient(phoneAPIKey string) *httpsms.Client {
	return httpsms.New(
		httpsms.WithBaseURL(apiBaseURL),
		httpsms.WithAPIKey(phoneAPIKey),
	)
}

func newWireMockClient() *wiremock.Client {
	return wiremock.NewClient(wiremockURL)
}

func randomPhoneNumber() string {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "+18005550000"
	}

	return fmt.Sprintf("+1800555%04d", n.Int64())
}

func randomEncryptionKey() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return uuid.New().String()
	}

	return fmt.Sprintf("%x", b)
}

func setupPhone(ctx context.Context, t *testing.T, messagesPerMinute uint) testPhone {
	t.Helper()

	phoneNumber := randomPhoneNumber()
	fcmToken := "fcm-" + uuid.New().String()
	client := newAPIClient()

	// Create the phone API key first so that a few seconds pass (during phone upsert)
	// before we use it, giving the cache time to clear.
	apiKeyResp, resp, err := client.PhoneAPIKeys.Store(ctx, &httpsms.PhoneAPIKeyStoreParams{
		Name: "test-key-" + uuid.New().String(),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.HTTPResponse.StatusCode, "phone api key store failed")

	phoneAPIKeyValue := apiKeyResp.Data.APIKey
	require.NotEmpty(t, phoneAPIKeyValue)

	_, resp, err = client.Phones.Upsert(ctx, &httpsms.PhoneUpsertParams{
		PhoneNumber:              phoneNumber,
		FcmToken:                 fcmToken,
		MessagesPerMinute:        messagesPerMinute,
		MaxSendAttempts:          2,
		MessageExpirationSeconds: 600,
		SIM:                      "SIM1",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.HTTPResponse.StatusCode, "phone upsert failed")

	phoneClient := newPhoneClient(phoneAPIKeyValue)
	_, resp, err = phoneClient.Phones.UpsertFCMToken(ctx, &httpsms.PhoneFCMTokenParams{
		PhoneNumber: phoneNumber,
		FcmToken:    fcmToken,
		SIM:         "SIM1",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.HTTPResponse.StatusCode, "fcm token bind failed")

	return testPhone{
		PhoneNumber: phoneNumber,
		PhoneAPIKey: phoneAPIKeyValue,
		FcmToken:    fcmToken,
	}
}

func setupWebhook(ctx context.Context, t *testing.T, phoneNumber string, events []string) (signingKey string, webhookPath string) {
	t.Helper()

	signingKey = randomEncryptionKey()
	webhookPath = "/webhooks/" + uuid.New().String()
	webhookURL := wiremockWebhookURL + webhookPath

	client := newAPIClient()
	_, resp, err := client.Webhooks.Store(ctx, &httpsms.WebhookStoreParams{
		SigningKey:   signingKey,
		URL:          webhookURL,
		PhoneNumbers: []string{phoneNumber},
		Events:       events,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.HTTPResponse.StatusCode, "webhook store failed")

	return signingKey, webhookPath
}

func fireEvent(ctx context.Context, t *testing.T, phoneAPIKey string, messageID string, eventName string) {
	t.Helper()

	url := fmt.Sprintf("%s/v1/messages/%s/events", apiBaseURL, messageID)
	payload := map[string]interface{}{
		"event_name": eventName,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", phoneAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "fire event %s failed: %s", eventName, string(respBody))
}

func pollMessageStatus(ctx context.Context, t *testing.T, messageID string, targetStatus string, timeout time.Duration) httpsms.Message {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		url := fmt.Sprintf("%s/v1/messages/%s", apiBaseURL, messageID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err == nil {
			req.Header.Set("x-api-key", userAPIKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				body, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if readErr == nil && resp.StatusCode == http.StatusOK {
					var result httpsms.MessageResponse
					if json.Unmarshal(body, &result) == nil && result.Data.Status == targetStatus {
						return result.Data
					}
				}
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("message %s did not reach status %q within %v", messageID, targetStatus, timeout)
	return httpsms.Message{}
}

func fetchOutstandingMessage(ctx context.Context, t *testing.T, phoneAPIKey string, messageID string) map[string]interface{} {
	t.Helper()

	url := fmt.Sprintf("%s/v1/messages/outstanding?message_id=%s", apiBaseURL, messageID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("x-api-key", phoneAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "outstanding: %s", string(body))

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &result))
	data, ok := result["data"].(map[string]interface{})
	require.True(t, ok, "no data in outstanding response")
	return data
}

func findFCMRequests(t *testing.T, messageID string) []wmJournal.GetRequestResponse {
	t.Helper()
	wm := newWireMockClient()

	allReqs, err := wm.GetAllRequests()
	require.NoError(t, err)

	var matched []wmJournal.GetRequestResponse
	for _, req := range allReqs.Requests {
		if strings.Contains(req.Request.URL, "/messages:send") || strings.Contains(req.Request.AbsoluteURL, "/messages:send") {
			if strings.Contains(req.Request.Body, messageID) {
				matched = append(matched, req)
			}
		}
	}

	return matched
}

func findWebhookRequests(t *testing.T, webhookPath string) []wmJournal.GetRequestResponse {
	t.Helper()
	wm := newWireMockClient()

	allReqs, err := wm.GetAllRequests()
	require.NoError(t, err)

	var matched []wmJournal.GetRequestResponse
	for _, req := range allReqs.Requests {
		if strings.Contains(req.Request.URL, webhookPath) || strings.Contains(req.Request.AbsoluteURL, webhookPath) {
			matched = append(matched, req)
		}
	}

	return matched
}

func assertWebhookJWT(t *testing.T, request wmJournal.Request, signingKey string) {
	t.Helper()

	authHeader := request.Headers["Authorization"]
	if authHeader == "" {
		authHeader = request.Headers["authorization"]
	}
	require.NotEmpty(t, authHeader, "webhook request missing Authorization header")
	require.True(t, strings.HasPrefix(authHeader, "Bearer "), "Authorization header must start with Bearer")

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		require.Equal(t, jwt.SigningMethodHS256, token.Method, "unexpected signing method")
		return []byte(signingKey), nil
	})
	require.NoError(t, err, "JWT validation failed")
	require.True(t, token.Valid, "JWT token is not valid")

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok, "cannot parse claims")
	require.Equal(t, "api.httpsms.com", claims["iss"], "issuer mismatch")
	require.NotEmpty(t, claims["sub"], "subject mismatch")

	exp, err := claims.GetExpirationTime()
	require.NoError(t, err)
	require.True(t, exp.After(time.Now()), "token is expired")

	nbf, err := claims.GetNotBefore()
	require.NoError(t, err)
	require.True(t, nbf.Before(time.Now()), "token not yet valid")
}

func waitForWebhookEvents(t *testing.T, webhookPath string, expectedCount int, timeout time.Duration) []wmJournal.GetRequestResponse {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		requests := findWebhookRequests(t, webhookPath)
		if len(requests) >= expectedCount {
			return requests
		}
		time.Sleep(500 * time.Millisecond)
	}

	requests := findWebhookRequests(t, webhookPath)
	require.GreaterOrEqual(t, len(requests), expectedCount, "expected at least %d webhook events on %s, got %d", expectedCount, webhookPath, len(requests))
	return requests
}

func waitForFCMPush(t *testing.T, messageID string, timeout time.Duration) []wmJournal.GetRequestResponse {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		requests := findFCMRequests(t, messageID)
		if len(requests) >= 1 {
			return requests
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("FCM push for message %s not found within %v", messageID, timeout)
	return nil
}
