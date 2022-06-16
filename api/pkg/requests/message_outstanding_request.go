package requests

import (
	"strconv"
	"strings"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
)

// MessageOutstanding is the payload fetching outstanding entities.Message
type MessageOutstanding struct {
	Limit string `json:"limit" query:"limit"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *MessageOutstanding) Sanitize() MessageOutstanding {
	if strings.TrimSpace(input.Limit) == "" {
		input.Limit = "1"
	}

	if input.Limit != "1" {
		input.Limit = "2"
	}

	return *input
}

// ToGetOutstandingParams converts request to services.MessageGetOutstandingParams
func (input *MessageOutstanding) ToGetOutstandingParams(source string) services.MessageGetOutstandingParams {
	return services.MessageGetOutstandingParams{
		Source: source,
		Limit:  input.getLimit(),
	}
}

// getLimit gets the take as a string
func (input *MessageOutstanding) getLimit() int {
	val, _ := strconv.Atoi(input.Limit)
	return val
}
