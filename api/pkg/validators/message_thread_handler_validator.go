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
	validator
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
			"is_archived": []string{
				"required",
				"in:true,false",
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

// ValidateUpdate validates requests.UserUpdate
func (validator *MessageThreadHandlerValidator) ValidateUpdate(_ context.Context, request requests.MessageThreadUpdate) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"messageThreadID": []string{
				"required",
				"uuid",
			},
		},
	})

	return v.ValidateStruct()
}
