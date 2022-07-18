package validators

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"

	"github.com/NdoleStudio/http-sms-manager/pkg/requests"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// MessageHandlerValidator validates models used in handlers.MessageHandler
type MessageHandlerValidator struct {
	validator
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

// ValidateMessageReceive validates the requests.MessageReceive request
func (validator MessageHandlerValidator) ValidateMessageReceive(_ context.Context, request requests.MessageReceive) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"to": []string{
				"required",
				phoneNumberRule,
			},
			"from": []string{
				"required",
			},
			"content": []string{
				"required",
				"min:1",
				"max:500",
			},
		},
	})

	return v.ValidateStruct()
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
	})

	return v.ValidateStruct()
}

// ValidateMessageOutstanding validates the requests.MessageOutstanding request
func (validator MessageHandlerValidator) ValidateMessageOutstanding(_ context.Context, request requests.MessageOutstanding) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"message_id": []string{
				"required",
				"uuid",
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
			"contact": []string{
				"required",
				"min:1",
			},
			"query": []string{
				"max:100",
			},
			"owner": []string{
				"required",
				phoneNumberRule,
			},
		},
	})
	return v.ValidateStruct()
}

// ValidateMessageEvent validates the requests.MessageEvent request
func (validator MessageHandlerValidator) ValidateMessageEvent(_ context.Context, request requests.MessageEvent) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"event_name": []string{
				"required",
				"in:" + strings.Join([]string{
					entities.MessageEventNameSent,
					entities.MessageEventNameFailed,
					entities.MessageEventNameDelivered,
				}, ","),
			},
			"messageID": []string{
				"required",
				"uuid",
			},
		},
	})
	return v.ValidateStruct()
}
