package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/http-sms-manager/pkg/requests"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// MessageThreadHandlerValidator validates models used in handlers.MessageThreadHandler
type MessageThreadHandlerValidator struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewMessageThreadHandlerValidator creates a new MessageThreadHandlerValidator
func NewMessageThreadHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *MessageThreadHandlerValidator) {
	return &MessageThreadHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateMessageThreadIndex validates the requests.MessageThreadIndex request
func (validator *MessageThreadHandlerValidator) ValidateMessageThreadIndex(_ context.Context, request requests.MessageThreadIndex) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"limit": []string{
				"required",
				"numeric",
				"min:1",
				"max:20",
			},
			"skip": []string{
				"required",
				"numeric",
				"min:0",
			},
			"query": []string{
				"required",
				"max:100",
			},
			"owner": []string{
				"required",
				phoneNumberRule,
			},
		},
		Messages: map[string][]string{
			"owner": {
				"regex:The 'owner' field must be a valid E.164 phone number: https://en.wikipedia.org/wiki/E.164",
			},
		},
	})
	return v.ValidateStruct()
}
