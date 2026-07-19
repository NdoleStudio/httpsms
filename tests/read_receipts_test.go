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

type integrationMessageThread struct {
	ID                 string  `json:"id"`
	Contact            string  `json:"contact"`
	IsRead             bool    `json:"is_read"`
	LastMessageContent *string `json:"last_message_content"`
}

func requestJSON(
	ctx context.Context,
	t *testing.T,
	method string,
	path string,
	apiKey string,
	payload any,
	expectedStatus int,
	output any,
) {
	t.Helper()

	var encoded []byte
	if payload != nil {
		var err error
		encoded, err = json.Marshal(payload)
		require.NoError(t, err)
	}

	deadline := time.Now().Add(20 * time.Second)
	for {
		request, err := http.NewRequestWithContext(ctx, method, apiBaseURL+path, bytes.NewReader(encoded))
		require.NoError(t, err)
		request.Header.Set("x-api-key", apiKey)
		request.Header.Set("Content-Type", "application/json")

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)

		responseBody, err := io.ReadAll(response.Body)
		response.Body.Close()
		require.NoError(t, err)

		if response.StatusCode == http.StatusUnauthorized &&
			apiKey != userAPIKey &&
			time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		require.Equal(t, expectedStatus, response.StatusCode, "response: %s", string(responseBody))
		if output != nil {
			require.NoError(t, json.Unmarshal(responseBody, output))
		}
		return
	}
}

func fetchMessageThreads(ctx context.Context, t *testing.T, owner string) []integrationMessageThread {
	t.Helper()

	var response struct {
		Data []integrationMessageThread `json:"data"`
	}
	path := fmt.Sprintf(
		"/v1/message-threads?owner=%s&skip=0&limit=20&is_archived=false",
		url.QueryEscape(owner),
	)
	requestJSON(ctx, t, http.MethodGet, path, userAPIKey, nil, http.StatusOK, &response)
	return response.Data
}

func waitForMessageThread(
	ctx context.Context,
	t *testing.T,
	owner string,
	contact string,
	timeout time.Duration,
	matches func(integrationMessageThread) bool,
) integrationMessageThread {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, thread := range fetchMessageThreads(ctx, t, owner) {
			if thread.Contact == contact && matches(thread) {
				return thread
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("thread %s -> %s did not reach the expected state within %v", owner, contact, timeout)
	return integrationMessageThread{}
}

func markMessageThreadRead(ctx context.Context, t *testing.T, threadID string) integrationMessageThread {
	t.Helper()

	var response struct {
		Data integrationMessageThread `json:"data"`
	}
	requestJSON(
		ctx,
		t,
		http.MethodPut,
		"/v1/message-threads/"+threadID,
		userAPIKey,
		map[string]any{"is_read": true},
		http.StatusOK,
		&response,
	)
	return response.Data
}

func TestMessageThreadReadReceipts(t *testing.T) {
	ctx := context.Background()
	phone := setupPhone(ctx, t, 60)
	contact := randomPhoneNumber()

	requestJSON(
		ctx,
		t,
		http.MethodPost,
		"/v1/messages/receive",
		phone.PhoneAPIKey,
		map[string]any{
			"from":      contact,
			"to":        phone.PhoneNumber,
			"content":   "Unread inbound message",
			"encrypted": false,
			"sim":       "SIM1",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		},
		http.StatusOK,
		nil,
	)

	thread := waitForMessageThread(ctx, t, phone.PhoneNumber, contact, 20*time.Second, func(thread integrationMessageThread) bool {
		return !thread.IsRead
	})
	assert.False(t, thread.IsRead)

	updated := markMessageThreadRead(ctx, t, thread.ID)
	assert.True(t, updated.IsRead)
	assert.Equal(t, contact, updated.Contact)
	require.NotNil(t, updated.LastMessageContent)
	assert.Equal(t, "Unread inbound message", *updated.LastMessageContent)
	waitForMessageThread(ctx, t, phone.PhoneNumber, contact, 10*time.Second, func(thread integrationMessageThread) bool {
		return thread.IsRead
	})

	requestJSON(
		ctx,
		t,
		http.MethodPost,
		"/v1/messages/calls/missed",
		phone.PhoneAPIKey,
		map[string]any{
			"from":      contact,
			"to":        phone.PhoneNumber,
			"sim":       "SIM1",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		},
		http.StatusOK,
		nil,
	)

	thread = waitForMessageThread(ctx, t, phone.PhoneNumber, contact, 20*time.Second, func(thread integrationMessageThread) bool {
		return !thread.IsRead &&
			thread.LastMessageContent != nil &&
			*thread.LastMessageContent == "Missed phone call"
	})
	assert.False(t, thread.IsRead)

	outboundContent := "Outbound activity preserves unread"
	client := newAPIClient()
	_, response, err := client.Messages.Send(ctx, &httpsms.MessageSendParams{
		From:    phone.PhoneNumber,
		To:      contact,
		Content: outboundContent,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)

	thread = waitForMessageThread(ctx, t, phone.PhoneNumber, contact, 20*time.Second, func(thread integrationMessageThread) bool {
		return thread.LastMessageContent != nil &&
			*thread.LastMessageContent == outboundContent
	})
	assert.False(t, thread.IsRead, "outbound activity must not clear unread state")
}
