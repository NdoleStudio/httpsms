// Package tools registers and implements the httpSMS MCP messaging tool
// catalog: list_phones, send_sms, list_message_threads,
// list_thread_messages, and list_incoming_messages.
//
// Every tool follows the same shape:
//
//  1. require the MCP access token's scope for this tool and recover the
//     calling Principal (auth.RequireScope);
//  2. mint a new short-lived API delegation token scoped to exactly the
//     one downstream httpSMS API operation this call is about to make
//     (auth.KeySet.SignAPIDelegationToken) -- the user ID bound into that
//     token is always the authenticated Principal's own Firebase UID, never
//     a value read from tool input;
//  3. call the typed httpsms.Client with that token;
//  4. return a stable, structured result.
//
// A tool never converts an upstream failure into a success-shaped empty
// result: httpsms.Client errors are already safe to expose to an MCP client
// (see the httpsms package's documented error-safety guarantees) and are
// returned as a tool-level error via toolError, never a protocol error.
package tools

import (
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
)

// Register adds every messaging tool to server, in the order approved by
// design: phones, send, threads, thread messages, incoming messages. keys
// mints the per-call API delegation token each tool needs; api is the
// typed httpSMS client each tool calls; apiTokenTTL bounds the lifetime of
// every minted delegation token.
func Register(server *mcp.Server, keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) {
	registerListPhones(server, keys, api, apiTokenTTL)
	registerSendSMS(server, keys, api, apiTokenTTL)
	registerListMessageThreads(server, keys, api, apiTokenTTL)
	registerListThreadMessages(server, keys, api, apiTokenTTL)
	registerListIncomingMessages(server, keys, api, apiTokenTTL)
}

// toolError converts err into a *mcp.CallToolResult carrying it as a
// tool-level error (CallToolResult.IsError set, err.Error() as the result's
// text content) rather than a JSON-RPC protocol error. err must already be
// safe to expose to an MCP client: every error httpsms.Client returns is
// documented to never carry a bearer token, request body, or SMS content,
// only a status code, the API's own message, field validation errors, and
// this client's own request ID.
func toolError(err error) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	result.SetError(err)
	return result
}

// readOnlyAnnotations marks a tool as read-only: it never modifies state
// and is safe to call repeatedly with the same arguments.
func readOnlyAnnotations() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
	}
}

// sendAnnotations marks a tool as performing a non-idempotent, potentially
// destructive side effect: sending a message is not safe to retry blindly,
// since repeating the call sends a second message.
func sendAnnotations() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(true),
		IdempotentHint:  false,
	}
}

// boolPtr returns a pointer to b, for building *bool-valued struct literals
// (mcp.ToolAnnotations.DestructiveHint and the *bool tool-input fields whose
// presence must be distinguishable from their zero value).
func boolPtr(b bool) *bool {
	return &b
}
