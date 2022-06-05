package requests

import (
	"strings"

	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
)

// MessageThreadIndex is the payload fetching entities.MessageThread sent between 2 numbers
type MessageThreadIndex struct {
	request
	Skip  string `json:"skip" query:"skip"`
	Query string `json:"query" query:"query"`
	Limit string `json:"limit" query:"limit"`
	Owner string `json:"owner" query:"owner"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *MessageThreadIndex) Sanitize() MessageThreadIndex {
	if strings.TrimSpace(input.Limit) == "" {
		input.Limit = "20"
	}

	input.Query = strings.TrimSpace(input.Query)

	input.Owner = input.sanitizeAddress(input.Owner)

	input.Skip = strings.TrimSpace(input.Skip)
	if input.Skip == "" {
		input.Skip = "0"
	}

	return *input
}

// ToGetParams converts request to services.MessageThreadGetParams
func (input *MessageThreadIndex) ToGetParams() services.MessageThreadGetParams {
	return services.MessageThreadGetParams{
		IndexParams: repositories.IndexParams{
			Skip:  input.getInt(input.Skip),
			Query: input.Query,
			Limit: input.getInt(input.Limit),
		},
		Owner: input.Owner,
	}
}
