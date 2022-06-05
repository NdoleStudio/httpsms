package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/http-sms-manager/pkg/requests"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

const (
	phoneNumberRule = "regex:^\\+[1-9]\\d{1,14}$"
)

// MessageHandlerValidator validates models used in handlers.MessageHandler
type MessageHandlerValidator struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewMessageHandlerValidator creates a new handlers.MessageHandler validator
func NewMessageHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *MessageHandlerValidator) {
	return &MessageHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateMessageSend validates the requests.MessageSend request
func (validator MessageHandlerValidator) ValidateMessageSend(_ context.Context, request requests.MessageSend) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"to": []string{
				"required",
				phoneNumberRule,
			},
			"from": []string{
				"required",
				phoneNumberRule,
			},
			"content": []string{
				"required",
				"min:1",
				"max:500",
			},
		},
		Messages: map[string][]string{
			"to": {
				"regex: The 'to' field must be a valid E.164 phone number: https://en.wikipedia.org/wiki/E.164",
			},
			"from": {
				"regex: The 'from' field must be a valid E.164 phone number: https://en.wikipedia.org/wiki/E.164",
			},
		},
	})

	return v.ValidateStruct()
}

// ValidateMessageOutstanding validates the requests.MessageOutstanding request
func (validator MessageHandlerValidator) ValidateMessageOutstanding(_ context.Context, request requests.MessageOutstanding) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"limit": []string{
				"required",
				"numeric",
				"min:1",
				"max:20",
			},
		},
	})
	return v.ValidateStruct()
}

// ValidateMessageIndex validates the requests.MessageIndex request
func (validator MessageHandlerValidator) ValidateMessageIndex(_ context.Context, request requests.MessageIndex) url.Values {
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
			"from": []string{
				"required",
				"min:1",
			},
			"to": []string{
				"required",
				phoneNumberRule,
			},
		},
		Messages: map[string][]string{
			"to": {
				"regex:The 'to' field must be a valid E.164 phone number: https://en.wikipedia.org/wiki/E.164",
			},
		},
	})
	return v.ValidateStruct()
}
