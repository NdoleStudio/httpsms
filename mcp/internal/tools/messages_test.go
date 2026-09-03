package tools_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
	"github.com/NdoleStudio/httpsms/mcp/internal/oauth"
	"github.com/NdoleStudio/httpsms/mcp/internal/tools"
)

const (
	testMCPIssuer    = "https://mcp.httpsms.com"
	testMCPAudience  = "https://mcp.httpsms.com/mcp"
	testAPIAudience  = "https://api.httpsms.com"
	testSigningKeyID = "test-key-1"
	testUserID       = "user-id"
	testUserEmail    = "user@example.com"
	testAPITokenTTL  = 2 * time.Minute
)

// allScopes are every scope required by any tool registered by
// tools.Register, used to build an authorized context for tests that are
// not specifically exercising scope denial.
var allScopes = []string{
	auth.ScopePhonesRead,
	auth.ScopeMessagesRead,
	auth.ScopeMessagesSend,
	auth.ScopePhoneAPIKeysWrite,
	auth.ScopeUserAPIKeyRotate,
}

// --- test doubles -----------------------------------------------------

// stubClient is a httpsms.Client test double that records every call made
// to it and returns pre-configured results, so tests can assert both on
// what a tool returned and on exactly which downstream API calls (if any)
// it made.
type stubClient struct {
	listPhonesCalls  []stubCall[httpsms.ListPhonesParams]
	listPhonesResult []httpsms.Phone
	listPhonesErr    error

	sendSMSCalls  []stubCall[httpsms.SendSMSParams]
	sendSMSResult httpsms.Message
	sendSMSErr    error

	listThreadsCalls  []stubCall[httpsms.ListMessageThreadsParams]
	listThreadsResult []httpsms.MessageThread
	listThreadsErr    error

	listThreadMessagesCalls  []stubCall[httpsms.ListThreadMessagesParams]
	listThreadMessagesResult []httpsms.Message
	listThreadMessagesErr    error

	listIncomingCalls  []stubCall[httpsms.ListIncomingMessagesParams]
	listIncomingResult []httpsms.Message
	listIncomingErr    error

	createKeyCalls  []stubCall[httpsms.CreatePhoneAPIKeyParams]
	createKeyResult httpsms.PhoneAPIKey
	createKeyErr    error

	rotateCalls  []stubCall[string]
	rotateResult httpsms.User
	rotateErr    error
}

// stubCall records one call's delegated token and parameters.
type stubCall[P any] struct {
	Token  string
	Params P
}

var _ httpsms.Client = (*stubClient)(nil)

func (s *stubClient) ListPhones(_ context.Context, token string, params httpsms.ListPhonesParams) ([]httpsms.Phone, error) {
	s.listPhonesCalls = append(s.listPhonesCalls, stubCall[httpsms.ListPhonesParams]{Token: token, Params: params})
	return s.listPhonesResult, s.listPhonesErr
}

func (s *stubClient) SendSMS(_ context.Context, token string, params httpsms.SendSMSParams) (httpsms.Message, error) {
	s.sendSMSCalls = append(s.sendSMSCalls, stubCall[httpsms.SendSMSParams]{Token: token, Params: params})
	return s.sendSMSResult, s.sendSMSErr
}

func (s *stubClient) ListMessageThreads(_ context.Context, token string, params httpsms.ListMessageThreadsParams) ([]httpsms.MessageThread, error) {
	s.listThreadsCalls = append(s.listThreadsCalls, stubCall[httpsms.ListMessageThreadsParams]{Token: token, Params: params})
	return s.listThreadsResult, s.listThreadsErr
}

func (s *stubClient) ListThreadMessages(_ context.Context, token string, params httpsms.ListThreadMessagesParams) ([]httpsms.Message, error) {
	s.listThreadMessagesCalls = append(s.listThreadMessagesCalls, stubCall[httpsms.ListThreadMessagesParams]{Token: token, Params: params})
	return s.listThreadMessagesResult, s.listThreadMessagesErr
}

func (s *stubClient) ListIncomingMessages(_ context.Context, token string, params httpsms.ListIncomingMessagesParams) ([]httpsms.Message, error) {
	s.listIncomingCalls = append(s.listIncomingCalls, stubCall[httpsms.ListIncomingMessagesParams]{Token: token, Params: params})
	return s.listIncomingResult, s.listIncomingErr
}

func (s *stubClient) CreatePhoneAPIKey(_ context.Context, token string, params httpsms.CreatePhoneAPIKeyParams) (httpsms.PhoneAPIKey, error) {
	s.createKeyCalls = append(s.createKeyCalls, stubCall[httpsms.CreatePhoneAPIKeyParams]{Token: token, Params: params})
	return s.createKeyResult, s.createKeyErr
}

func (s *stubClient) RotateUserAPIKey(_ context.Context, token string, userID string) (httpsms.User, error) {
	s.rotateCalls = append(s.rotateCalls, stubCall[string]{Token: token, Params: userID})
	return s.rotateResult, s.rotateErr
}

// totalCalls reports how many downstream API calls s has recorded across
// every method, so a test can assert that a denied or invalid call never
// reached the httpSMS API.
func (s *stubClient) totalCalls() int {
	return len(s.listPhonesCalls) + len(s.sendSMSCalls) + len(s.listThreadsCalls) +
		len(s.listThreadMessagesCalls) + len(s.listIncomingCalls) +
		len(s.createKeyCalls) + len(s.rotateCalls)
}

// --- test fixtures ------------------------------------------------------

// newTestKeySet builds a KeySet with test issuer/audiences already
// configured, mirroring internal/auth's own test fixture (it cannot be
// imported directly: internal/auth's fixture lives in package auth_test).
func newTestKeySet(t *testing.T) *auth.KeySet {
	t.Helper()

	keys, err := auth.NewKeySet(newTestRSAPrivateKeyPEM(t), testSigningKeyID)
	require.NoError(t, err)
	require.NoError(t, keys.Configure(testMCPIssuer, testMCPAudience, testAPIAudience))
	return keys
}

// contextWithPrincipal returns a context carrying a verified MCP bearer
// token for a principal holding scopes, exactly as a real request context
// would carry one after passing through
// mcpauth.RequireBearerToken(verifier.VerifyMCPToken, ...). It does this by
// actually running that middleware against a synthetic HTTP request and
// capturing the context it produces, rather than reaching into any
// unexported context key.
func contextWithPrincipal(t *testing.T, keys *auth.KeySet, scopes []string) context.Context {
	t.Helper()

	raw, err := keys.SignMCPAccessToken(auth.Principal{UserID: testUserID, Email: testUserEmail}, "test-client", scopes, time.Minute)
	require.NoError(t, err)

	return contextFromBearerToken(t, keys, raw)
}

// contextFromBearerToken runs the real bearer-token middleware against raw
// and returns the resulting request context, whatever it turns out to
// contain (or not contain, for an invalid token).
func contextFromBearerToken(t *testing.T, keys *auth.KeySet, raw string) context.Context {
	t.Helper()

	verifier := auth.NewVerifier(keys)
	middleware := mcpauth.RequireBearerToken(verifier.VerifyMCPToken, nil)

	var captured context.Context
	handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r.Context()
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	if raw != "" {
		req.Header.Set("Authorization", "Bearer "+raw)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if captured == nil {
		// Authentication was rejected before reaching the inner handler
		// (e.g. no token at all): return the plain request context, which
		// carries no TokenInfo, exactly like a real unauthenticated call.
		return req.Context()
	}
	return captured
}

// testConfirmationTTL is the rotation-confirmation-handle TTL used by
// every test session; it must be short enough that
// TestRotateUserAPIKeyConfirmationHandleExpires can advance past it with a
// small, fast miniredis.FastForward call.
const testConfirmationTTL = 5 * time.Minute

// newSession registers every tool against api using keys and apiTokenTTL,
// connects an in-memory client/server pair rooted at ctx (so every tool
// call in the resulting session observes whatever principal/scopes ctx
// carries), and returns the client session plus a cleanup func. It backs
// rotate_user_api_key's confirmation handles with a fresh, throwaway
// miniredis instance: tests that need to control or inspect that store
// directly (expiry, replay) should use newSessionWithStore instead.
func newSession(t *testing.T, ctx context.Context, keys *auth.KeySet, api httpsms.Client) *mcp.ClientSession {
	t.Helper()

	store, _ := newTestConfirmationStore(t)
	return newSessionWithStore(t, ctx, keys, api, store, testConfirmationTTL)
}

// newTestConfirmationStore starts an in-memory miniredis server and returns
// an oauth.Store backed by it along with the miniredis handle, for tests
// that need to fast-forward time or otherwise inspect confirmation state
// directly.
func newTestConfirmationStore(t *testing.T) (oauth.Store, *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return oauth.NewRedisStore(client), server
}

// newSessionWithStore is newSession with an explicit confirmation store and
// TTL, for tests that drive rotate_user_api_key's confirmation flow.
func newSessionWithStore(t *testing.T, ctx context.Context, keys *auth.KeySet, api httpsms.Client, store oauth.Store, confirmationTTL time.Duration) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "httpsms-mcp-test", Version: "test"}, nil)
	tools.Register(server, keys, api, testAPITokenTTL, store, confirmationTTL)

	t1, t2 := mcp.NewInMemoryTransports()
	_, err := server.Connect(ctx, t1, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(context.Background(), t2, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = session.Close() })
	return session
}

// callTool calls name with arguments on session and decodes its structured
// output into out. It fails the test if the call is a protocol error or a
// tool-level error.
func callTool(t *testing.T, session *mcp.ClientSession, name string, arguments map[string]any, out any) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	require.NoError(t, err, "tools/call must not be a protocol error")
	require.False(t, result.IsError, "expected a successful tool result")

	if out != nil {
		raw, err := json.Marshal(result.StructuredContent)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, out))
	}
	return result
}

// callToolExpectingError calls name with arguments and asserts the result
// is a tool-level error (not a protocol error).
func callToolExpectingError(t *testing.T, session *mcp.ClientSession, name string, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	require.NoError(t, err, "tools/call must not be a protocol error")
	require.True(t, result.IsError, "expected a tool-level error result")
	return result
}

// resultText concatenates the text of every TextContent block in result,
// for asserting on tool error messages.
func resultText(result *mcp.CallToolResult) string {
	var text string
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text += tc.Text
		}
	}
	return text
}

// toolByName returns the *mcp.Tool named name from session's tool list.
func toolByName(t *testing.T, session *mcp.ClientSession, name string) *mcp.Tool {
	t.Helper()

	for tool, err := range session.Tools(context.Background(), nil) {
		require.NoError(t, err)
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q was not registered", name)
	return nil
}

// schemaProperty returns schema's "properties"."name" entry as a
// map[string]any, for asserting on inferred/customized JSON schema
// constraints from the client's point of view (a map[string]any, per
// mcp.Tool.InputSchema's documented client-side representation).
func schemaProperty(t *testing.T, schema any, name string) map[string]any {
	t.Helper()

	m, ok := schema.(map[string]any)
	require.True(t, ok, "schema must decode to a map[string]any")
	props, ok := m["properties"].(map[string]any)
	require.True(t, ok, "schema must have a properties map")
	prop, ok := props[name].(map[string]any)
	require.True(t, ok, "schema must have a %q property", name)
	return prop
}

func schemaRequired(t *testing.T, schema any) []string {
	t.Helper()

	m, ok := schema.(map[string]any)
	require.True(t, ok, "schema must decode to a map[string]any")
	raw, ok := m["required"].([]any)
	if !ok {
		return nil
	}
	required := make([]string, len(raw))
	for i, v := range raw {
		required[i] = v.(string)
	}
	return required
}

// --- registration ---------------------------------------------------------

func TestRegisterRegistersExactlySevenTools(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	session := newSession(t, ctx, keys, &stubClient{})

	var names []string
	for tool, err := range session.Tools(context.Background(), nil) {
		require.NoError(t, err)
		names = append(names, tool.Name)
	}
	sort.Strings(names)

	assert.Equal(t, []string{
		"create_phone_api_key",
		"list_incoming_messages",
		"list_message_threads",
		"list_phones",
		"list_thread_messages",
		"rotate_user_api_key",
		"send_sms",
	}, names)
}

func TestListPhonesToolIsMarkedReadOnly(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	session := newSession(t, ctx, keys, &stubClient{})

	tool := toolByName(t, session, "list_phones")
	require.NotNil(t, tool.Annotations)
	assert.True(t, tool.Annotations.ReadOnlyHint)
}

func TestSendSMSToolIsMarkedDestructiveAndNotIdempotent(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	session := newSession(t, ctx, keys, &stubClient{})

	tool := toolByName(t, session, "send_sms")
	require.NotNil(t, tool.Annotations)
	assert.False(t, tool.Annotations.ReadOnlyHint)
	assert.False(t, tool.Annotations.IdempotentHint)
	require.NotNil(t, tool.Annotations.DestructiveHint)
	assert.True(t, *tool.Annotations.DestructiveHint)
}

// --- schemas ---------------------------------------------------------

func TestListMessageThreadsSchemaEnforcesMaxLimitOfTwenty(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	session := newSession(t, ctx, keys, &stubClient{})

	tool := toolByName(t, session, "list_message_threads")
	limit := schemaProperty(t, tool.InputSchema, "limit")
	assert.Equal(t, float64(20), limit["maximum"])

	assert.Contains(t, schemaRequired(t, tool.InputSchema), "owner")
}

func TestListThreadMessagesSchemaRequiresOwnerAndContact(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	session := newSession(t, ctx, keys, &stubClient{})

	tool := toolByName(t, session, "list_thread_messages")
	required := schemaRequired(t, tool.InputSchema)
	assert.Contains(t, required, "owner")
	assert.Contains(t, required, "contact")
	assert.NotContains(t, required, "query")
}

func TestSendSMSSchemaRequiresFromToContentOnly(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	session := newSession(t, ctx, keys, &stubClient{})

	tool := toolByName(t, session, "send_sms")
	required := schemaRequired(t, tool.InputSchema)
	assert.ElementsMatch(t, []string{"from", "to", "content"}, required)

	m, ok := tool.InputSchema.(map[string]any)
	require.True(t, ok)
	props, ok := m["properties"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, props, "sim", "send_sms must not expose an unsupported sim field")
}

func TestListMessageThreadsRejectsLimitAboveTwenty(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "list_message_threads", map[string]any{
		"owner": "+18005550199",
		"limit": 21,
	})
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls(), "an invalid request must never reach the httpSMS API")
}

func TestListThreadMessagesRejectsMissingOwnerAndContact(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "list_thread_messages", map[string]any{
		"query": "hello",
	})
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls())
}

// --- list_phones ---------------------------------------------------------

func TestListPhonesReturnsStableStructuredContent(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)

	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	stub := &stubClient{listPhonesResult: []httpsms.Phone{
		{ID: "phone-1", PhoneNumber: "+18005550199", SIM: "DEFAULT", MessagesPerMinute: 10, CreatedAt: createdAt, UpdatedAt: createdAt},
	}}
	session := newSession(t, ctx, keys, stub)

	var out tools.ListPhonesOutput
	callTool(t, session, "list_phones", map[string]any{"query": "8005550199", "limit": 5}, &out)

	require.Len(t, out.Phones, 1)
	assert.Equal(t, "phone-1", out.Phones[0].ID)
	assert.Equal(t, "+18005550199", out.Phones[0].PhoneNumber)
	assert.Equal(t, "DEFAULT", out.Phones[0].SIM)
	assert.Equal(t, 1, out.Count)

	require.Len(t, stub.listPhonesCalls, 1)
	call := stub.listPhonesCalls[0]
	assert.Equal(t, "8005550199", call.Params.Query)
	assert.Equal(t, 5, call.Params.Limit)
	assert.NotEmpty(t, call.Token)
}

func TestListPhonesMintsAPhonesReadScopedDelegationTokenBoundToGetPhones(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	callTool(t, session, "list_phones", nil, new(tools.ListPhonesOutput))

	require.Len(t, stub.listPhonesCalls, 1)
	assertDelegationToken(t, keys, stub.listPhonesCalls[0].Token, http.MethodGet, "/v1/phones", []string{auth.ScopePhonesRead})
}

func TestListPhonesDeniedWithoutPhonesReadScope(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, []string{auth.ScopeMessagesRead}) // valid token, wrong scope
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "list_phones", nil)
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls(), "a scope-denied call must never reach the httpSMS API")
}

func TestListPhonesDeniedWithoutAnyToken(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := context.Background() // no verified MCP bearer token at all
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "list_phones", nil)
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls())
}

func TestListPhonesSurfacesAPIErrorAsToolError(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{listPhonesErr: &httpsms.APIError{StatusCode: http.StatusTooManyRequests, Message: "rate limited", RequestID: "req-1"}}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "list_phones", nil)
	assert.Contains(t, resultText(result), "rate limited")
}

// --- send_sms ---------------------------------------------------------

func TestSendSMSForwardsAllSupportedOptionalFields(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)

	sendAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	stub := &stubClient{sendSMSResult: httpsms.Message{ID: "message-1", Owner: "+18005550199", Contact: "+18005550100", Content: "hello", Status: "pending"}}
	session := newSession(t, ctx, keys, stub)

	var out tools.SendSMSOutput
	callTool(t, session, "send_sms", map[string]any{
		"from":        "+18005550199",
		"to":          "+18005550100",
		"content":     "hello",
		"attachments": []any{"https://example.com/image.jpg"},
		"encrypted":   true,
		"request_id":  "req-123",
		"send_at":     sendAt.Format(time.RFC3339),
	}, &out)

	assert.Equal(t, "message-1", out.Message.ID)

	require.Len(t, stub.sendSMSCalls, 1)
	params := stub.sendSMSCalls[0].Params
	assert.Equal(t, "+18005550199", params.From)
	assert.Equal(t, "+18005550100", params.To)
	assert.Equal(t, "hello", params.Content)
	assert.Equal(t, []string{"https://example.com/image.jpg"}, params.Attachments)
	assert.True(t, params.Encrypted)
	assert.Equal(t, "req-123", params.RequestID)
	require.NotNil(t, params.SendAt)
	assert.True(t, sendAt.Equal(*params.SendAt))
}

func TestSendSMSMintsAMessagesSendScopedDelegationTokenBoundToPostSend(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	callTool(t, session, "send_sms", map[string]any{"from": "+18005550199", "to": "+18005550100", "content": "hi"}, new(tools.SendSMSOutput))

	require.Len(t, stub.sendSMSCalls, 1)
	assertDelegationToken(t, keys, stub.sendSMSCalls[0].Token, http.MethodPost, "/v1/messages/send", []string{auth.ScopeMessagesSend})
}

func TestSendSMSDeniedWithoutMessagesSendScope(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, []string{auth.ScopePhonesRead})
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "send_sms", map[string]any{"from": "+18005550199", "to": "+18005550100", "content": "hi"})
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls())
}

func TestSendSMSSurfacesAPIErrorAsToolError(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{sendSMSErr: &httpsms.APIError{StatusCode: http.StatusPaymentRequired, Message: "insufficient balance"}}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "send_sms", map[string]any{"from": "+18005550199", "to": "+18005550100", "content": "hi"})
	assert.Contains(t, resultText(result), "insufficient balance")
}

// --- list_message_threads ---------------------------------------------------------

func TestListMessageThreadsForwardsFilters(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{listThreadsResult: []httpsms.MessageThread{{ID: "thread-1", Owner: "+18005550199", Contact: "+18005550100"}}}
	session := newSession(t, ctx, keys, stub)

	var out tools.ListMessageThreadsOutput
	callTool(t, session, "list_message_threads", map[string]any{
		"owner":         "+18005550199",
		"is_archived":   true,
		"with_contacts": true,
		"query":         "friend",
		"skip":          2,
		"limit":         10,
	}, &out)

	require.Len(t, out.Threads, 1)
	assert.Equal(t, 1, out.Count)

	require.Len(t, stub.listThreadsCalls, 1)
	params := stub.listThreadsCalls[0].Params
	assert.Equal(t, "+18005550199", params.Owner)
	require.NotNil(t, params.IsArchived)
	assert.True(t, *params.IsArchived)
	assert.True(t, params.WithContacts)
	assert.Equal(t, "friend", params.Query)
	assert.Equal(t, 2, params.Skip)
	assert.Equal(t, 10, params.Limit)

	assertDelegationToken(t, keys, stub.listThreadsCalls[0].Token, http.MethodGet, "/v1/message-threads", []string{auth.ScopeMessagesRead})
}

func TestListMessageThreadsDeniedWithoutMessagesReadScope(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, []string{auth.ScopePhonesRead})
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "list_message_threads", map[string]any{"owner": "+18005550199"})
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls())
}

// --- list_thread_messages ---------------------------------------------------------

func TestListThreadMessagesForwardsFilters(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{listThreadMessagesResult: []httpsms.Message{{ID: "message-1", Owner: "+18005550199", Contact: "+18005550100", Content: "hi"}}}
	session := newSession(t, ctx, keys, stub)

	var out tools.ListThreadMessagesOutput
	callTool(t, session, "list_thread_messages", map[string]any{
		"owner":   "+18005550199",
		"contact": "+18005550100",
		"query":   "hi",
		"skip":    1,
		"limit":   5,
	}, &out)

	require.Len(t, out.Messages, 1)
	assert.Equal(t, 1, out.Count)

	require.Len(t, stub.listThreadMessagesCalls, 1)
	params := stub.listThreadMessagesCalls[0].Params
	assert.Equal(t, "+18005550199", params.Owner)
	assert.Equal(t, "+18005550100", params.Contact)
	assert.Equal(t, "hi", params.Query)
	assert.Equal(t, 1, params.Skip)
	assert.Equal(t, 5, params.Limit)

	assertDelegationToken(t, keys, stub.listThreadMessagesCalls[0].Token, http.MethodGet, "/v1/messages", []string{auth.ScopeMessagesRead})
}

func TestListThreadMessagesDeniedWithoutMessagesReadScope(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, []string{auth.ScopePhonesRead})
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "list_thread_messages", map[string]any{"owner": "+18005550199", "contact": "+18005550100"})
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls())
}

// --- list_incoming_messages ---------------------------------------------------------

func TestListIncomingMessagesCallsTheDedicatedIncomingEndpoint(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{listIncomingResult: []httpsms.Message{{ID: "message-1", Type: "mobile-originated", Content: "hi"}}}
	session := newSession(t, ctx, keys, stub)

	var out tools.ListIncomingMessagesOutput
	callTool(t, session, "list_incoming_messages", map[string]any{
		"owners":          []any{"+18005550199"},
		"statuses":        []any{"received"},
		"query":           "hi",
		"sort_by":         "order_timestamp",
		"sort_descending": true,
		"skip":            0,
		"limit":           25,
	}, &out)

	require.Len(t, out.Messages, 1)
	assert.Equal(t, 1, out.Count)

	require.Len(t, stub.listIncomingCalls, 1)
	params := stub.listIncomingCalls[0].Params
	assert.Equal(t, []string{"+18005550199"}, params.Owners)
	assert.Equal(t, []string{"received"}, params.Statuses)
	assert.Equal(t, "hi", params.Query)
	assert.Equal(t, "order_timestamp", params.SortBy)
	require.NotNil(t, params.SortDescending)
	assert.True(t, *params.SortDescending)
	assert.Equal(t, 25, params.Limit)

	assertDelegationToken(t, keys, stub.listIncomingCalls[0].Token, http.MethodGet, "/v1/messages/incoming", []string{auth.ScopeMessagesRead})

	// This tool must never call the CAPTCHA-protected general search route:
	// the stub only implements ListIncomingMessages, so any use of a
	// different underlying route would have to go through it too. Assert
	// exactly one call was made overall.
	assert.Equal(t, 1, stub.totalCalls())
}

func TestListIncomingMessagesDeniedWithoutMessagesReadScope(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, []string{auth.ScopePhonesRead})
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "list_incoming_messages", nil)
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls())
}

func TestListIncomingMessagesSurfacesAPIErrorAsToolError(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{listIncomingErr: &httpsms.APIError{StatusCode: http.StatusInternalServerError, Message: "httpSMS API request failed"}}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "list_incoming_messages", nil)
	assert.Contains(t, resultText(result), "httpSMS API request failed")
}

// --- helpers shared across tool tests ---------------------------------------------------------

// assertDelegationToken verifies raw is an API delegation token minted by
// keys for exactly method/path and carrying exactly scopes -- proving each
// tool mints a fresh, narrowly-bound token per call rather than reusing or
// widening one.
func assertDelegationToken(t *testing.T, keys *auth.KeySet, raw string, method string, path string, scopes []string) {
	t.Helper()

	claims := parseDelegationClaims(t, raw, keys)
	assert.Equal(t, method, claims.Method)
	assert.Equal(t, path, claims.Path)
	assert.Equal(t, scopes, claims.Scopes)
	assert.Equal(t, testUserID, claims.Subject)
	require.Len(t, claims.Audience, 1)
	assert.Equal(t, testAPIAudience, claims.Audience[0])
}

// parseDelegationClaims verifies raw against keys' own public key and
// returns its claims, failing the test if raw does not parse or verify.
func parseDelegationClaims(t *testing.T, raw string, keys *auth.KeySet) *auth.AccessClaims {
	t.Helper()

	claims := new(auth.AccessClaims)
	token, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return keys.PublicKey(), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	require.NoError(t, err)
	require.True(t, token.Valid)
	return claims
}

// newTestRSAPrivateKeyPEM generates a throwaway 2048-bit RSA private key
// encoded as PKCS#1 PEM, for use only in tests.
func newTestRSAPrivateKeyPEM(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}
