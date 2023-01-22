package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/requests"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// WebhookHandlerValidator validates models used in handlers.WebhookHandler
type WebhookHandlerValidator struct {
	validator
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewWebhookHandlerValidator creates a new handlers.WebhookHandler validator
func NewWebhookHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *WebhookHandlerValidator) {
	return &WebhookHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateIndex validates the requests.HeartbeatIndex request
func (validator *WebhookHandlerValidator) ValidateIndex(_ context.Context, request requests.WebhookIndex) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"limit": []string{
				"required",
				"numeric",
				"min:1",
				"max:100",
			},
			"skip": []string{
				"required",
				"numeric",
				"min:0",
			},
			"query": []string{
				"max:100",
			},
		},
	})
	return v.ValidateStruct()
}

// ValidateStore validates the requests.WebhookStore request
func (validator *WebhookHandlerValidator) ValidateStore(_ context.Context, request requests.WebhookStore) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"signing_key": []string{
				"required",
				"min:1",
				"max:255",
			},
			"url": []string{
				"required",
				"url",
				"max:255",
			},
			"events": []string{
				"required",
				webhookEventsRule,
			},
		},
	})
	return v.ValidateStruct()
}

// ValidateUpdate validates the requests.WebhookUpdate request
func (validator *WebhookHandlerValidator) ValidateUpdate(_ context.Context, request requests.WebhookUpdate) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"signing_key": []string{
				"required",
				"min:1",
				"max:255",
			},
			"webhookID": []string{
				"required",
				"uuid",
			},
			"url": []string{
				"required",
				"url",
				"max:255",
			},
			"events": []string{
				"required",
				webhookEventsRule,
			},
		},
	})
	return v.ValidateStruct()
}
