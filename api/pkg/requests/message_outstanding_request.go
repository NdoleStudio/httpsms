package requests

import (
	"strconv"
	"strings"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
)

// MessageOutstanding is the payload fetching outstanding entities.Message
type MessageOutstanding struct {
	Take string `json:"take" query:"take"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *MessageOutstanding) Sanitize() MessageOutstanding {
	if strings.TrimSpace(input.Take) == "" {
		input.Take = "1"
	}
	return *input
}

// ToGetOutstandingParams converts request to services.MessageGetOutstandingParams
func (input *MessageOutstanding) ToGetOutstandingParams(source string) services.MessageGetOutstandingParams {
	return services.MessageGetOutstandingParams{
		Source: source,
		Take:   input.getTake(),
	}
}

// getTake gets the take as a string
func (input *MessageOutstanding) getTake() int {
	val, _ := strconv.Atoi(input.Take)
	return val
}
