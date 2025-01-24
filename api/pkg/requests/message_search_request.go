package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/NdoleStudio/httpsms/pkg/repositories"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

// MessageSearch is the payload fetching entities.Message
type MessageSearch struct {
	request
	Skip           string   `json:"skip" query:"skip"`
	Owners         []string `json:"owners" query:"owners"`
	Types          []string `json:"types" query:"types"`
	Statuses       []string `json:"statuses" query:"statuses"`
	Query          string   `json:"query" query:"query"`
	SortBy         string   `json:"sort_by" query:"sort_by"`
	SortDescending bool     `json:"sort_descending" query:"sort_descending"`
	Limit          string   `json:"limit" query:"limit"`

	IPAddress string `json:"ip_address" swaggerignore:"true"`
	Token     string `json:"token" swaggerignore:"true"`
}

// Sanitize sets defaults to MessageSearch
func (input *MessageSearch) Sanitize() MessageSearch {
	if strings.TrimSpace(input.Limit) == "" {
		input.Limit = "100"
	}

	input.Query = strings.TrimSpace(input.Query)

	input.Skip = strings.TrimSpace(input.Skip)
	if input.Skip == "" {
		input.Skip = "0"
	}

	return *input
}

// ToSearchParams converts request to services.MessageSearchParams
func (input *MessageSearch) ToSearchParams(userID entities.UserID) *services.MessageSearchParams {
	var types []entities.MessageType
	for _, t := range input.Types {
		types = append(types, entities.MessageType(t))
	}

	var statuses []entities.MessageStatus
	for _, s := range input.Statuses {
		statuses = append(statuses, entities.MessageStatus(s))
	}

	return &services.MessageSearchParams{
		IndexParams: repositories.IndexParams{
			Skip:           input.getInt(input.Skip),
			Query:          input.Query,
			SortBy:         input.SortBy,
			SortDescending: input.SortDescending,
			Limit:          input.getInt(input.Limit),
		},
		UserID:   userID,
		Owners:   input.Owners,
		Types:    types,
		Statuses: statuses,
	}
}
