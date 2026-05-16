package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	httpsms "github.com/NdoleStudio/httpsms-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendSMS_Encrypted(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)

	encryptionKey := randomEncryptionKey()
	signingKey, webhookPath := setupWebhook(ctx, t, phone.PhoneNumber, []string{
		"message.phone.sent",
		"message.phone.delivered",
	})

	client := newAPIClient()
	plaintext := "Hello encrypted world " + randomEncryptionKey()
	ciphertext, err := client.Cipher.Encrypt(encryptionKey, plaintext)
	require.NoError(t, err)
	require.NotEqual(t, plaintext, ciphertext)

	contactNumber := randomPhoneNumber()
	sendResp, resp, err := client.Messages.Send(ctx, &httpsms.MessageSendParams{
		From:      phone.PhoneNumber,
		To:        contactNumber,
		Content:   ciphertext,
		Encrypted: true,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.HTTPResponse.StatusCode)

	messageID := sendResp.Data.ID.String()
	require.NotEmpty(t, messageID)
	t.Logf("sent encrypted message: %s", messageID)

	fcmRequests := waitForFCMPush(t, messageID, 30*time.Second)
	require.Len(t, fcmRequests, 1)

	outstanding := fetchOutstandingMessage(ctx, t, phone.PhoneAPIKey, messageID)
	assert.Equal(t, true, outstanding["encrypted"])
	assert.Equal(t, ciphertext, outstanding["content"])
	assert.NotEqual(t, plaintext, outstanding["content"])

	fireEvent(ctx, t, phone.PhoneAPIKey, messageID, "SENT")
	time.Sleep(200 * time.Millisecond)
	fireEvent(ctx, t, phone.PhoneAPIKey, messageID, "DELIVERED")

	msg := pollMessageStatus(ctx, t, messageID, "delivered", 30*time.Second)
	assert.Equal(t, "delivered", msg.Status)
	assert.True(t, msg.Encrypted)
	assert.Equal(t, ciphertext, msg.Content)

	decrypted, err := client.Cipher.Decrypt(encryptionKey, msg.Content)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	webhookReqs := waitForWebhookEvents(t, webhookPath, 2, 30*time.Second)
	for _, req := range webhookReqs {
		assertWebhookJWT(t, req.Request, signingKey)
	}

	var eventTypes []string
	for _, req := range webhookReqs {
		if et, ok := req.Request.Headers["X-Event-Type"]; ok {
			eventTypes = append(eventTypes, et)
		} else if et, ok := req.Request.Headers["x-event-type"]; ok {
			eventTypes = append(eventTypes, et)
		}
	}
	assert.Contains(t, eventTypes, "message.phone.sent")
	assert.Contains(t, eventTypes, "message.phone.delivered")
}

func TestReceiveSMS_Encrypted(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)

	encryptionKey := randomEncryptionKey()
	signingKey, webhookPath := setupWebhook(ctx, t, phone.PhoneNumber, []string{
		"message.phone.received",
	})

	client := newAPIClient()
	plaintext := "Incoming secret message " + randomEncryptionKey()
	ciphertext, err := client.Cipher.Encrypt(encryptionKey, plaintext)
	require.NoError(t, err)

	contactNumber := randomPhoneNumber()
	receivePayload := map[string]interface{}{
		"from":      contactNumber,
		"to":        phone.PhoneNumber,
		"content":   ciphertext,
		"encrypted": true,
		"sim":       "SIM1",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(receivePayload)
	require.NoError(t, err)

	url := apiBaseURL + "/v1/messages/receive"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", phone.PhoneAPIKey)

	httpResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "receive response: %s", string(respBody))

	var receiveResult httpsms.MessageResponse
	require.NoError(t, json.Unmarshal(respBody, &receiveResult))
	messageID := receiveResult.Data.ID.String()
	require.NotEmpty(t, messageID)
	t.Logf("received encrypted message: %s", messageID)

	msg := pollMessageStatus(ctx, t, messageID, "received", 15*time.Second)
	assert.Equal(t, "received", msg.Status)
	assert.True(t, msg.Encrypted)
	assert.Equal(t, ciphertext, msg.Content)
	assert.NotEqual(t, plaintext, msg.Content)

	decrypted, err := client.Cipher.Decrypt(encryptionKey, msg.Content)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	webhookReqs := waitForWebhookEvents(t, webhookPath, 1, 30*time.Second)
	require.GreaterOrEqual(t, len(webhookReqs), 1)
	assertWebhookJWT(t, webhookReqs[0].Request, signingKey)

	eventType := webhookReqs[0].Request.Headers["X-Event-Type"]
	if eventType == "" {
		eventType = webhookReqs[0].Request.Headers["x-event-type"]
	}
	assert.Equal(t, "message.phone.received", eventType)
}

func TestSendSMS_RateLimit(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 10)

	signingKey, webhookPath := setupWebhook(ctx, t, phone.PhoneNumber, []string{
		"message.phone.sent",
		"message.phone.delivered",
	})

	client := newAPIClient()
	contactNumber := randomPhoneNumber()

	sendResp1, resp1, err := client.Messages.Send(ctx, &httpsms.MessageSendParams{
		From:    phone.PhoneNumber,
		To:      contactNumber,
		Content: "Rate limit test message 1",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp1.HTTPResponse.StatusCode)
	msgID1 := sendResp1.Data.ID.String()

	sendResp2, resp2, err := client.Messages.Send(ctx, &httpsms.MessageSendParams{
		From:    phone.PhoneNumber,
		To:      contactNumber,
		Content: "Rate limit test message 2",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp2.HTTPResponse.StatusCode)
	msgID2 := sendResp2.Data.ID.String()

	t.Logf("sent messages: %s, %s", msgID1, msgID2)

	fcm1 := waitForFCMPush(t, msgID1, 30*time.Second)
	require.Len(t, fcm1, 1)

	fcm2 := waitForFCMPush(t, msgID2, 30*time.Second)
	require.Len(t, fcm2, 1)

	time1 := fcm1[0].Request.LoggedDate
	time2 := fcm2[0].Request.LoggedDate
	gapMs := time2 - time1
	if gapMs < 0 {
		gapMs = time1 - time2
	}
	t.Logf("FCM push gap: %dms", gapMs)
	assert.GreaterOrEqual(t, gapMs, int64(5500), "rate limit gap should be >= 5500ms (6s minus timing tolerance), got %dms", gapMs)

	fireEvent(ctx, t, phone.PhoneAPIKey, msgID1, "SENT")
	fireEvent(ctx, t, phone.PhoneAPIKey, msgID1, "DELIVERED")
	fireEvent(ctx, t, phone.PhoneAPIKey, msgID2, "SENT")
	fireEvent(ctx, t, phone.PhoneAPIKey, msgID2, "DELIVERED")

	msg1 := pollMessageStatus(ctx, t, msgID1, "delivered", 15*time.Second)
	msg2 := pollMessageStatus(ctx, t, msgID2, "delivered", 15*time.Second)
	assert.Equal(t, "delivered", msg1.Status)
	assert.Equal(t, "delivered", msg2.Status)

	webhookReqs := waitForWebhookEvents(t, webhookPath, 4, 30*time.Second)
	for _, req := range webhookReqs {
		assertWebhookJWT(t, req.Request, signingKey)
	}
}

func TestRotateAPIKey_InvalidatesCache(t *testing.T) {
	ctx := context.Background()

	// Use a dedicated test user so we don't mutate the shared userAPIKey
	rotateUserAPIKey := "rotate-test-api-key"
	rotateUserID := "rotate-test-user-id"

	// 1) Confirm the dedicated user's API key works and warm the cache
	meURL := apiBaseURL + "/v1/users/me"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meURL, nil)
	require.NoError(t, err)
	req.Header.Set("x-api-key", rotateUserAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "initial auth failed: %s", string(body))

	// Parse the current API key from the response
	var meResp struct {
		Data struct {
			ID     string `json:"id"`
			APIKey string `json:"api_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &meResp))
	require.Equal(t, rotateUserID, meResp.Data.ID)
	oldAPIKey := meResp.Data.APIKey
	require.NotEmpty(t, oldAPIKey)
	t.Logf("user ID: %s, old API key prefix: %s...", rotateUserID, oldAPIKey[:10])

	// 2) Rotate the API key
	rotateURL := fmt.Sprintf("%s/v1/users/%s/api-keys", apiBaseURL, rotateUserID)
	req, err = http.NewRequestWithContext(ctx, http.MethodDelete, rotateURL, nil)
	require.NoError(t, err)
	req.Header.Set("x-api-key", rotateUserAPIKey)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "rotate failed: %s", string(body))

	// Parse new API key from rotate response
	var rotateResp struct {
		Data struct {
			APIKey string `json:"api_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &rotateResp))
	newAPIKey := rotateResp.Data.APIKey
	require.NotEmpty(t, newAPIKey)
	require.NotEqual(t, oldAPIKey, newAPIKey, "API key should have changed after rotation")
	t.Logf("new API key prefix: %s...", newAPIKey[:10])

	// 3) Old API key should immediately fail (401) — this is the bug regression check
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, meURL, nil)
	require.NoError(t, err)
	req.Header.Set("x-api-key", oldAPIKey)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "old API key should return 401 after rotation")

	// 4) New API key should work
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, meURL, nil)
	require.NoError(t, err)
	req.Header.Set("x-api-key", newAPIKey)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "new API key should work: %s", string(body))
}

func TestSendSMS_OutstandingFlow(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)

	signingKey, webhookPath := setupWebhook(ctx, t, phone.PhoneNumber, []string{
		"message.phone.sent",
		"message.phone.delivered",
	})

	client := newAPIClient()
	contactNumber := randomPhoneNumber()
	content := "Outstanding flow test " + randomEncryptionKey()

	sendResp, resp, err := client.Messages.Send(ctx, &httpsms.MessageSendParams{
		From:    phone.PhoneNumber,
		To:      contactNumber,
		Content: content,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.HTTPResponse.StatusCode)

	messageID := sendResp.Data.ID.String()
	t.Logf("sent message: %s", messageID)

	fcmReqs := waitForFCMPush(t, messageID, 30*time.Second)
	require.Len(t, fcmReqs, 1)
	assert.Contains(t, fcmReqs[0].Request.Body, messageID)
	assert.True(t, strings.Contains(fcmReqs[0].Request.URL, "/messages:send") || strings.Contains(fcmReqs[0].Request.AbsoluteURL, "/messages:send"))

	outstanding := fetchOutstandingMessage(ctx, t, phone.PhoneAPIKey, messageID)
	assert.Equal(t, messageID, outstanding["id"])
	assert.Equal(t, content, outstanding["content"])
	assert.Equal(t, phone.PhoneNumber, outstanding["owner"])
	assert.Equal(t, contactNumber, outstanding["contact"])

	fireEvent(ctx, t, phone.PhoneAPIKey, messageID, "SENT")
	time.Sleep(200 * time.Millisecond)
	fireEvent(ctx, t, phone.PhoneAPIKey, messageID, "DELIVERED")

	msg := pollMessageStatus(ctx, t, messageID, "delivered", 30*time.Second)
	assert.Equal(t, "delivered", msg.Status)
	assert.Equal(t, content, msg.Content)

	webhookReqs := waitForWebhookEvents(t, webhookPath, 2, 30*time.Second)
	for _, req := range webhookReqs {
		assertWebhookJWT(t, req.Request, signingKey)
	}
}

func TestHeartbeat_StoreAndIndex(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)

	// Store a heartbeat via phone API key (retry to allow async phone-API-key association)
	storePayload := map[string]interface{}{
		"phone_numbers": []string{phone.PhoneNumber},
		"charging":      true,
	}

	url := apiBaseURL + "/v1/heartbeats"
	var respBody []byte
	var statusCode int
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		body, err := json.Marshal(storePayload)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", phone.PhoneAPIKey)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		respBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)

		statusCode = resp.StatusCode
		if statusCode == http.StatusCreated {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.Equal(t, http.StatusCreated, statusCode, "store heartbeat failed: %s", string(respBody))

	// Read heartbeats back via user API key
	client := newAPIClient()
	heartbeats, indexResp, err := client.Heartbeats.Index(ctx, &httpsms.HeartbeatIndexParams{
		Owner: phone.PhoneNumber,
		Limit: 1,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, indexResp.HTTPResponse.StatusCode)

	require.NotNil(t, heartbeats)
	require.GreaterOrEqual(t, len(heartbeats.Data), 1, "expected at least 1 heartbeat")

	hb := heartbeats.Data[0]
	assert.Equal(t, phone.PhoneNumber, hb.Owner)
	assert.True(t, hb.Charging)
	assert.False(t, hb.Timestamp.IsZero(), "timestamp should not be zero")
}
