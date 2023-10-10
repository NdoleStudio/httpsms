package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/requests"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// WebhookHandlerValidator validates models used in handlers.WebhookHandler
type WebhookHandlerValidator struct {
	validator
	logger       telemetry.Logger
	tracer       telemetry.Tracer
	phoneService *services.PhoneService
}

// NewWebhookHandlerValidator creates a new handlers.WebhookHandler validator
func NewWebhookHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	phoneService *services.PhoneService,
) (v *WebhookHandlerValidator) {
	return &WebhookHandlerValidator{
		logger:       logger.WithService(fmt.Sprintf("%T", v)),
		tracer:       tracer,
		phoneService: phoneService,
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
func (validator *WebhookHandlerValidator) ValidateStore(ctx context.Context, userID entities.UserID, request requests.WebhookStore) url.Values {
	ctx, span := validator.tracer.Start(ctx)
	defer span.End()

	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"signing_key": []string{
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
			"phone_numbers": []string{
				"required",
				multipleContactPhoneNumberRule,
			},
		},
	})

	result := v.ValidateStruct()
	if len(result) > 0 {
		return result
	}

	for _, address := range request.PhoneNumbers {
		_, err := validator.phoneService.Load(ctx, userID, address)
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			result.Add("from", fmt.Sprintf("The phone number [%s] is not available in your account. Install the android app on your phone to store a webhook with this phone number", address))
		}
	}
	return result
}

// ValidateUpdate validates the requests.WebhookUpdate request
func (validator *WebhookHandlerValidator) ValidateUpdate(ctx context.Context, userID entities.UserID, request requests.WebhookUpdate) url.Values {
	ctx, span := validator.tracer.Start(ctx)
	defer span.End()

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
			"phone_numbers": []string{
				"required",
				multipleContactPhoneNumberRule,
			},
		},
	})

	result := v.ValidateStruct()
	if len(result) > 0 {
		return result
	}

	for _, address := range request.PhoneNumbers {
		_, err := validator.phoneService.Load(ctx, userID, address)
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			result.Add("from", fmt.Sprintf("The phone number [%s] is not available in your account. Install the android app on your phone to store a webhook with this phone number", address))
		}
	}
	return result
}
