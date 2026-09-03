package httpsms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	// requestIDHeader is the header this client sets on every outgoing
	// request with a per-call, client-generated identifier, so a returned
	// APIError can be correlated with the client-side log line and trace
	// span that issued the call. The httpSMS API does not currently read or
	// echo this header back.
	requestIDHeader = "X-Request-Id"

	// maxResponseBytes bounds how much of a response body this client will
	// read, so a misbehaving or unexpectedly large upstream response cannot
	// exhaust memory.
	maxResponseBytes = 2 * 1024 * 1024 // 2 MiB

	// requestTimeout bounds the total time (dial, TLS, request, and
	// response) any single call to the httpSMS API is allowed to take, on
	// top of whatever deadline the caller's context already carries.
	requestTimeout = 15 * time.Second

	// dialTimeout bounds how long TCP connection establishment (DNS
	// resolution plus connect) may take for a single dial, independently
	// of the overall requestTimeout, so a slow or black-holed network path
	// fails fast instead of consuming the whole request budget on dialing
	// alone.
	dialTimeout = 5 * time.Second

	// tlsHandshakeTimeout bounds how long the TLS handshake may take once a
	// TCP connection is established.
	tlsHandshakeTimeout = 5 * time.Second

	// responseHeaderTimeout bounds how long this client waits for the
	// response status line and headers after the request (including its
	// body, if any) has been fully written, so a server that accepts a
	// connection but never responds cannot hold a call open until the
	// overall requestTimeout.
	responseHeaderTimeout = 10 * time.Second

	maxIdleConns        = 100
	maxIdleConnsPerHost = 10
	idleConnTimeout     = 90 * time.Second
)

// Client is the typed httpSMS API client used by every MCP tool. Every
// method takes a delegated API bearer token minted by the caller for this
// exact operation (never minted, cached, or inspected by the client) and
// the parameters that operation supports.
type Client interface {
	// ListPhones calls GET /v1/phones.
	ListPhones(ctx context.Context, token string, params ListPhonesParams) ([]Phone, error)

	// SendSMS calls POST /v1/messages/send.
	SendSMS(ctx context.Context, token string, params SendSMSParams) (Message, error)

	// ListMessageThreads calls GET /v1/message-threads.
	ListMessageThreads(ctx context.Context, token string, params ListMessageThreadsParams) ([]MessageThread, error)

	// ListThreadMessages calls GET /v1/messages.
	ListThreadMessages(ctx context.Context, token string, params ListThreadMessagesParams) ([]Message, error)

	// ListIncomingMessages calls GET /v1/messages/incoming.
	ListIncomingMessages(ctx context.Context, token string, params ListIncomingMessagesParams) ([]Message, error)

	// CreatePhoneAPIKey calls POST /v1/phone-api-keys.
	CreatePhoneAPIKey(ctx context.Context, token string, params CreatePhoneAPIKeyParams) (PhoneAPIKey, error)

	// RotateUserAPIKey calls DELETE /v1/users/{userID}/api-keys.
	RotateUserAPIKey(ctx context.Context, token string, userID string) (User, error)
}

// HTTPClient is the Client implementation calling the httpSMS HTTP API.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

var _ Client = (*HTTPClient)(nil)

// NewClient returns an *HTTPClient calling baseURL (for example
// "https://api.httpsms.com"). The returned client is bounded and makes a
// single attempt per call: an explicit overall request timeout plus
// separate dial, TLS handshake, and response header timeouts, a
// size-limited connection pool, OpenTelemetry context propagation through
// otelhttp.Transport (with query string values redacted from span
// attributes; see queryRedactingTransport), and no automatic retries.
// Retrying automatically would risk duplicating the side effect of a
// non-idempotent call such as sending an SMS, creating a phone API key, or
// rotating the user's primary API key.
func NewClient(baseURL string) *HTTPClient {
	transport := &http.Transport{
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		DialContext: (&net.Dialer{
			Timeout: dialTimeout,
		}).DialContext,
	}

	instrumented := otelhttp.NewTransport(&queryRestoringTransport{base: transport})

	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   requestTimeout,
			Transport: &queryRedactingTransport{next: instrumented},
		},
	}
}

// ListPhones calls GET /v1/phones.
func (c *HTTPClient) ListPhones(ctx context.Context, token string, params ListPhonesParams) ([]Phone, error) {
	query := url.Values{}
	setIntIfPositive(query, "skip", params.Skip)
	setStringIfNotEmpty(query, "query", params.Query)
	setIntIfPositive(query, "limit", params.Limit)

	var phones []Phone
	if err := c.do(ctx, token, http.MethodGet, "/v1/phones", query, nil, &phones); err != nil {
		return nil, err
	}
	return phones, nil
}

// messageSendRequest is the wire body for POST /v1/messages/send. Its JSON
// field names are a contract with api/pkg/requests.MessageSend and must not
// change independently of it.
type messageSendRequest struct {
	From        string     `json:"from"`
	To          string     `json:"to"`
	Content     string     `json:"content"`
	Attachments []string   `json:"attachments,omitempty"`
	Encrypted   bool       `json:"encrypted,omitempty"`
	RequestID   string     `json:"request_id,omitempty"`
	SendAt      *time.Time `json:"send_at,omitempty"`
}

// SendSMS calls POST /v1/messages/send.
func (c *HTTPClient) SendSMS(ctx context.Context, token string, params SendSMSParams) (Message, error) {
	body := messageSendRequest{
		From:        params.From,
		To:          params.To,
		Content:     params.Content,
		Attachments: params.Attachments,
		Encrypted:   params.Encrypted,
		RequestID:   params.RequestID,
		SendAt:      params.SendAt,
	}

	var message Message
	if err := c.do(ctx, token, http.MethodPost, "/v1/messages/send", nil, body, &message); err != nil {
		return Message{}, err
	}
	return message, nil
}

// ListMessageThreads calls GET /v1/message-threads.
func (c *HTTPClient) ListMessageThreads(ctx context.Context, token string, params ListMessageThreadsParams) ([]MessageThread, error) {
	query := url.Values{}
	setStringIfNotEmpty(query, "owner", params.Owner)
	setBoolPointer(query, "is_archived", params.IsArchived)
	setBoolIfTrue(query, "contacts", params.WithContacts)
	setStringIfNotEmpty(query, "query", params.Query)
	setIntIfPositive(query, "skip", params.Skip)
	setIntIfPositive(query, "limit", params.Limit)

	var threads []MessageThread
	if err := c.do(ctx, token, http.MethodGet, "/v1/message-threads", query, nil, &threads); err != nil {
		return nil, err
	}
	return threads, nil
}

// ListThreadMessages calls GET /v1/messages.
func (c *HTTPClient) ListThreadMessages(ctx context.Context, token string, params ListThreadMessagesParams) ([]Message, error) {
	query := url.Values{}
	setStringIfNotEmpty(query, "owner", params.Owner)
	setStringIfNotEmpty(query, "contact", params.Contact)
	setStringIfNotEmpty(query, "query", params.Query)
	setIntIfPositive(query, "skip", params.Skip)
	setIntIfPositive(query, "limit", params.Limit)

	var messages []Message
	if err := c.do(ctx, token, http.MethodGet, "/v1/messages", query, nil, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// ListIncomingMessages calls GET /v1/messages/incoming.
func (c *HTTPClient) ListIncomingMessages(ctx context.Context, token string, params ListIncomingMessagesParams) ([]Message, error) {
	query := url.Values{}
	setRepeated(query, "owners", params.Owners)
	setRepeated(query, "statuses", params.Statuses)
	setStringIfNotEmpty(query, "query", params.Query)
	setStringIfNotEmpty(query, "sort_by", params.SortBy)
	setBoolPointer(query, "sort_descending", params.SortDescending)
	setIntIfPositive(query, "skip", params.Skip)
	setIntIfPositive(query, "limit", params.Limit)

	var messages []Message
	if err := c.do(ctx, token, http.MethodGet, "/v1/messages/incoming", query, nil, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// phoneAPIKeyStoreRequest is the wire body for POST /v1/phone-api-keys. Its
// JSON field names are a contract with
// api/pkg/requests.PhoneAPIKeyStoreRequest and must not change
// independently of it.
type phoneAPIKeyStoreRequest struct {
	Name string `json:"name"`
}

// CreatePhoneAPIKey calls POST /v1/phone-api-keys.
func (c *HTTPClient) CreatePhoneAPIKey(ctx context.Context, token string, params CreatePhoneAPIKeyParams) (PhoneAPIKey, error) {
	body := phoneAPIKeyStoreRequest{Name: params.Name}

	var key PhoneAPIKey
	if err := c.do(ctx, token, http.MethodPost, "/v1/phone-api-keys", nil, body, &key); err != nil {
		return PhoneAPIKey{}, err
	}
	return key, nil
}

// RotateUserAPIKey calls DELETE /v1/users/{userID}/api-keys. userID is
// always the authenticated subject's own Firebase UID; callers must never
// accept it as untrusted tool input.
func (c *HTTPClient) RotateUserAPIKey(ctx context.Context, token string, userID string) (User, error) {
	path := "/v1/users/" + url.PathEscape(userID) + "/api-keys"

	var user User
	if err := c.do(ctx, token, http.MethodDelete, path, nil, nil, &user); err != nil {
		return User{}, err
	}
	return user, nil
}

// do issues a single, bounded HTTP request against the httpSMS API,
// authenticated with token, and decodes the response envelope's "data"
// field into output.
//
// input, when non-nil, is JSON-encoded as the request body and a
// "Content-Type: application/json" header is sent. output, when non-nil,
// receives the decoded "data" field of a successful response.
//
// do makes exactly one attempt: it never retries, so callers can safely use
// it for non-idempotent operations (sending an SMS, creating a phone API
// key, rotating the primary API key) without risking a duplicated side
// effect from a transport-level retry.
func (c *HTTPClient) do(
	ctx context.Context,
	token string,
	method string,
	path string,
	query url.Values,
	input any,
	output any,
) error {
	requestID := uuid.NewString()

	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("httpsms: cannot encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("httpsms: cannot build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set(requestIDHeader, requestID)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// http.Client errors may wrap the request URL (never a secret: the
		// bearer token is a header, not part of the URL) but never the
		// request body or headers, so it is safe to wrap here.
		return fmt.Errorf("httpsms: request [%s] failed: %w", requestID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return &APIError{StatusCode: resp.StatusCode, RequestID: requestID, Message: "cannot read httpSMS API response"}
	}
	if len(raw) > maxResponseBytes {
		return &APIError{StatusCode: resp.StatusCode, RequestID: requestID, Message: "httpSMS API response exceeded the maximum allowed size"}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp.StatusCode, requestID, raw)
	}

	if output == nil {
		return nil
	}

	var envelope Response[json.RawMessage]
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return &APIError{StatusCode: resp.StatusCode, RequestID: requestID, Message: "cannot decode httpSMS API response"}
	}

	if err := json.Unmarshal(envelope.Data, output); err != nil {
		return &APIError{StatusCode: resp.StatusCode, RequestID: requestID, Message: "cannot decode httpSMS API response data"}
	}

	return nil
}

// parseAPIError decodes a non-2xx httpSMS API response body into an
// *APIError. It never fails: a malformed or unexpected body still yields an
// *APIError with a generic message rather than an opaque decode error,
// since the caller already knows the call failed from the status code.
func parseAPIError(statusCode int, requestID string, raw []byte) error {
	apiErr := &APIError{StatusCode: statusCode, RequestID: requestID}

	var envelope struct {
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		apiErr.Message = "httpSMS API returned a malformed error response"
		return apiErr
	}

	apiErr.Message = envelope.Message
	if apiErr.Message == "" {
		apiErr.Message = "httpSMS API request failed"
	}

	var fields map[string][]string
	if len(envelope.Data) > 0 && json.Unmarshal(envelope.Data, &fields) == nil {
		apiErr.Fields = fields
	}

	return apiErr
}

// setIntIfPositive sets key to value's decimal string form only when value
// is greater than zero, so a caller's zero value (indistinguishable from
// "not set") is omitted and the API applies its own default.
func setIntIfPositive(values url.Values, key string, value int) {
	if value > 0 {
		values.Set(key, strconv.Itoa(value))
	}
}

// setStringIfNotEmpty sets key to value only when value is non-empty.
func setStringIfNotEmpty(values url.Values, key string, value string) {
	if value != "" {
		values.Set(key, value)
	}
}

// setBoolPointer sets key to value's string form only when value is
// non-nil, so an unset optional filter is omitted rather than sent as
// "false".
func setBoolPointer(values url.Values, key string, value *bool) {
	if value != nil {
		values.Set(key, strconv.FormatBool(*value))
	}
}

// setBoolIfTrue sets key to "true" only when value is true, so a filter
// whose zero value already matches the API's default is omitted.
func setBoolIfTrue(values url.Values, key string, value bool) {
	if value {
		values.Set(key, "true")
	}
}

// setRepeated adds one query value per item in items under key, matching
// the repeated-key encoding the API's query binder expects for []string
// fields (for example "owners=a&owners=b").
func setRepeated(values url.Values, key string, items []string) {
	for _, item := range items {
		values.Add(key, item)
	}
}
