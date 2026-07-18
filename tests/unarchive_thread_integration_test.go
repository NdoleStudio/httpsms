package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	httpsms "github.com/NdoleStudio/httpsms-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type integrationThread struct {
	ID                 string  `json:"id"`
	Contact            string  `json:"contact"`
	Owner              string  `json:"owner"`
	IsArchived         bool    `json:"is_archived"`
	LastMessageContent *string `json:"last_message_content"`
}

// setUnarchiveThread flips the per-phone unarchive_thread flag via a raw PUT
// (the httpsms-go client has no field for it).
func setUnarchiveThread(ctx context.Context, t *testing.T, phoneNumber string, enabled bool) {
	t.Helper()

	// sim is required by PUT /v1/phones validation, so it must be sent; setupPhone
	// always provisions the test phone on SIM1, so this preserves the existing slot.
	payload := map[string]interface{}{
		"phone_number":     phoneNumber,
		"sim":              "SIM1",
		"unarchive_thread": enabled,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiBaseURL+"/v1/phones", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", userAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "set unarchive_thread failed: %s", string(respBody))
}

// receiveInbound submits an inbound message as the phone and returns the message ID.
func receiveInbound(ctx context.Context, t *testing.T, phoneAPIKey, from, to, content string, ts time.Time) string {
	t.Helper()

	payload := map[string]interface{}{
		"from":      from,
		"to":        to,
		"content":   content,
		"sim":       "SIM1",
		"timestamp": ts.UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"/v1/messages/receive", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", phoneAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "receive failed: %s", string(respBody))

	var result httpsms.MessageResponse
	require.NoError(t, json.Unmarshal(respBody, &result))
	id := result.Data.ID.String()
	require.NotEmpty(t, id)
	return id
}

// fetchThreads returns threads for an owner filtered by archived state.
func fetchThreads(ctx context.Context, t *testing.T, owner string, archived bool) []integrationThread {
	t.Helper()

	reqURL := fmt.Sprintf("%s/v1/message-threads?owner=%s&is_archived=%t&limit=20", apiBaseURL, url.QueryEscape(owner), archived)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	require.NoError(t, err)
	req.Header.Set("x-api-key", userAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "fetch threads failed: %s", string(respBody))

	var result struct {
		Data []integrationThread `json:"data"`
	}
	require.NoError(t, json.Unmarshal(respBody, &result))
	return result.Data
}

func findThreadByContact(threads []integrationThread, contact string) *integrationThread {
	for i := range threads {
		if threads[i].Contact == contact {
			return &threads[i]
		}
	}
	return nil
}

// waitForThread polls the archived/unarchived thread list until a thread for the
// contact appears (optionally matching a last-message content), then returns it.
func waitForThread(ctx context.Context, t *testing.T, owner, contact string, archived bool, wantContent string, timeout time.Duration) *integrationThread {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		thread := findThreadByContact(fetchThreads(ctx, t, owner, archived), contact)
		if thread != nil && (wantContent == "" || (thread.LastMessageContent != nil && *thread.LastMessageContent == wantContent)) {
			return thread
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// archiveThread archives a thread by ID.
func archiveThread(ctx context.Context, t *testing.T, threadID string) {
	t.Helper()

	body, err := json.Marshal(map[string]interface{}{"is_archived": true})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiBaseURL+"/v1/message-threads/"+threadID, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", userAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "archive thread failed: %s", string(respBody))
}

func TestUnarchiveThreadOnReceive_Enabled(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)
	setUnarchiveThread(ctx, t, phone.PhoneNumber, true)

	contact := randomPhoneNumber()
	content1 := "first inbound " + randomEncryptionKey()
	content2 := "second inbound " + randomEncryptionKey()

	// First inbound message creates the thread.
	msgID1 := receiveInbound(ctx, t, phone.PhoneAPIKey, contact, phone.PhoneNumber, content1, time.Now().Add(-1*time.Minute))
	pollMessageStatus(ctx, t, msgID1, "received", 15*time.Second)

	thread := waitForThread(ctx, t, phone.PhoneNumber, contact, false, "", 15*time.Second)
	require.NotNil(t, thread, "thread not created for contact %s", contact)

	// Archive it.
	archiveThread(ctx, t, thread.ID)
	archived := waitForThread(ctx, t, phone.PhoneNumber, contact, true, "", 10*time.Second)
	require.NotNil(t, archived, "thread was not archived")

	// Second inbound message should unarchive it.
	msgID2 := receiveInbound(ctx, t, phone.PhoneAPIKey, contact, phone.PhoneNumber, content2, time.Now())
	pollMessageStatus(ctx, t, msgID2, "received", 15*time.Second)

	unarchived := waitForThread(ctx, t, phone.PhoneNumber, contact, false, content2, 20*time.Second)
	require.NotNil(t, unarchived, "thread was not unarchived after inbound message")
	assert.False(t, unarchived.IsArchived)
	require.NotNil(t, unarchived.LastMessageContent)
	assert.Equal(t, content2, *unarchived.LastMessageContent)
}

func TestUnarchiveThreadOnReceive_Disabled(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60) // unarchive_thread defaults to false; do not enable it

	contact := randomPhoneNumber()
	content1 := "first inbound " + randomEncryptionKey()
	content2 := "second inbound " + randomEncryptionKey()

	msgID1 := receiveInbound(ctx, t, phone.PhoneAPIKey, contact, phone.PhoneNumber, content1, time.Now().Add(-1*time.Minute))
	pollMessageStatus(ctx, t, msgID1, "received", 15*time.Second)

	thread := waitForThread(ctx, t, phone.PhoneNumber, contact, false, "", 15*time.Second)
	require.NotNil(t, thread, "thread not created for contact %s", contact)

	archiveThread(ctx, t, thread.ID)
	archived := waitForThread(ctx, t, phone.PhoneNumber, contact, true, "", 10*time.Second)
	require.NotNil(t, archived, "thread was not archived")

	// Second inbound message must NOT unarchive it. Sync on the archived thread's
	// last_message_content updating to content2 (proves the listener processed it),
	// then assert it is still archived.
	msgID2 := receiveInbound(ctx, t, phone.PhoneAPIKey, contact, phone.PhoneNumber, content2, time.Now())
	pollMessageStatus(ctx, t, msgID2, "received", 15*time.Second)

	stillArchived := waitForThread(ctx, t, phone.PhoneNumber, contact, true, content2, 20*time.Second)
	require.NotNil(t, stillArchived, "archived thread did not reflect the second inbound message")
	assert.True(t, stillArchived.IsArchived, "thread should remain archived when unarchive_thread is disabled")

	// And it must not have leaked into the unarchived list.
	assert.Nil(t, findThreadByContact(fetchThreads(ctx, t, phone.PhoneNumber, false), contact),
		"thread should not appear in the unarchived list")
}
