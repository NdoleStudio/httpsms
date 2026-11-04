package requests

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/NdoleStudio/httpsms/pkg/repositories"

	"github.com/NdoleStudio/httpsms/pkg/services"
)

// MessageIncoming is the payload for fetching mobile-originated entities.Message
type MessageIncoming struct {
	request
	Skip           string   `json:"skip" query:"skip"`
	Owners         []string `json:"owners" query:"owners"`
	Statuses       []string `json:"statuses" query:"statuses"`
	Query          string   `json:"query" query:"query"`
	SortBy         string   `json:"sort_by" query:"sort_by"`
	SortDescending bool     `json:"sort_descending" query:"sort_descending"`
	Limit          string   `json:"limit" query:"limit"`
}

// Sanitize sets defaults to MessageIncoming
func (input *MessageIncoming) Sanitize() MessageIncoming {
	if strings.TrimSpace(input.Limit) == "" {
		input.Limit = "100"
	}

	input.Query = strings.TrimSpace(input.Query)

	input.Skip = strings.TrimSpace(input.Skip)
	if input.Skip == "" {
		input.Skip = "0"
	}

	input.SortBy = strings.TrimSpace(input.SortBy)
	if input.SortBy == "" {
		input.SortBy = "created_at"
		input.SortDescending = true
	}

	return *input
}

// ToSearchParams converts request to services.MessageSearchParams, forcing mobile-originated messages
func (input MessageIncoming) ToSearchParams(userID entities.UserID) *services.MessageSearchParams {
	statuses := make([]entities.MessageStatus, 0, len(input.Statuses))
	for _, status := range input.Statuses {
		statuses = append(statuses, entities.MessageStatus(status))
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
		Types:    []entities.MessageType{entities.MessageTypeMobileOriginated},
		Statuses: statuses,
	}
}
