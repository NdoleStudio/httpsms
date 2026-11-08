package tools

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
)

// listPhonesPath is the exact API route the list_phones tool's delegation
// token is bound to. It is a wire contract with api/pkg/auth's delegated
// MCP route table and must not change independently of it.
const listPhonesPath = "/v1/phones"

// ListPhonesInput is the input for the list_phones tool.
type ListPhonesInput struct {
	// Query filters phones whose phone number contains this substring.
	Query string `json:"query,omitempty" jsonschema:"filter phones whose phone number contains this substring"`
	// Skip is the number of matching phones to skip, for pagination.
	Skip int `json:"skip,omitempty" jsonschema:"number of matching phones to skip, for pagination"`
	// Limit bounds how many phones are returned.
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of phones to return"`
}

// ListPhonesOutput is the output for the list_phones tool.
type ListPhonesOutput struct {
	// Phones are the user's registered httpSMS sending phones matching the
	// request.
	Phones []httpsms.Phone `json:"phones"`
	// Count is len(Phones).
	Count int `json:"count"`
}

// registerListPhones registers the list_phones tool. It calls
// GET /v1/phones and requires the phones:read scope.
func registerListPhones(server *mcp.Server, keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_phones",
		Description: "List the user's registered httpSMS sending phones, " +
			"including each phone's number, SIM slot, and per-minute sending " +
			"rate. Use this to find a valid \"from\" number before sending an " +
			"SMS or listing message threads.",
		Annotations: readOnlyAnnotations(),
	}, newListPhonesHandler(keys, api, apiTokenTTL))
}

func newListPhonesHandler(keys *auth.KeySet, api httpsms.Client, apiTokenTTL time.Duration) mcp.ToolHandlerFor[ListPhonesInput, ListPhonesOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListPhonesInput) (*mcp.CallToolResult, ListPhonesOutput, error) {
		principal, err := auth.RequireScope(ctx, auth.ScopePhonesRead)
		if err != nil {
			return nil, ListPhonesOutput{}, err
		}

		token, err := keys.SignAPIDelegationToken(principal, []string{auth.ScopePhonesRead}, http.MethodGet, listPhonesPath, apiTokenTTL)
		if err != nil {
			return nil, ListPhonesOutput{}, fmt.Errorf("sign API delegation token: %w", err)
		}

		phones, err := api.ListPhones(ctx, token, httpsms.ListPhonesParams{
			Query: in.Query,
			Skip:  in.Skip,
			Limit: in.Limit,
		})
		if err != nil {
			return toolError(err), ListPhonesOutput{}, nil
		}

		return nil, ListPhonesOutput{Phones: phones, Count: len(phones)}, nil
	}
}
