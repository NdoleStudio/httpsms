package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
)

// ContactIndex lists contacts for a user.
type ContactIndex struct {
	request
	Skip           string `json:"skip" query:"skip"`
	Query          string `json:"query" query:"query"`
	SortBy         string `json:"sort_by" query:"sort_by"`
	SortDescending bool   `json:"sort_descending" query:"sort_descending"`
	Limit          string `json:"limit" query:"limit"`
}

// Sanitize sets defaults for the list request.
func (input *ContactIndex) Sanitize() ContactIndex {
	input.Query = strings.TrimSpace(input.Query)
	input.SortBy = strings.TrimSpace(input.SortBy)
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
		Skip:           input.getInt(input.Skip),
		Query:          input.Query,
		SortBy:         input.SortBy,
		SortDescending: input.SortDescending,
		Limit:          input.getInt(input.Limit),
	}
}
