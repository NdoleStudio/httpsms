package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/url"
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
	adapterControlURL  = "http://localhost:9092"
	userAPIKey         = "test-user-api-key"
	systemAPIKey       = "system-user-api-key"
)

type testPhone struct {
	PhoneNumber string
	PhoneAPIKey string
	FcmToken    string
}

type adapterTestPhone struct {
	testPhone
	PhoneID   string
	GatewayID string
}

type notificationRecord struct {
	GatewayID string            `json:"gateway_id"`
	Data      map[string]string `json:"data"`
	MessageID string            `json:"message_id,omitempty"`
	Kind      string            `json:"kind"`
	Processed bool              `json:"processed"`
	Error     string            `json:"error,omitempty"`
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

	waitForPhoneAuthorization(ctx, t, phoneAPIKeyValue, phoneNumber, 20*time.Second)

	return testPhone{
		PhoneNumber: phoneNumber,
		PhoneAPIKey: phoneAPIKeyValue,
		FcmToken:    fcmToken,
	}
}

func setupAdapterPhone(ctx context.Context, t *testing.T, messagesPerMinute uint) adapterTestPhone {
	t.Helper()

	gatewayID := uuid.NewString()
	phoneNumber := randomPhoneNumber()
	client := newAPIClient()

	apiKeyResponse, response, err := client.PhoneAPIKeys.Store(ctx, &httpsms.PhoneAPIKeyStoreParams{
		Name: "adapter-test-key-" + uuid.NewString(),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode, "phone api key store failed")

	phoneAPIKey := apiKeyResponse.Data.APIKey
	require.NotEmpty(t, phoneAPIKey)

	registrationBody, err := json.Marshal(map[string]any{
		"phone_number":  phoneNumber,
		"phone_api_key": phoneAPIKey,
	})
	require.NoError(t, err)
	registrationRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		fmt.Sprintf("%s/test/gateways/%s", adapterControlURL, gatewayID),
		bytes.NewReader(registrationBody),
	)
	require.NoError(t, err)
	registrationRequest.Header.Set("Content-Type", "application/json")
	registrationResponse, err := http.DefaultClient.Do(registrationRequest)
	require.NoError(t, err)
	registrationResponseBody, err := io.ReadAll(registrationResponse.Body)
	registrationResponse.Body.Close()
	require.NoError(t, err)
	require.Equal(
		t,
		http.StatusNoContent,
		registrationResponse.StatusCode,
		"adapter gateway registration failed: %s",
		string(registrationResponseBody),
	)

	callbackURL := fmt.Sprintf("https://adapter-emulator:9091/notifications/%s", gatewayID)
	phoneResponse, response, err := client.Phones.Upsert(ctx, &httpsms.PhoneUpsertParams{
		PhoneNumber:              phoneNumber,
		FcmToken:                 callbackURL,
		MessagesPerMinute:        messagesPerMinute,
		MaxSendAttempts:          2,
		MessageExpirationSeconds: 600,
		SIM:                      "SIM1",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode, "phone upsert failed")
	require.NotEmpty(t, phoneResponse.Data.ID)

	phoneClient := newPhoneClient(phoneAPIKey)
	_, response, err = phoneClient.Phones.UpsertFCMToken(ctx, &httpsms.PhoneFCMTokenParams{
		PhoneNumber: phoneNumber,
		FcmToken:    callbackURL,
		SIM:         "SIM1",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode, "adapter callback bind failed")

	waitForPhoneAuthorization(ctx, t, phoneAPIKey, phoneNumber, 20*time.Second)

	return adapterTestPhone{
		testPhone: testPhone{
			PhoneNumber: phoneNumber,
			PhoneAPIKey: phoneAPIKey,
			FcmToken:    callbackURL,
		},
		PhoneID:   phoneResponse.Data.ID,
		GatewayID: gatewayID,
	}
}

func dispatchInternalEvent(ctx context.Context, t *testing.T, event map[string]any) {
	t.Helper()

	body, err := json.Marshal(event)
	require.NoError(t, err)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"/v1/events", bytes.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", systemAPIKey)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, response.StatusCode, "event dispatch failed: %s", string(responseBody))
}

func waitForAdapterMessageRecords(
	t *testing.T,
	gatewayID string,
	messageID string,
	timeout time.Duration,
) []notificationRecord {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var records []notificationRecord
	var lastErr error
	for time.Now().Before(deadline) {
		records, lastErr = fetchAdapterNotificationRecords(gatewayID, messageID)
		if lastErr == nil && len(records) > 0 && adapterRecordsProcessed(records) {
			return records
		}
		time.Sleep(500 * time.Millisecond)
	}

	require.NoError(t, lastErr)
	require.NotEmpty(t, records, "adapter message record for %s was not available within %v", messageID, timeout)
	require.True(t, adapterRecordsProcessed(records), "adapter message records were not processed: %#v", records)
	return records
}

func waitForAdapterHeartbeatRecord(
	t *testing.T,
	gatewayID string,
	timeout time.Duration,
) notificationRecord {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		records, err := fetchAdapterNotificationRecords(gatewayID, "")
		lastErr = err
		if err == nil {
			for _, record := range records {
				if record.Kind == "heartbeat" && record.Processed {
					return record
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	require.NoError(t, lastErr)
	t.Fatalf("processed adapter heartbeat record was not available within %v", timeout)
	return notificationRecord{}
}

func triggerAdapterIncoming(
	ctx context.Context,
	t *testing.T,
	phone adapterTestPhone,
	contact string,
	content string,
) string {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"contact":   contact,
		"content":   content,
		"encrypted": false,
	})
	require.NoError(t, err)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/test/gateways/%s/incoming", adapterControlURL, phone.GatewayID),
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode, "adapter incoming trigger failed: %s", string(responseBody))

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(responseBody, &result))
	require.NotEmpty(t, result.Data.ID)
	return result.Data.ID
}

func fetchAdapterNotificationRecords(gatewayID string, messageID string) ([]notificationRecord, error) {
	endpoint := fmt.Sprintf("%s/test/gateways/%s/notifications", adapterControlURL, gatewayID)
	if messageID != "" {
		endpoint += "?message_id=" + url.QueryEscape(messageID)
	}

	response, err := (&http.Client{Timeout: 5 * time.Second}).Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch adapter notification records: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read adapter notification records: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch adapter notification records: status %d: %s", response.StatusCode, string(responseBody))
	}

	var result struct {
		Data []notificationRecord `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode adapter notification records: %w", err)
	}
	return result.Data, nil
}

func adapterRecordsProcessed(records []notificationRecord) bool {
	for _, record := range records {
		if !record.Processed {
			return false
		}
	}
	return true
}

func waitForPhoneAuthorization(
	ctx context.Context,
	t *testing.T,
	phoneAPIKey string,
	phoneNumber string,
	timeout time.Duration,
) {
	t.Helper()

	body, err := json.Marshal(map[string]interface{}{
		"phone_numbers": []string{phoneNumber},
		"charging":      true,
	})
	require.NoError(t, err)

	var responseBody []byte
	var statusCode int
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		request, requestErr := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			apiBaseURL+"/v1/heartbeats",
			bytes.NewReader(body),
		)
		require.NoError(t, requestErr)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("x-api-key", phoneAPIKey)

		response, requestErr := http.DefaultClient.Do(request)
		require.NoError(t, requestErr)
		responseBody, requestErr = io.ReadAll(response.Body)
		response.Body.Close()
		require.NoError(t, requestErr)

		statusCode = response.StatusCode
		if statusCode == http.StatusCreated {
			return
		}
		if statusCode != http.StatusUnauthorized {
			require.Equal(t, http.StatusCreated, statusCode, "phone authorization check failed: %s", string(responseBody))
		}

		time.Sleep(500 * time.Millisecond)
	}

	require.Equal(t, http.StatusCreated, statusCode, "phone authorization was not ready within %v: %s", timeout, string(responseBody))
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

type BulkMessageEntry struct {
	RequestID      string `json:"request_id"`
	Total          int    `json:"total"`
	ScheduledCount int    `json:"scheduled_count"`
	PendingCount   int    `json:"pending_count"`
	FailedCount    int    `json:"failed_count"`
	ExpiredCount   int    `json:"expired_count"`
	SentCount      int    `json:"sent_count"`
	DeliveredCount int    `json:"delivered_count"`
	CreatedAt      string `json:"created_at"`
}

func uploadBulkFile(ctx context.Context, t *testing.T, filename string, fileBytes []byte) (int, []byte) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("document", filename)
	require.NoError(t, err)

	_, err = part.Write(fileBytes)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	url := apiBaseURL + "/v1/bulk-messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-api-key", userAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, body
}

func fetchBulkMessages(ctx context.Context, t *testing.T) []BulkMessageEntry {
	t.Helper()

	url := apiBaseURL + "/v1/bulk-messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("x-api-key", userAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "fetch bulk messages failed: %s", string(body))

	var result struct {
		Data []BulkMessageEntry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &result))
	return result.Data
}

func searchMessages(ctx context.Context, t *testing.T, contact string, owner string) []httpsms.Message {
	t.Helper()

	url := fmt.Sprintf("%s/v1/messages?contact=%s&owner=%s&limit=20&skip=0", apiBaseURL, contact, owner)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("x-api-key", userAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "search messages failed: %s", string(body))

	var result struct {
		Data []httpsms.Message `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &result))
	return result.Data
}

func findBulkEntry(entries []BulkMessageEntry, requestID string) *BulkMessageEntry {
	for i := range entries {
		if entries[i].RequestID == requestID {
			return &entries[i]
		}
	}
	return nil
}
