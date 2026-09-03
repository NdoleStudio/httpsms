package tools_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
	"github.com/NdoleStudio/httpsms/mcp/internal/oauth"
	"github.com/NdoleStudio/httpsms/mcp/internal/tools"
)

// --- create_phone_api_key ---------------------------------------------------------

func TestCreatePhoneAPIKeyForwardsOnlyNameAndReturnsTheSecretOnce(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	stub := &stubClient{createKeyResult: httpsms.PhoneAPIKey{
		ID: "key-1", Name: "android-phone", PhoneNumbers: []string{"+18005550199"}, APIKey: "phone-api-key-secret", CreatedAt: createdAt, UpdatedAt: createdAt,
	}}
	session := newSession(t, ctx, keys, stub)

	var out tools.CreatePhoneAPIKeyOutput
	result := callTool(t, session, "create_phone_api_key", map[string]any{"name": "android-phone"}, &out)

	assert.Equal(t, "key-1", out.ID)
	assert.Equal(t, "android-phone", out.Name)
	assert.Equal(t, "phone-api-key-secret", out.APIKey)
	assert.True(t, out.Sensitive)
	assert.NotEmpty(t, resultText(result), "the result must instruct the user to store the key immediately")

	require.Len(t, stub.createKeyCalls, 1)
	assert.Equal(t, "android-phone", stub.createKeyCalls[0].Params.Name)
	assertDelegationToken(t, keys, stub.createKeyCalls[0].Token, http.MethodPost, "/v1/phone-api-keys", []string{auth.ScopePhoneAPIKeysWrite})
}

func TestCreatePhoneAPIKeyDeniedWithoutScope(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, []string{auth.ScopePhonesRead})
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "create_phone_api_key", map[string]any{"name": "android-phone"})
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls(), "a scope-denied call must never reach the httpSMS API")
}

func TestCreatePhoneAPIKeyDeniedWithoutAnyToken(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := context.Background()
	stub := &stubClient{}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "create_phone_api_key", map[string]any{"name": "android-phone"})
	assert.NotEmpty(t, resultText(result))
	assert.Equal(t, 0, stub.totalCalls())
}

func TestCreatePhoneAPIKeySurfacesAPIErrorAsToolError(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{createKeyErr: &httpsms.APIError{StatusCode: http.StatusUnprocessableEntity, Message: "name is required"}}
	session := newSession(t, ctx, keys, stub)

	result := callToolExpectingError(t, session, "create_phone_api_key", map[string]any{"name": ""})
	assert.Contains(t, resultText(result), "name is required")
}

func TestCreatePhoneAPIKeyToolIsMarkedNotIdempotentAndNotDestructive(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	session := newSession(t, ctx, keys, &stubClient{})

	tool := toolByName(t, session, "create_phone_api_key")
	require.NotNil(t, tool.Annotations)
	assert.False(t, tool.Annotations.ReadOnlyHint)
	assert.False(t, tool.Annotations.IdempotentHint)
	if tool.Annotations.DestructiveHint != nil {
		assert.False(t, *tool.Annotations.DestructiveHint)
	}
}

// TestCreatePhoneAPIKeyNeverLeaksSecretOutsideStructuredResult asserts the
// minted secret appears only in the tool's structured result, never on
// stdout/stderr (the only "logs" this service can currently produce
// mid-request; see observability.New for the service-wide JSON logger).
func TestCreatePhoneAPIKeyNeverLeaksSecretOutsideStructuredResult(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	const secret = "unique-phone-api-key-secret-4b9f9e6c-do-not-log"
	stub := &stubClient{createKeyResult: httpsms.PhoneAPIKey{ID: "key-1", Name: "android-phone", APIKey: secret}}
	session := newSession(t, ctx, keys, stub)

	var out tools.CreatePhoneAPIKeyOutput
	captured := captureStdoutStderr(t, func() {
		callTool(t, session, "create_phone_api_key", map[string]any{"name": "android-phone"}, &out)
	})

	require.Equal(t, secret, out.APIKey, "the structured result is the one place the secret must appear")
	assert.NotContains(t, captured, secret, "the secret must never be written to stdout or stderr")
}

// --- rotate_user_api_key: confirmation lifecycle ---------------------------------------------------------

// newRotateSession builds a client/server session for rotate_user_api_key
// with the client's automatic multi-round-trip (MRTR) retry middleware
// disabled (see mcp.MultiRoundTripOptions.Disabled), so CallTool returns
// the server's raw per-round-trip *mcp.CallToolResult -- InputRequests,
// RequestState, and NeedsInput() -- instead of transparently completing an
// entire confirm-then-rotate dance in a single call. This mirrors a client
// that cannot complete an MRTR elicitation at all (the "legacy" case this
// tool must also support) while giving every test full, explicit control
// over each individual round trip.
func newRotateSession(t *testing.T, ctx context.Context, keys *auth.KeySet, api httpsms.Client, store oauth.Store, confirmationTTL time.Duration) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "httpsms-mcp-test", Version: "test"}, nil)
	tools.Register(server, keys, api, testAPITokenTTL, store, confirmationTTL)

	t1, t2 := mcp.NewInMemoryTransports()
	_, err := server.Connect(ctx, t1, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, &mcp.ClientOptions{
		MultiRoundTrip: &mcp.MultiRoundTripOptions{Disabled: true},
	})
	session, err := client.Connect(context.Background(), t2, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = session.Close() })
	return session
}

// acceptedConfirmation is the InputResponses value a client sends back to
// accept rotate_user_api_key's confirm_rotation elicitation.
func acceptedConfirmation() *mcp.ElicitResult {
	return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirmed": true}}
}

func TestRotateUserAPIKeyDeniedWithoutScope(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, []string{auth.ScopePhonesRead})
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err, "tools/call must not be a protocol error")
	require.True(t, result.IsError)
	assert.Equal(t, 0, stub.totalCalls(), "a scope-denied call must never reach the httpSMS API")
}

func TestRotateUserAPIKeyDeniedWithoutAnyToken(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := context.Background()
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Equal(t, 0, stub.totalCalls())
}

// TestRotateUserAPIKeyFirstCallAsksForConfirmationAndNeverCallsTheAPI is the
// direct analogue of the task-8 brief's illustrative handler-level
// assertion, exercised end-to-end through a real tools/call round trip.
func TestRotateUserAPIKeyFirstCallAsksForConfirmationAndNeverCallsTheAPI(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err, "tools/call must not be a protocol error")
	require.False(t, result.IsError)
	require.True(t, result.NeedsInput(), "the first call must ask for confirmation, not rotate immediately")
	require.Contains(t, result.InputRequests, "confirm_rotation")
	elicit, ok := result.InputRequests["confirm_rotation"].(*mcp.ElicitParams)
	require.True(t, ok)
	assert.NotEmpty(t, elicit.Message)
	assert.Contains(t, elicit.Message, "stop working", "the elicitation message must warn the current key will stop working")
	assert.NotEmpty(t, result.RequestState)
	assert.Nil(t, result.StructuredContent)
	assert.Equal(t, 0, stub.totalCalls(), "the API must never be called before confirmation")
}

// TestRotateUserAPIKeyMRTRAcceptedConfirmationRotatesExactlyOnce drives the
// full MRTR round trip manually: an initial call, then a retry echoing back
// an accepted confirm_rotation response and the RequestState handle.
func TestRotateUserAPIKeyMRTRAcceptedConfirmationRotatesExactlyOnce(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateResult: httpsms.User{ID: testUserID, Email: testUserEmail, APIKey: "new-primary-api-key"}}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	require.True(t, first.NeedsInput())
	handle := first.RequestState
	require.NotEmpty(t, handle)

	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:           "rotate_user_api_key",
		InputResponses: mcp.InputResponseMap{"confirm_rotation": acceptedConfirmation()},
		RequestState:   handle,
	})
	require.NoError(t, err)
	require.False(t, second.IsError, resultText(second))
	require.False(t, second.NeedsInput())

	var out tools.RotateUserAPIKeyOutput
	require.NoError(t, decodeStructuredContent(second, &out))
	assert.Equal(t, "new-primary-api-key", out.User.APIKey)
	assert.True(t, out.Sensitive, "the rotated key must be explicitly marked sensitive, like create_phone_api_key's output")
	assert.NotEmpty(t, out.Warning)

	require.Len(t, stub.rotateCalls, 1)
	assert.Equal(t, testUserID, stub.rotateCalls[0].Params, "rotation must always target the authenticated principal's own user ID")
	assertDelegationToken(t, keys, stub.rotateCalls[0].Token, http.MethodDelete, "/v1/users/"+testUserID+"/api-keys", []string{auth.ScopeUserAPIKeyRotate})
}

// TestRotateUserAPIKeyLegacyConfirmationHandleResultMarksNewKeySensitive
// asserts a successful rotation's result -- both its structured output and
// its human-readable text content -- explicitly identifies the brand-new
// primary API key as a sensitive, one-time value, matching
// create_phone_api_key's CreatePhoneAPIKeyOutput.Sensitive/text pairing:
// callers must be told, in both channels, to store it now because it will
// never be shown again.
func TestRotateUserAPIKeyLegacyConfirmationHandleResultMarksNewKeySensitive(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateResult: httpsms.User{ID: testUserID, APIKey: "new-primary-api-key"}}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	handle := first.RequestState
	require.NotEmpty(t, handle)

	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"confirmation_handle": handle},
	})
	require.NoError(t, err)
	require.False(t, second.IsError, resultText(second))

	var out tools.RotateUserAPIKeyOutput
	require.NoError(t, decodeStructuredContent(second, &out))
	assert.True(t, out.Sensitive, "structured output must mark the new key sensitive")

	text := resultText(second)
	assert.Contains(t, text, "will not be shown again", "text content must warn the new key will not be shown again")
	assert.Contains(t, text, "Store", "text content must instruct the caller to store the new key now")
}

// TestRotateUserAPIKeyIgnoresAnyUserIDSuppliedAsToolInput asserts that even
// if a caller tries to smuggle a different user ID into the call
// arguments, it can never reach the handler at all: RotateUserAPIKeyInput
// has no field that could carry one, so the MCP SDK's automatic input
// schema validation rejects the extra "user_id" property before the
// handler ever runs (defense in depth on top of the handler itself always
// targeting the authenticated principal recovered from the verified MCP
// bearer token, never anything read from tool input -- see the successful
// rotation tests above, none of which ever supply a user ID as input).
func TestRotateUserAPIKeyIgnoresAnyUserIDSuppliedAsToolInput(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateResult: httpsms.User{ID: testUserID, APIKey: "new-key"}}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"user_id": "attacker-controlled-user-id"},
	})
	require.NoError(t, err, "tools/call must not be a protocol error")
	require.True(t, result.IsError)
	assert.Contains(t, resultText(result), "user_id")
	assert.Equal(t, 0, stub.totalCalls(), "an invalid call must never reach the httpSMS API")
}

func TestRotateUserAPIKeyMRTRDeclinedConfirmationDoesNotRotate(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	handle := first.RequestState

	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:           "rotate_user_api_key",
		InputResponses: mcp.InputResponseMap{"confirm_rotation": &mcp.ElicitResult{Action: "decline"}},
		RequestState:   handle,
	})
	require.NoError(t, err)
	require.True(t, second.IsError)
	assert.NotEmpty(t, resultText(second))
	assert.Equal(t, 0, stub.totalCalls(), "a declined confirmation must never reach the httpSMS API")
}

func TestRotateUserAPIKeyMRTRAcceptedButUnconfirmedContentDoesNotRotate(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)

	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:           "rotate_user_api_key",
		InputResponses: mcp.InputResponseMap{"confirm_rotation": &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirmed": false}}},
		RequestState:   first.RequestState,
	})
	require.NoError(t, err)
	require.True(t, second.IsError)
	assert.Equal(t, 0, stub.totalCalls())
}

// TestRotateUserAPIKeyMRTRMalformedResponseDoesNotRotate asserts a response
// of the wrong InputResponse concrete type (not *mcp.ElicitResult) under
// the confirm_rotation key is rejected instead of causing a panic or an
// accidental rotation.
func TestRotateUserAPIKeyMRTRMalformedResponseDoesNotRotate(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)

	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:           "rotate_user_api_key",
		InputResponses: mcp.InputResponseMap{"confirm_rotation": &mcp.ListRootsResult{}},
		RequestState:   first.RequestState,
	})
	require.NoError(t, err)
	require.True(t, second.IsError)
	assert.Equal(t, 0, stub.totalCalls())
}

// TestRotateUserAPIKeyMRTRReplayIsRejected asserts a handle that has
// already been redeemed by a completed rotation can never be redeemed
// again, even with a freshly re-accepted confirmation response.
func TestRotateUserAPIKeyMRTRReplayIsRejected(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateResult: httpsms.User{ID: testUserID, APIKey: "new-key"}}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	handle := first.RequestState

	retryParams := &mcp.CallToolParams{
		Name:           "rotate_user_api_key",
		InputResponses: mcp.InputResponseMap{"confirm_rotation": acceptedConfirmation()},
		RequestState:   handle,
	}

	second, err := session.CallTool(context.Background(), retryParams)
	require.NoError(t, err)
	require.False(t, second.IsError, resultText(second))
	require.Len(t, stub.rotateCalls, 1)

	// Replaying the exact same retry (same handle, same accepted response)
	// must fail: the handle was already consumed by the successful call
	// above and must never authorize a second rotation.
	third, err := session.CallTool(context.Background(), retryParams)
	require.NoError(t, err)
	require.True(t, third.IsError)
	assert.Len(t, stub.rotateCalls, 1, "a replayed confirmation must never call the API a second time")
}

// TestRotateUserAPIKeyAmbiguousConfirmationBothHandleAndMRTRStateIsRejected
// asserts that a call supplying both an explicit legacy
// confirmation_handle argument and MRTR confirmation state
// (RequestState/InputResponses) is rejected outright, rather than silently
// preferring one confirmation method over the other. Critically, the
// handle from the first call must remain unconsumed by this ambiguous
// attempt: a follow-up call that echoes it back cleanly (only as a legacy
// argument) must still succeed and rotate exactly once.
func TestRotateUserAPIKeyAmbiguousConfirmationBothHandleAndMRTRStateIsRejected(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateResult: httpsms.User{ID: testUserID, APIKey: "new-key"}}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	handle := first.RequestState
	require.NotEmpty(t, handle)

	ambiguous, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:           "rotate_user_api_key",
		Arguments:      map[string]any{"confirmation_handle": handle},
		InputResponses: mcp.InputResponseMap{"confirm_rotation": acceptedConfirmation()},
		RequestState:   handle,
	})
	require.NoError(t, err, "tools/call must not be a protocol error")
	require.True(t, ambiguous.IsError, "a call supplying both confirmation methods must be rejected")
	assert.Equal(t, 0, stub.totalCalls(), "an ambiguous confirmation attempt must never call the API")

	// The handle must still be unconsumed: a clean legacy retry with only
	// the argument set (no MRTR state) must still succeed.
	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"confirmation_handle": handle},
	})
	require.NoError(t, err)
	require.False(t, second.IsError, resultText(second))
	assert.Len(t, stub.rotateCalls, 1, "the unconsumed handle must still authorize exactly one rotation")
}

// TestRotateUserAPIKeyAmbiguousConfirmationHandleWithInputResponsesOnlyIsRejected
// covers the narrower ambiguous shape where MRTR state is signalled only
// via InputResponses (no RequestState echoed back), alongside an explicit
// legacy confirmation_handle argument.
func TestRotateUserAPIKeyAmbiguousConfirmationHandleWithInputResponsesOnlyIsRejected(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateResult: httpsms.User{ID: testUserID, APIKey: "new-key"}}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	handle := first.RequestState
	require.NotEmpty(t, handle)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:           "rotate_user_api_key",
		Arguments:      map[string]any{"confirmation_handle": handle},
		InputResponses: mcp.InputResponseMap{"confirm_rotation": acceptedConfirmation()},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Equal(t, 0, stub.totalCalls())
}

// --- rotate_user_api_key: legacy explicit confirmation_handle ---------------------------------------------------------

func TestRotateUserAPIKeyLegacyConfirmationHandleRotatesExactlyOnce(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateResult: httpsms.User{ID: testUserID, APIKey: "new-primary-api-key"}}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	handle := first.RequestState
	require.NotEmpty(t, handle)

	// The legacy retry is a brand-new, ordinary tool call: no
	// InputResponses, no RequestState, just the handle read off the first
	// call's raw JSON result and echoed back as a plain argument.
	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"confirmation_handle": handle},
	})
	require.NoError(t, err)
	require.False(t, second.IsError, resultText(second))

	var out tools.RotateUserAPIKeyOutput
	require.NoError(t, decodeStructuredContent(second, &out))
	assert.Equal(t, "new-primary-api-key", out.User.APIKey)

	require.Len(t, stub.rotateCalls, 1)
	assert.Equal(t, testUserID, stub.rotateCalls[0].Params)
}

func TestRotateUserAPIKeyLegacyHandleReplayIsRejected(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateResult: httpsms.User{ID: testUserID, APIKey: "new-key"}}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	handle := first.RequestState

	args := map[string]any{"confirmation_handle": handle}
	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key", Arguments: args})
	require.NoError(t, err)
	require.False(t, second.IsError, resultText(second))
	require.Len(t, stub.rotateCalls, 1)

	third, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key", Arguments: args})
	require.NoError(t, err)
	require.True(t, third.IsError)
	assert.Len(t, stub.rotateCalls, 1, "a replayed legacy handle must never call the API a second time")
}

func TestRotateUserAPIKeyLegacyHandleExpiredIsRejected(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	store, server := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, time.Minute)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)
	handle := first.RequestState

	server.FastForward(2 * time.Minute)

	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"confirmation_handle": handle},
	})
	require.NoError(t, err)
	require.True(t, second.IsError)
	assert.Equal(t, 0, stub.totalCalls())
}

func TestRotateUserAPIKeyLegacyHandleUnknownIsRejected(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"confirmation_handle": "this-handle-was-never-issued"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Equal(t, 0, stub.totalCalls())
}

// TestRotateUserAPIKeyLegacyHandleWrongUserIsRejected asserts a
// confirmation handle bound to a different user's Firebase UID (however it
// might have leaked or been guessed) can never authorize rotation for the
// current caller.
func TestRotateUserAPIKeyLegacyHandleWrongUserIsRejected(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	require.NoError(t, store.PutConfirmation(context.Background(), oauth.Confirmation{
		Handle:    "handle-for-a-different-user",
		UserID:    "someone-elses-firebase-uid",
		ClientID:  "test-client",
		Operation: "rotate_user_api_key",
	}, testConfirmationTTL))

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"confirmation_handle": "handle-for-a-different-user"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Equal(t, 0, stub.totalCalls())
}

// TestRotateUserAPIKeyLegacyHandleWrongClientIsRejected mirrors
// TestRotateUserAPIKeyLegacyHandleWrongUserIsRejected for the OAuth client
// binding: the same user, but a handle minted for a different OAuth
// client, must not authorize this session's rotation.
func TestRotateUserAPIKeyLegacyHandleWrongClientIsRejected(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes) // contextWithPrincipal always binds client_id "test-client"
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	require.NoError(t, store.PutConfirmation(context.Background(), oauth.Confirmation{
		Handle:    "handle-for-a-different-client",
		UserID:    testUserID,
		ClientID:  "some-other-oauth-client",
		Operation: "rotate_user_api_key",
	}, testConfirmationTTL))

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"confirmation_handle": "handle-for-a-different-client"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Equal(t, 0, stub.totalCalls())
}

// TestRotateUserAPIKeyLegacyHandleWrongOperationIsRejected asserts a handle
// minted for the correct user and client but a different operation (e.g. a
// future confirmable tool this service might add) cannot be replayed here.
func TestRotateUserAPIKeyLegacyHandleWrongOperationIsRejected(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	require.NoError(t, store.PutConfirmation(context.Background(), oauth.Confirmation{
		Handle:    "handle-for-a-different-operation",
		UserID:    testUserID,
		ClientID:  "test-client",
		Operation: "some_other_future_operation",
	}, testConfirmationTTL))

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"confirmation_handle": "handle-for-a-different-operation"},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Equal(t, 0, stub.totalCalls())
}

func TestRotateUserAPIKeySurfacesAPIErrorAsToolError(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateErr: &httpsms.APIError{StatusCode: http.StatusTooManyRequests, Message: "too many rotations"}}
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, stub, store, testConfirmationTTL)

	first, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "rotate_user_api_key"})
	require.NoError(t, err)

	second, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "rotate_user_api_key",
		Arguments: map[string]any{"confirmation_handle": first.RequestState},
	})
	require.NoError(t, err)
	require.True(t, second.IsError)
	assert.Contains(t, resultText(second), "too many rotations")
}

func TestRotateUserAPIKeyToolIsMarkedDestructiveAndNotIdempotent(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	store, _ := newTestConfirmationStore(t)
	session := newRotateSession(t, ctx, keys, &stubClient{}, store, testConfirmationTTL)

	tool := toolByName(t, session, "rotate_user_api_key")
	require.NotNil(t, tool.Annotations)
	assert.False(t, tool.Annotations.ReadOnlyHint)
	assert.False(t, tool.Annotations.IdempotentHint)
	require.NotNil(t, tool.Annotations.DestructiveHint)
	assert.True(t, *tool.Annotations.DestructiveHint)
}

// TestRotateUserAPIKeyMRTRFullRoundTripViaElicitationHandler proves the
// happy path works transparently, end-to-end, for a real MRTR-capable
// client: a single high-level CallTool call, with the SDK's own client-side
// middleware automatically fulfilling the confirm_rotation elicitation
// through an ElicitationHandler and retrying, exactly as documented in the
// go-sdk's own Example_mrtr.
func TestRotateUserAPIKeyMRTRFullRoundTripViaElicitationHandler(t *testing.T) {
	keys := newTestKeySet(t)
	ctx := contextWithPrincipal(t, keys, allScopes)
	stub := &stubClient{rotateResult: httpsms.User{ID: testUserID, APIKey: "new-primary-api-key"}}
	store, _ := newTestConfirmationStore(t)

	server := mcp.NewServer(&mcp.Implementation{Name: "httpsms-mcp-test", Version: "test"}, nil)
	tools.Register(server, keys, stub, testAPITokenTTL, store, testConfirmationTTL)

	t1, t2 := mcp.NewInMemoryTransports()
	_, err := server.Connect(ctx, t1, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, &mcp.ClientOptions{
		ElicitationHandler: func(_ context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			assert.Contains(t, req.Params.Message, "stop working")
			return acceptedConfirmation(), nil
		},
	})
	session, err := client.Connect(context.Background(), t2, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	var out tools.RotateUserAPIKeyOutput
	callTool(t, session, "rotate_user_api_key", nil, &out)

	assert.Equal(t, "new-primary-api-key", out.User.APIKey)
	assert.Len(t, stub.rotateCalls, 1)
}

// --- shared test helpers ---------------------------------------------------------

// decodeStructuredContent decodes result's StructuredContent into out, for
// asserting on a rotate_user_api_key result's output without relying on the
// callTool helper's built-in "must not be a tool error" assertion (some
// call sites here already asserted that separately, with a more useful
// failure message via resultText).
func decodeStructuredContent(result *mcp.CallToolResult, out any) error {
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// captureStdoutStderr redirects the process's stdout and stderr to a pipe
// for the duration of fn, and returns everything written to either.
func captureStdoutStderr(t *testing.T, fn func()) string {
	t.Helper()

	origStdout, origStderr := os.Stdout, os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout, os.Stderr = w, w

	captured := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		captured <- buf.String()
	}()

	fn()

	require.NoError(t, w.Close())
	os.Stdout, os.Stderr = origStdout, origStderr
	return <-captured
}
