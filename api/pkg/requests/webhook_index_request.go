package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
)

// WebhookIndex is the payload for fetching entities.Webhook of a user
type WebhookIndex struct {
	request
	Skip  string `json:"skip" query:"skip"`
	Query string `json:"query" query:"query"`
	Limit string `json:"limit" query:"limit"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *WebhookIndex) Sanitize() WebhookIndex {
	if strings.TrimSpace(input.Limit) == "" {
		input.Limit = "1"
	}
	input.Query = strings.TrimSpace(input.Query)
	input.Skip = strings.TrimSpace(input.Skip)
	if input.Skip == "" {
		input.Skip = "0"
	}
	return *input
}

// ToIndexParams converts HeartbeatIndex to repositories.IndexParams
func (input *WebhookIndex) ToIndexParams() repositories.IndexParams {
	return repositories.IndexParams{
		Skip:  input.getInt(input.Skip),
		Query: input.Query,
		Limit: input.getInt(input.Limit),
	}
}
