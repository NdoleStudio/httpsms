package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
)

// ContactIndex lists contacts for a user.
type ContactIndex struct {
	request
	Skip  string `json:"skip" query:"skip"`
	Query string `json:"query" query:"query"`
	Limit string `json:"limit" query:"limit"`
}

// Sanitize sets defaults for the list request.
func (input *ContactIndex) Sanitize() ContactIndex {
	input.Query = strings.TrimSpace(input.Query)
	input.Skip = strings.TrimSpace(input.Skip)
	input.Limit = strings.TrimSpace(input.Limit)

	if input.Skip == "" {
		input.Skip = "0"
	}
	if input.Limit == "" {
		input.Limit = "20"
	}
	return *input
}

// ToIndexParams converts the request into repositories.IndexParams.
func (input *ContactIndex) ToIndexParams() repositories.IndexParams {
	return repositories.IndexParams{
		Skip:  input.getInt(input.Skip),
		Query: input.Query,
		Limit: input.getInt(input.Limit),
	}
}
