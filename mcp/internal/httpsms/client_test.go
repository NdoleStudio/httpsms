package httpsms_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer starts an httptest.Server that runs assert (given the
// decoded request) and writes response as the JSON body with status.
func newTestServer(t *testing.T, status int, response any, assertReq func(t *testing.T, r *http.Request)) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if assertReq != nil {
			assertReq(t, r)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	t.Cleanup(server.Close)
	return server
}

func requireBearer(t *testing.T, r *http.Request, token string) {
	t.Helper()
	assert.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
}

func requireRequestID(t *testing.T, r *http.Request) string {
	t.Helper()
	requestID := r.Header.Get("X-Request-Id")
	assert.NotEmpty(t, requestID, "expected a non-empty X-Request-Id header")
	return requestID
}

func TestClient_ListPhones(t *testing.T) {
	const token = "delegated-token-list-phones"

	server := newTestServer(t, http.StatusOK, httpsms.Response[[]httpsms.Phone]{
		Status:  "success",
		Message: "fetched 1 phone",
		Data: []httpsms.Phone{
			{ID: "phone-1", PhoneNumber: "+18005550199", SIM: "DEFAULT", MessagesPerMinute: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		},
	}, func(t *testing.T, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/phones", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		requireBearer(t, r, token)
		requireRequestID(t, r)

		query := r.URL.Query()
		assert.Equal(t, "5", query.Get("skip"))
		assert.Equal(t, "acme", query.Get("query"))
		assert.Equal(t, "10", query.Get("limit"))
	})

	client := httpsms.NewClient(server.URL)
	phones, err := client.ListPhones(t.Context(), token, httpsms.ListPhonesParams{Skip: 5, Query: "acme", Limit: 10})
	require.NoError(t, err)
	require.Len(t, phones, 1)
	assert.Equal(t, "phone-1", phones[0].ID)
	assert.Equal(t, "+18005550199", phones[0].PhoneNumber)
	assert.Equal(t, "DEFAULT", phones[0].SIM)
}

func TestClient_UsesADistinctRequestIDPerCall(t *testing.T) {
	seenRequestIDs := map[string]bool{}

	server := newTestServer(t, http.StatusOK, httpsms.Response[[]httpsms.Phone]{Status: "success", Data: []httpsms.Phone{}}, func(t *testing.T, r *http.Request) {
		seenRequestIDs[requireRequestID(t, r)] = true
	})

	client := httpsms.NewClient(server.URL)
	_, err := client.ListPhones(t.Context(), "token", httpsms.ListPhonesParams{})
	require.NoError(t, err)
	_, err = client.ListPhones(t.Context(), "token", httpsms.ListPhonesParams{})
	require.NoError(t, err)

	assert.Len(t, seenRequestIDs, 2, "expected a distinct request ID per call")
}

func TestClient_ListPhones_OmitsZeroSkipAndLimit(t *testing.T) {
	server := newTestServer(t, http.StatusOK, httpsms.Response[[]httpsms.Phone]{Status: "success", Data: []httpsms.Phone{}}, func(t *testing.T, r *http.Request) {
		query := r.URL.Query()
		assert.False(t, query.Has("skip"))
		assert.False(t, query.Has("limit"))
		assert.False(t, query.Has("query"))
	})

	client := httpsms.NewClient(server.URL)
	_, err := client.ListPhones(t.Context(), "token", httpsms.ListPhonesParams{})
	require.NoError(t, err)
}

func TestClient_SendSMS(t *testing.T) {
	const token = "delegated-token-send-sms"
	sendAt := time.Date(2025, 12, 19, 16, 39, 57, 0, time.UTC)

	server := newTestServer(t, http.StatusOK, httpsms.Response[httpsms.Message]{
		Status:  "success",
		Message: "message added to queue",
		Data: httpsms.Message{
			ID:      "message-1",
			Owner:   "+18005550199",
			Contact: "+18005550100",
			Content: "hello",
			Status:  "pending",
			Type:    "mobile-terminated",
			SIM:     "DEFAULT",
		},
	}, func(t *testing.T, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/messages/send", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		requireBearer(t, r, token)
		requireRequestID(t, r)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "+18005550199", body["from"])
		assert.Equal(t, "+18005550100", body["to"])
		assert.Equal(t, "hello", body["content"])
		assert.Equal(t, true, body["encrypted"])
		assert.Equal(t, "req-1", body["request_id"])
		assert.Equal(t, []any{"https://example.com/image.jpg"}, body["attachments"])
		assert.Equal(t, "2025-12-19T16:39:57Z", body["send_at"])
	})

	client := httpsms.NewClient(server.URL)
	message, err := client.SendSMS(t.Context(), token, httpsms.SendSMSParams{
		From:        "+18005550199",
		To:          "+18005550100",
		Content:     "hello",
		Encrypted:   true,
		RequestID:   "req-1",
		Attachments: []string{"https://example.com/image.jpg"},
		SendAt:      &sendAt,
	})
	require.NoError(t, err)
	assert.Equal(t, "message-1", message.ID)
	assert.Equal(t, "pending", message.Status)
}

func TestClient_ListMessageThreads(t *testing.T) {
	const token = "delegated-token-list-threads"
	archived := true

	server := newTestServer(t, http.StatusOK, httpsms.Response[[]httpsms.MessageThread]{
		Status: "success",
		Data: []httpsms.MessageThread{
			{ID: "thread-1", Owner: "+18005550199", Contact: "+18005550100", IsArchived: true, UnreadCount: 2},
		},
	}, func(t *testing.T, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/message-threads", r.URL.Path)
		requireBearer(t, r, token)
		requireRequestID(t, r)

		query := r.URL.Query()
		assert.Equal(t, "+18005550199", query.Get("owner"))
		assert.Equal(t, "true", query.Get("is_archived"))
		assert.Equal(t, "true", query.Get("contacts"))
		assert.Equal(t, "vip", query.Get("query"))
		assert.Equal(t, "2", query.Get("skip"))
		assert.Equal(t, "15", query.Get("limit"))
	})

	client := httpsms.NewClient(server.URL)
	threads, err := client.ListMessageThreads(t.Context(), token, httpsms.ListMessageThreadsParams{
		Owner:        "+18005550199",
		IsArchived:   &archived,
		WithContacts: true,
		Query:        "vip",
		Skip:         2,
		Limit:        15,
	})
	require.NoError(t, err)
	require.Len(t, threads, 1)
	assert.True(t, threads[0].IsArchived)
	assert.EqualValues(t, 2, threads[0].UnreadCount)
}

func TestClient_ListMessageThreads_OmitsUnsetArchiveFilter(t *testing.T) {
	server := newTestServer(t, http.StatusOK, httpsms.Response[[]httpsms.MessageThread]{Status: "success", Data: []httpsms.MessageThread{}}, func(t *testing.T, r *http.Request) {
		query := r.URL.Query()
		assert.False(t, query.Has("is_archived"))
		assert.False(t, query.Has("contacts"))
	})

	client := httpsms.NewClient(server.URL)
	_, err := client.ListMessageThreads(t.Context(), "token", httpsms.ListMessageThreadsParams{Owner: "+18005550199"})
	require.NoError(t, err)
}

func TestClient_ListThreadMessages(t *testing.T) {
	const token = "delegated-token-list-thread-messages"

	server := newTestServer(t, http.StatusOK, httpsms.Response[[]httpsms.Message]{
		Status: "success",
		Data: []httpsms.Message{
			{ID: "message-1", Owner: "+18005550199", Contact: "+18005550100", Content: "hi", Encrypted: true},
		},
	}, func(t *testing.T, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		requireBearer(t, r, token)
		requireRequestID(t, r)

		query := r.URL.Query()
		assert.Equal(t, "+18005550199", query.Get("owner"))
		assert.Equal(t, "+18005550100", query.Get("contact"))
		assert.Equal(t, "3", query.Get("skip"))
		assert.Equal(t, "20", query.Get("limit"))
	})

	client := httpsms.NewClient(server.URL)
	messages, err := client.ListThreadMessages(t.Context(), token, httpsms.ListThreadMessagesParams{
		Owner:   "+18005550199",
		Contact: "+18005550100",
		Skip:    3,
		Limit:   20,
	})
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "hi", messages[0].Content)
	assert.True(t, messages[0].Encrypted)
}

func TestClient_ListIncomingMessages(t *testing.T) {
	const token = "delegated-token-list-incoming"
	descending := true

	server := newTestServer(t, http.StatusOK, httpsms.Response[[]httpsms.Message]{
		Status: "success",
		Data: []httpsms.Message{
			{ID: "message-2", Type: "mobile-originated", Status: "received"},
		},
	}, func(t *testing.T, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/messages/incoming", r.URL.Path)
		requireBearer(t, r, token)
		requireRequestID(t, r)

		query := r.URL.Query()
		assert.ElementsMatch(t, []string{"+18005550199", "+18005550188"}, query["owners"])
		assert.ElementsMatch(t, []string{"received", "pending"}, query["statuses"])
		assert.Equal(t, "created_at", query.Get("sort_by"))
		assert.Equal(t, "true", query.Get("sort_descending"))
		assert.Equal(t, "search text", query.Get("query"))
	})

	client := httpsms.NewClient(server.URL)
	messages, err := client.ListIncomingMessages(t.Context(), token, httpsms.ListIncomingMessagesParams{
		Owners:         []string{"+18005550199", "+18005550188"},
		Statuses:       []string{"received", "pending"},
		Query:          "search text",
		SortBy:         "created_at",
		SortDescending: &descending,
	})
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "mobile-originated", messages[0].Type)
}

func TestClient_CreatePhoneAPIKey(t *testing.T) {
	const token = "delegated-token-create-key"

	server := newTestServer(t, http.StatusOK, httpsms.Response[httpsms.PhoneAPIKey]{
		Status: "success",
		Data: httpsms.PhoneAPIKey{
			ID:     "key-1",
			Name:   "My Phone API Key",
			APIKey: "pk_secretvalue",
		},
	}, func(t *testing.T, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/phone-api-keys", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		requireBearer(t, r, token)
		requireRequestID(t, r)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "My Phone API Key", body["name"])
	})

	client := httpsms.NewClient(server.URL)
	key, err := client.CreatePhoneAPIKey(t.Context(), token, httpsms.CreatePhoneAPIKeyParams{Name: "My Phone API Key"})
	require.NoError(t, err)
	assert.Equal(t, "key-1", key.ID)
	assert.Equal(t, "pk_secretvalue", key.APIKey)
}

func TestClient_RotateUserAPIKey(t *testing.T) {
	const token = "delegated-token-rotate-key"
	const userID = "WB7DRDWrJZRGbYrv2CKGkqbzvqdC"

	server := newTestServer(t, http.StatusOK, httpsms.Response[httpsms.User]{
		Status: "success",
		Data: httpsms.User{
			ID:     userID,
			Email:  "user@example.com",
			APIKey: "new-secret-api-key",
		},
	}, func(t *testing.T, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/v1/users/"+userID+"/api-keys", r.URL.Path)
		requireBearer(t, r, token)
		requireRequestID(t, r)
		assert.Empty(t, r.Header.Get("Content-Type"), "a bodyless request should not set Content-Type")

		body, err := readAll(r)
		require.NoError(t, err)
		assert.Empty(t, body)
	})

	client := httpsms.NewClient(server.URL)
	user, err := client.RotateUserAPIKey(t.Context(), token, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "new-secret-api-key", user.APIKey)
}

func TestClient_DecodesFieldValidationErrors(t *testing.T) {
	server := newTestServer(t, http.StatusUnprocessableEntity, map[string]any{
		"status":  "error",
		"message": "validation errors while sending message",
		"data": map[string][]string{
			"to": {"The to field is required"},
		},
	}, nil)

	client := httpsms.NewClient(server.URL)
	_, err := client.SendSMS(t.Context(), "token", httpsms.SendSMSParams{From: "+18005550199", Content: "hi"})
	require.Error(t, err)

	var apiErr *httpsms.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.StatusCode)
	assert.Equal(t, "validation errors while sending message", apiErr.Message)
	assert.Equal(t, []string{"The to field is required"}, apiErr.Fields["to"])
	assert.NotEmpty(t, apiErr.RequestID)
	assert.NotContains(t, apiErr.Error(), "token")
}

func TestClient_DecodesStringDataErrorWithoutFields(t *testing.T) {
	server := newTestServer(t, http.StatusUnauthorized, map[string]any{
		"status":  "error",
		"message": "You are not authorized to carry out this request.",
		"data":    "Make sure your API key is set in the [X-API-Key] header in the request",
	}, nil)

	client := httpsms.NewClient(server.URL)
	_, err := client.ListPhones(t.Context(), "token", httpsms.ListPhonesParams{})
	require.Error(t, err)

	var apiErr *httpsms.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "You are not authorized to carry out this request.", apiErr.Message)
	assert.Nil(t, apiErr.Fields)
}

func TestClient_RejectsOversizedResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Write a response far larger than the client's 2 MiB cap.
		_, _ = w.Write([]byte(`{"status":"success","message":"","data":[`))
		chunk := strings.Repeat("0", 1024)
		for i := 0; i < 3*1024; i++ { // ~3 MiB of padding
			_, _ = w.Write([]byte(chunk))
		}
		_, _ = w.Write([]byte(`]}`))
	}))
	t.Cleanup(server.Close)

	client := httpsms.NewClient(server.URL)
	_, err := client.ListPhones(t.Context(), "token", httpsms.ListPhonesParams{})
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "size")
}

func TestClient_MalformedResponseBodyIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	t.Cleanup(server.Close)

	client := httpsms.NewClient(server.URL)
	_, err := client.ListPhones(t.Context(), "token", httpsms.ListPhonesParams{})
	require.Error(t, err)
}

func TestClient_PropagatesContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(server.Close)

	client := httpsms.NewClient(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.ListPhones(ctx, "token", httpsms.ListPhonesParams{})
	require.Error(t, err)
}

func readAll(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	return io.ReadAll(r.Body)
}
