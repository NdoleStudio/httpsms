package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/http-sms-manager/pkg/requests"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
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
				"regex:^\\+[1-9]\\d{10,14}$",
			},
			"from": []string{
				"required",
				"regex:^\\+[1-9]\\d{1,14}$",
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
