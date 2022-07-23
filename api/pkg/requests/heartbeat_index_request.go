package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
)

// HeartbeatIndex is the payload for fetching entities.Heartbeat of a phone number
type HeartbeatIndex struct {
	request
	Skip  string `json:"skip" query:"skip"`
	Owner string `json:"owner" query:"owner"`
	Query string `json:"query" query:"query"`
	Limit string `json:"limit" query:"limit"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *HeartbeatIndex) Sanitize() HeartbeatIndex {
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
func (input *HeartbeatIndex) ToIndexParams() repositories.IndexParams {
	return repositories.IndexParams{
		Skip:  input.getInt(input.Skip),
		Query: input.Query,
		Limit: input.getInt(input.Limit),
	}
}
