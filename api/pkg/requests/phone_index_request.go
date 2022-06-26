package requests

import (
	"strings"

	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
)

// PhoneIndex is the payload fetching registered phones
type PhoneIndex struct {
	request
	Skip  string `json:"skip" query:"skip"`
	Owner string `json:"owner" query:"owner"`
	Query string `json:"query" query:"query"`
	Limit string `json:"limit" query:"limit"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *PhoneIndex) Sanitize() PhoneIndex {
	if strings.TrimSpace(input.Limit) == "" {
		input.Limit = "1"
	}
	input.Query = strings.TrimSpace(input.Query)
	input.Owner = input.sanitizeAddress(input.Owner)
	input.Skip = strings.TrimSpace(input.Skip)
	if input.Skip == "" {
		input.Skip = "0"
	}
	return *input
}

// ToIndexParams converts HeartbeatIndex to repositories.IndexParams
func (input *PhoneIndex) ToIndexParams() repositories.IndexParams {
	return repositories.IndexParams{
		Skip:  input.getInt(input.Skip),
		Query: input.Query,
		Limit: input.getInt(input.Limit),
	}
}
