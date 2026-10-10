package tools

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
)

// Exact API routes the tools in this file mint delegation tokens for. Each
// is a wire contract with api/pkg/auth's delegated MCP route table and
// must not change independently of it.
const (
	sendSMSPath              = "/v1/messages/send"
	listMessageThreadsPath   = "/v1/message-threads"
	listThreadMessagesPath   = "/v1/messages"
	listIncomingMessagesPath = "/v1/messages/incoming"
)

// listMessageThreadsMaxLimit bounds how many message threads a single
// list_message_threads call may request. It is enforced in the tool's
// input schema, so an out-of-range request is rejected by the MCP SDK's
// automatic input validation before the handler -- and therefore before
// any downstream API call -- ever runs.
const listMessageThreadsMaxLimit = 20

// SendSMSInput is the input for the send_sms tool.
//
// SendSMSInput has no "sim" field: the httpSMS API selects the sending SIM
// implicitly from From (every registered phone number is already bound to
// exactly one SIM slot), so a separate SIM selector would be accepted but
// never forwarded to the API by api/pkg/requests.MessageSend -- a no-op
// field this tool deliberately does not expose.
type SendSMSInput struct {
	// From is the registered httpSMS phone number to send from, in E.164
	// format.
	From string `json:"from" jsonschema:"registered httpSMS phone number to send from, in E.164 format"`
	// To is the destination phone number, in E.164 format.
	To string `json:"to" jsonschema:"destination phone number, in E.164 format"`
	// Content is the SMS content.
	Content string `json:"content" jsonschema:"SMS content"`
	// Attachments are optional MMS attachment URLs.
	Attachments []string `json:"attachments,omitempty" jsonschema:"URLs of MMS attachments; sending any attachment sends the message as an MMS"`
	// Encrypted marks Content as end-to-end encrypted by the sending
	// device.
	Encrypted bool `json:"encrypted,omitempty" jsonschema:"whether Content is end-to-end encrypted by the sending device"`
	// RequestID is a caller-supplied idempotency key for this send.
	RequestID string `json:"request_id,omitempty" jsonschema:"caller-supplied idempotency key used to track this request"`
	// SendAt schedules the message instead of sending immediately.
	SendAt *time.Time `json:"send_at,omitempty" jsonschema:"schedule the message to be sent at this future time instead of immediately"`
}

// SendSMSOutput is the output for the send_sms tool.
type SendSMSOutput struct {
	// Message is the message record created by the send request.
	Message httpsms.Message `json:"message"`
}

// registerSendSMS registers the send_sms tool. It calls
// POST /v1/messages/send and requires the messages:send scope.
func registerSendSMS(server *mcp.Server, keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "send_sms",
		Description: "Send an SMS or MMS message from one of the user's " +
			"registered httpSMS phones. Sending any attachment sends the " +
			"message as an MMS. Provide request_id to make retries safe.",
		Annotations: sendAnnotations(),
	}, newSendSMSHandler(keys, api, apiTokenTTL))
}

func newSendSMSHandler(keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) mcp.ToolHandlerFor[SendSMSInput, SendSMSOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in SendSMSInput) (*mcp.CallToolResult, SendSMSOutput, error) {
		principal, err := auth.RequireScope(ctx, auth.ScopeMessagesSend)
		if err != nil {
			return nil, SendSMSOutput{}, err
		}

		token, err := keys.SignAPIDelegationToken(principal, []string{auth.ScopeMessagesSend}, http.MethodPost, sendSMSPath, apiTokenTTL)
		if err != nil {
			return nil, SendSMSOutput{}, fmt.Errorf("sign API delegation token: %w", err)
		}

		message, err := api.SendSMS(ctx, token, httpsms.SendSMSParams{
			From:        in.From,
			To:          in.To,
			Content:     in.Content,
			Attachments: in.Attachments,
			Encrypted:   in.Encrypted,
			RequestID:   in.RequestID,
			SendAt:      in.SendAt,
		})
		if err != nil {
			return toolError(err), SendSMSOutput{}, nil
		}

		return nil, SendSMSOutput{Message: message}, nil
	}
}

// ListMessageThreadsInput is the input for the list_message_threads tool.
type ListMessageThreadsInput struct {
	// Owner is the registered httpSMS phone number owning the threads, in
	// E.164 format.
	Owner string `json:"owner" jsonschema:"registered httpSMS phone number owning the threads, in E.164 format"`
	// IsArchived filters to archived (true) or unarchived (false) threads
	// only. Omit to get unarchived threads.
	IsArchived *bool `json:"is_archived,omitempty" jsonschema:"filter to archived (true) or unarchived (false) threads only; omit for unarchived threads"`
	// WithContacts includes each contact's saved name, if any.
	WithContacts bool `json:"with_contacts,omitempty" jsonschema:"include each contact's saved name, if any"`
	// Query filters threads by contact name or number substring.
	Query string `json:"query,omitempty" jsonschema:"filter threads by contact name or phone number substring"`
	// Skip is the number of matching threads to skip, for pagination.
	Skip int `json:"skip,omitempty" jsonschema:"number of matching threads to skip, for pagination"`
	// Limit bounds how many threads are returned, up to
	// listMessageThreadsMaxLimit.
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of threads to return"`
}

// ListMessageThreadsOutput is the output for the list_message_threads tool.
type ListMessageThreadsOutput struct {
	// Threads are the matching conversations between Owner and its
	// contacts.
	Threads []httpsms.MessageThread `json:"threads"`
	// Count is len(Threads).
	Count int `json:"count"`
}

// registerListMessageThreads registers the list_message_threads tool. It
// calls GET /v1/message-threads and requires the messages:read scope.
func registerListMessageThreads(server *mcp.Server, keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_message_threads",
		Description: "List the user's message-thread conversations between " +
			"a registered phone (owner) and its contacts.",
		InputSchema: listMessageThreadsInputSchema(),
		Annotations: readOnlyAnnotations(),
	}, newListMessageThreadsHandler(keys, api, apiTokenTTL))
}

// listMessageThreadsInputSchema infers ListMessageThreadsInput's default
// schema and then clamps "limit" to [1, listMessageThreadsMaxLimit] and
// "skip" to a non-negative minimum, so the MCP SDK's automatic input
// validation rejects an out-of-range request before the handler runs.
func listMessageThreadsInputSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[ListMessageThreadsInput](nil)
	if err != nil {
		panic(fmt.Sprintf("tools: cannot infer list_message_threads input schema: %v", err))
	}

	schema.Properties["limit"].Minimum = jsonschema.Ptr(1.0)
	schema.Properties["limit"].Maximum = jsonschema.Ptr(float64(listMessageThreadsMaxLimit))
	schema.Properties["skip"].Minimum = jsonschema.Ptr(0.0)

	return schema
}

func newListMessageThreadsHandler(keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) mcp.ToolHandlerFor[ListMessageThreadsInput, ListMessageThreadsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListMessageThreadsInput) (*mcp.CallToolResult, ListMessageThreadsOutput, error) {
		principal, err := auth.RequireScope(ctx, auth.ScopeMessagesRead)
		if err != nil {
			return nil, ListMessageThreadsOutput{}, err
		}

		token, err := keys.SignAPIDelegationToken(principal, []string{auth.ScopeMessagesRead}, http.MethodGet, listMessageThreadsPath, apiTokenTTL)
		if err != nil {
			return nil, ListMessageThreadsOutput{}, fmt.Errorf("sign API delegation token: %w", err)
		}

		threads, err := api.ListMessageThreads(ctx, token, httpsms.ListMessageThreadsParams{
			Owner:        in.Owner,
			IsArchived:   in.IsArchived,
			WithContacts: in.WithContacts,
			Query:        in.Query,
			Skip:         in.Skip,
			Limit:        in.Limit,
		})
		if err != nil {
			return toolError(err), ListMessageThreadsOutput{}, nil
		}

		return nil, ListMessageThreadsOutput{Threads: threads, Count: len(threads)}, nil
	}
}

// ListThreadMessagesInput is the input for the list_thread_messages tool.
// Owner and Contact are both required: together they identify the single
// thread being read.
type ListThreadMessagesInput struct {
	// Owner is the registered httpSMS phone number that owns the thread, in
	// E.164 format.
	Owner string `json:"owner" jsonschema:"registered httpSMS phone number that owns the thread, in E.164 format"`
	// Contact is the other party in the thread, in E.164 format.
	Contact string `json:"contact" jsonschema:"the other party's phone number in the thread, in E.164 format"`
	// Query filters messages by content substring.
	Query string `json:"query,omitempty" jsonschema:"filter messages whose content contains this substring"`
	// Skip is the number of matching messages to skip, for pagination.
	Skip int `json:"skip,omitempty" jsonschema:"number of matching messages to skip, for pagination"`
	// Limit bounds how many messages are returned.
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of messages to return"`
}

// ListThreadMessagesOutput is the output for the list_thread_messages tool.
type ListThreadMessagesOutput struct {
	// Messages are the matching messages exchanged between Owner and
	// Contact.
	Messages []httpsms.Message `json:"messages"`
	// Count is len(Messages).
	Count int `json:"count"`
}

// registerListThreadMessages registers the list_thread_messages tool. It
// calls GET /v1/messages and requires the messages:read scope.
func registerListThreadMessages(server *mcp.Server, keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_thread_messages",
		Description: "List the messages exchanged between a registered " +
			"phone (owner) and a specific contact.",
		Annotations: readOnlyAnnotations(),
	}, newListThreadMessagesHandler(keys, api, apiTokenTTL))
}

func newListThreadMessagesHandler(keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) mcp.ToolHandlerFor[ListThreadMessagesInput, ListThreadMessagesOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListThreadMessagesInput) (*mcp.CallToolResult, ListThreadMessagesOutput, error) {
		principal, err := auth.RequireScope(ctx, auth.ScopeMessagesRead)
		if err != nil {
			return nil, ListThreadMessagesOutput{}, err
		}

		token, err := keys.SignAPIDelegationToken(principal, []string{auth.ScopeMessagesRead}, http.MethodGet, listThreadMessagesPath, apiTokenTTL)
		if err != nil {
			return nil, ListThreadMessagesOutput{}, fmt.Errorf("sign API delegation token: %w", err)
		}

		messages, err := api.ListThreadMessages(ctx, token, httpsms.ListThreadMessagesParams{
			Owner:   in.Owner,
			Contact: in.Contact,
			Query:   in.Query,
			Skip:    in.Skip,
			Limit:   in.Limit,
		})
		if err != nil {
			return toolError(err), ListThreadMessagesOutput{}, nil
		}

		return nil, ListThreadMessagesOutput{Messages: messages, Count: len(messages)}, nil
	}
}

// ListIncomingMessagesInput is the input for the list_incoming_messages
// tool.
type ListIncomingMessagesInput struct {
	// Owners optionally restricts results to these registered phone
	// numbers. Omit to search across every registered phone.
	Owners []string `json:"owners,omitempty" jsonschema:"restrict results to these registered phone numbers; omit to search every registered phone"`
	// Statuses optionally restricts results to these message statuses.
	Statuses []string `json:"statuses,omitempty" jsonschema:"restrict results to these message statuses"`
	// Query filters messages by content or contact substring.
	Query string `json:"query,omitempty" jsonschema:"filter messages by content or contact phone number substring"`
	// SortBy optionally names the field results are ordered by.
	SortBy string `json:"sort_by,omitempty" jsonschema:"field to sort results by"`
	// SortDescending optionally reverses the sort order.
	SortDescending *bool `json:"sort_descending,omitempty" jsonschema:"sort in descending order; omit to use the API's default order"`
	// Skip is the number of matching messages to skip, for pagination.
	Skip int `json:"skip,omitempty" jsonschema:"number of matching messages to skip, for pagination"`
	// Limit bounds how many messages are returned.
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of messages to return"`
}

// ListIncomingMessagesOutput is the output for the list_incoming_messages
// tool.
type ListIncomingMessagesOutput struct {
	// Messages are the matching mobile-originated (incoming) messages.
	Messages []httpsms.Message `json:"messages"`
	// Count is len(Messages).
	Count int `json:"count"`
}

// registerListIncomingMessages registers the list_incoming_messages tool.
// It calls GET /v1/messages/incoming (never the CAPTCHA-protected
// /v1/messages/search route) and requires the messages:read scope.
func registerListIncomingMessages(server *mcp.Server, keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_incoming_messages",
		Description: "List the user's incoming (mobile-originated) SMS " +
			"messages received on any registered phone, optionally filtered " +
			"by owner, status, or content.",
		Annotations: readOnlyAnnotations(),
	}, newListIncomingMessagesHandler(keys, api, apiTokenTTL))
}

func newListIncomingMessagesHandler(keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) mcp.ToolHandlerFor[ListIncomingMessagesInput, ListIncomingMessagesOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListIncomingMessagesInput) (*mcp.CallToolResult, ListIncomingMessagesOutput, error) {
		principal, err := auth.RequireScope(ctx, auth.ScopeMessagesRead)
		if err != nil {
			return nil, ListIncomingMessagesOutput{}, err
		}

		token, err := keys.SignAPIDelegationToken(principal, []string{auth.ScopeMessagesRead}, http.MethodGet, listIncomingMessagesPath, apiTokenTTL)
		if err != nil {
			return nil, ListIncomingMessagesOutput{}, fmt.Errorf("sign API delegation token: %w", err)
		}

		messages, err := api.ListIncomingMessages(ctx, token, httpsms.ListIncomingMessagesParams{
			Owners:         in.Owners,
			Statuses:       in.Statuses,
			Query:          in.Query,
			SortBy:         in.SortBy,
			SortDescending: in.SortDescending,
			Skip:           in.Skip,
			Limit:          in.Limit,
		})
		if err != nil {
			return toolError(err), ListIncomingMessagesOutput{}, nil
		}

		return nil, ListIncomingMessagesOutput{Messages: messages, Count: len(messages)}, nil
	}
}
