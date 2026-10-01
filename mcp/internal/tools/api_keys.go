package tools

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
	"github.com/NdoleStudio/httpsms/mcp/internal/oauth"
)

// Exact API route create_phone_api_key's delegation token is bound to. It is
// a wire contract with api/pkg/auth's delegated MCP route table and must not
// change independently of it.
const createPhoneAPIKeyPath = "/v1/phone-api-keys"

// rotateConfirmationRequestID is the InputRequests map key rotate_user_api_key
// uses for its confirmation elicitation. The client (or, for older protocol
// versions, the SDK's own server-side MRTR shim) must echo this same key back
// in InputResponses.
const rotateConfirmationRequestID = "confirm_rotation"

// rotateUserAPIKeyOperation is the Confirmation.Operation value stored for a
// rotate_user_api_key confirmation handle, binding a redeemed handle to this
// exact tool and never any other confirmable operation this service might add
// in the future.
const rotateUserAPIKeyOperation = "rotate_user_api_key"

// CreatePhoneAPIKeyInput is the input for the create_phone_api_key tool.
type CreatePhoneAPIKeyInput struct {
	// Name is a human-readable label for the new phone API key.
	Name string `json:"name" jsonschema:"human-readable label for the new phone API key"`
}

// CreatePhoneAPIKeyOutput is the output for the create_phone_api_key tool.
// APIKey is a secret, one-time display value: it is returned only in this
// structured result and is never logged, traced, or persisted by this
// service.
type CreatePhoneAPIKeyOutput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	APIKey    string `json:"api_key"`
	Sensitive bool   `json:"sensitive"`
}

// registerCreatePhoneAPIKey registers the create_phone_api_key tool. It
// calls POST /v1/phone-api-keys and requires the phone-api-keys:write scope.
func registerCreatePhoneAPIKey(server *mcp.Server, keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "create_phone_api_key",
		Description: "Create a new httpSMS phone API key, used to authenticate " +
			"the httpSMS Android app for a subset of the user's phones. This is " +
			"a sensitive, non-idempotent operation: every call mints a brand-new " +
			"secret key, which is returned exactly once and can never be " +
			"retrieved again -- store it immediately.",
		Annotations: createAPIKeyAnnotations(),
	}, newCreatePhoneAPIKeyHandler(keys, api, apiTokenTTL))
}

func newCreatePhoneAPIKeyHandler(keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) mcp.ToolHandlerFor[CreatePhoneAPIKeyInput, CreatePhoneAPIKeyOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in CreatePhoneAPIKeyInput) (*mcp.CallToolResult, CreatePhoneAPIKeyOutput, error) {
		principal, err := auth.RequireScope(ctx, auth.ScopePhoneAPIKeysWrite)
		if err != nil {
			return nil, CreatePhoneAPIKeyOutput{}, err
		}

		token, err := keys.SignAPIDelegationToken(principal, []string{auth.ScopePhoneAPIKeysWrite}, http.MethodPost, createPhoneAPIKeyPath, apiTokenTTL)
		if err != nil {
			return nil, CreatePhoneAPIKeyOutput{}, fmt.Errorf("sign API delegation token: %w", err)
		}

		key, err := api.CreatePhoneAPIKey(ctx, token, httpsms.CreatePhoneAPIKeyParams{Name: in.Name})
		if err != nil {
			return toolError(err), CreatePhoneAPIKeyOutput{}, nil
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: "Store this API key now: it will not be shown again. Configure " +
					"it in the httpSMS Android app for the intended phone(s).",
			}},
		}
		return result, CreatePhoneAPIKeyOutput{
			ID:        key.ID,
			Name:      key.Name,
			APIKey:    key.APIKey,
			Sensitive: true,
		}, nil
	}
}

// RotateUserAPIKeyInput is the input for the rotate_user_api_key tool.
type RotateUserAPIKeyInput struct {
	// ConfirmationHandle is the one-time confirmation handle returned by a
	// prior, unconfirmed call to this tool. Only legacy clients that cannot
	// complete an MCP multi-round-trip (MRTR) elicitation need to supply
	// this explicitly; MRTR-capable clients instead fulfill the tool's
	// "confirm_rotation" elicitation and never need to set this field.
	ConfirmationHandle string `json:"confirmation_handle,omitempty" jsonschema:"one-time confirmation handle returned by a prior unconfirmed call to this tool, for legacy clients that cannot complete an MRTR elicitation"`
}

// RotateUserAPIKeyOutput is the output for the rotate_user_api_key tool. It
// is populated only once rotation has actually happened, after confirmation.
// User.APIKey is a secret, one-time display value: it is returned only in
// this structured result and is never logged, traced, or persisted by this
// service.
type RotateUserAPIKeyOutput struct {
	// User is the authenticated user's record after rotation, carrying the
	// brand-new primary API key.
	User httpsms.User `json:"user"`
	// Sensitive marks User.APIKey as a secret, one-time display value: it
	// is shown here exactly once and can never be retrieved again,
	// matching create_phone_api_key's CreatePhoneAPIKeyOutput.Sensitive.
	Sensitive bool `json:"sensitive"`
	// Warning restates that the previous primary API key has just stopped
	// working and every device or integration using it must be updated.
	Warning string `json:"warning"`
}

// registerRotateUserAPIKey registers the rotate_user_api_key tool. It calls
// DELETE /v1/users/{userID}/api-keys (userID is always the authenticated
// principal's own Firebase UID, never tool input) and requires the
// user-api-key:rotate scope. Rotation only proceeds after the caller
// confirms it, through either an MCP multi-round-trip (MRTR) elicitation or
// (for legacy clients that cannot complete one) an explicit
// confirmation_handle from a prior call; see store and confirmationTTL.
func registerRotateUserAPIKey(server *mcp.Server, keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration, store oauth.Store, confirmationTTL time.Duration) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "rotate_user_api_key",
		Description: "Rotate the user's primary httpSMS API key, invalidating " +
			"the current one and minting a brand-new secret in its place. This " +
			"is a sensitive, destructive, non-idempotent operation that requires " +
			"the caller to explicitly confirm before it takes effect.",
		Annotations: rotateAPIKeyAnnotations(),
	}, newRotateUserAPIKeyHandler(keys, api, apiTokenTTL, store, confirmationTTL))
}

func newRotateUserAPIKeyHandler(keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration, store oauth.Store, confirmationTTL time.Duration) mcp.ToolHandlerFor[RotateUserAPIKeyInput, *RotateUserAPIKeyOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in RotateUserAPIKeyInput) (*mcp.CallToolResult, *RotateUserAPIKeyOutput, error) {
		principal, err := auth.RequireScope(ctx, auth.ScopeUserAPIKeyRotate)
		if err != nil {
			return nil, nil, err
		}

		// clientID is always present once RequireScope has succeeded: every
		// MCP access token this service mints carries a client_id claim
		// (possibly empty for a hypothetical clientless token), and
		// Verifier.VerifyMCPToken always stores it.
		clientID, _ := auth.ClientIDFromContext(ctx)

		granted, err := resolveRotationConfirmation(ctx, store, req, in, principal, clientID)
		if err != nil {
			return nil, nil, err
		}

		if !granted {
			result, err := beginRotationConfirmation(ctx, store, principal, clientID, confirmationTTL)
			if err != nil {
				return nil, nil, err
			}
			return result, nil, nil
		}

		token, err := keys.SignAPIDelegationToken(principal, []string{auth.ScopeUserAPIKeyRotate}, http.MethodDelete, rotateUserAPIKeyPath(principal.UserID), apiTokenTTL)
		if err != nil {
			return nil, nil, fmt.Errorf("sign API delegation token: %w", err)
		}

		user, err := api.RotateUserAPIKey(ctx, token, principal.UserID)
		if err != nil {
			return toolError(err), nil, nil
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: "Store this new API key now: it will not be shown again. " +
					"The previous primary API key has been invalidated; update " +
					"every device or integration (including the httpSMS Android " +
					"app, if configured with the primary key) that used it with " +
					"this new key.",
			}},
		}
		return result, &RotateUserAPIKeyOutput{
			User:      user,
			Sensitive: true,
			Warning: "The previous primary API key has been invalidated. Update " +
				"every device or integration (including the httpSMS Android app, " +
				"if configured with the primary key) that used it with this new key.",
		}, nil
	}
}

// beginRotationConfirmation generates and stores a fresh one-time
// confirmation handle bound to principal, clientID, and
// rotateUserAPIKeyOperation, then returns the CallToolResult that asks the
// caller to confirm before rotation proceeds: an MRTR elicitation carrying
// the handle as RequestState. A legacy client that cannot complete that
// elicitation can instead read RequestState directly off this same JSON
// result and echo it back as RotateUserAPIKeyInput.ConfirmationHandle on a
// brand-new call.
func beginRotationConfirmation(ctx context.Context, store oauth.Store, principal auth.Principal, clientID string, confirmationTTL time.Duration) (*mcp.CallToolResult, error) {
	handle, err := oauth.NewConfirmationHandle()
	if err != nil {
		return nil, fmt.Errorf("generate rotation confirmation handle: %w", err)
	}

	if err := store.PutConfirmation(ctx, oauth.Confirmation{
		Handle:    handle,
		UserID:    principal.UserID,
		ClientID:  clientID,
		Operation: rotateUserAPIKeyOperation,
		CreatedAt: time.Now().UTC(),
	}, confirmationTTL); err != nil {
		return nil, fmt.Errorf("store rotation confirmation: %w", err)
	}

	return &mcp.CallToolResult{
		InputRequests: mcp.InputRequestMap{
			rotateConfirmationRequestID: rotateConfirmationElicitParams(),
		},
		RequestState: handle,
	}, nil
}

// rotateConfirmationElicitParams is the MRTR elicitation rotate_user_api_key
// asks the caller to fulfill before rotation proceeds. Its Message carries
// the required warning that the current primary API key will stop working.
func rotateConfirmationElicitParams() *mcp.ElicitParams {
	return &mcp.ElicitParams{
		Message: "Rotating your primary httpSMS API key immediately invalidates " +
			"the current key. Every device or integration using it (including " +
			"the httpSMS Android app, if configured with the primary key) will " +
			"stop working until reconfigured with the new key. Confirm to proceed.",
		RequestedSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"confirmed": {
					Type:        "boolean",
					Description: "Set to true to confirm rotating the primary API key.",
				},
			},
			Required: []string{"confirmed"},
		},
	}
}

// resolveRotationConfirmation determines whether the caller has already
// confirmed rotation.
//
// It returns (true, nil) once a previously issued confirmation handle has
// been redeemed and validated: it existed, had not expired or already been
// redeemed, was bound to this exact principal/clientID/operation, and (for
// the MRTR path) carries an accepted "confirmed" elicitation response.
//
// It returns (false, nil) when this call made no confirmation attempt at
// all (RotateUserAPIKeyInput.ConfirmationHandle is empty and req carries no
// RequestState): the caller has not been asked yet.
//
// It returns (false, err) when a confirmation attempt was made but could
// not be validated (unknown/expired/already-redeemed handle, a handle
// bound to a different user/client/operation, a declined/malformed
// elicitation response, or -- checked first, before any handle is touched
// -- an ambiguous call that supplies both an explicit legacy
// ConfirmationHandle and MRTR confirmation state). Every handle that is
// actually looked up is consumed exactly once by ConsumeConfirmation before
// this function inspects it, so a caller can never replay it, whether or
// not the attempt is ultimately accepted.
func resolveRotationConfirmation(ctx context.Context, store oauth.Store, req *mcp.CallToolRequest, in RotateUserAPIKeyInput, principal auth.Principal, clientID string) (bool, error) {
	hasExplicitHandle := in.ConfirmationHandle != ""
	hasMRTRState := req.Params.RequestState != "" || len(req.Params.InputResponses) > 0
	if hasExplicitHandle && hasMRTRState {
		// Ambiguous: never silently prefer one confirmation method over
		// the other. Reject before consuming anything or calling the API.
		return false, errors.New("rotate_user_api_key received both a confirmation_handle argument and MRTR confirmation state (RequestState/InputResponses); use exactly one confirmation method, not both")
	}

	handle := in.ConfirmationHandle
	viaMRTR := false
	if handle == "" {
		if req.Params.RequestState == "" {
			// No confirmation handle at all: this is the first call.
			return false, nil
		}
		handle = req.Params.RequestState
		viaMRTR = true
	}

	confirmation, err := store.ConsumeConfirmation(ctx, handle)
	if err != nil {
		if errors.Is(err, oauth.ErrNotFound) {
			return false, errors.New("this rotation confirmation has expired, was already used, or is invalid; call rotate_user_api_key again to request a new confirmation")
		}
		return false, fmt.Errorf("consume rotation confirmation: %w", err)
	}

	if !confirmationBindingMatches(confirmation, principal, clientID) {
		return false, errors.New("this rotation confirmation is not valid for the current user, client, or operation")
	}

	if viaMRTR {
		if err := validateRotationElicitationResponse(req); err != nil {
			return false, err
		}
	}

	return true, nil
}

// confirmationBindingMatches reports whether confirmation was issued for
// exactly principal, clientID, and rotateUserAPIKeyOperation. Every
// comparison is constant-time: confirmation.UserID, ClientID, and Operation
// are all values this service itself generated and stored, but comparing
// them in variable time would still let a timing side channel distinguish a
// near-miss from a random guess.
func confirmationBindingMatches(confirmation oauth.Confirmation, principal auth.Principal, clientID string) bool {
	return subtle.ConstantTimeCompare([]byte(confirmation.UserID), []byte(principal.UserID)) == 1 &&
		subtle.ConstantTimeCompare([]byte(confirmation.ClientID), []byte(clientID)) == 1 &&
		subtle.ConstantTimeCompare([]byte(confirmation.Operation), []byte(rotateUserAPIKeyOperation)) == 1
}

// validateRotationElicitationResponse requires req to carry an accepted
// "confirmed": true response to the confirm_rotation elicitation. Any other
// shape -- a missing response, a response of the wrong type, a declined or
// cancelled action, or an accepted response missing "confirmed": true -- is
// rejected without ever calling the httpSMS API.
func validateRotationElicitationResponse(req *mcp.CallToolRequest) error {
	response, ok := req.Params.InputResponses[rotateConfirmationRequestID].(*mcp.ElicitResult)
	if !ok {
		return errors.New("expected a confirm_rotation elicitation response")
	}
	if response.Action != "accept" {
		return errors.New("rotation was not confirmed")
	}
	confirmed, _ := response.Content["confirmed"].(bool)
	if !confirmed {
		return errors.New("rotation was not confirmed")
	}
	return nil
}

// rotateUserAPIKeyPath returns the exact DELETE /v1/users/{userID}/api-keys
// path for userID, byte-for-byte identical to the path
// httpsms.HTTPClient.RotateUserAPIKey builds and actually requests. The API
// delegation token minted for this call must be bound to this same literal
// path (not a wildcard pattern), because api/pkg/auth's delegated MCP
// verifier requires an exact match between a token's Path claim and the
// real request path.
func rotateUserAPIKeyPath(userID string) string {
	return "/v1/users/" + url.PathEscape(userID) + "/api-keys"
}
