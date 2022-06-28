package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/http-sms-manager/pkg/requests"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// PhoneHandlerValidator validates models used in handlers.PhoneHandler
type PhoneHandlerValidator struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewPhoneHandlerValidator creates a new handlers.PhoneHandler validator
func NewPhoneHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *PhoneHandlerValidator) {
	return &PhoneHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateIndex validates the requests.HeartbeatIndex request
func (validator *PhoneHandlerValidator) ValidateIndex(_ context.Context, request requests.PhoneIndex) url.Values {
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
				"max:100",
			},
		},
	})
	return v.ValidateStruct()
}

// ValidateUpsert validates requests.PhoneUpsert
func (validator *PhoneHandlerValidator) ValidateUpsert(_ context.Context, request requests.PhoneUpsert) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"phone_number": []string{
				"required",
				phoneNumberRule,
			},
			"fcm_token": []string{
				"required",
				"min:1",
				"max:1000",
			},
		},
		Messages: map[string][]string{
			"phone_number": {
				"regex: The phone_number field must be a valid E.164 phone number: https://en.wikipedia.org/wiki/E.164",
			},
		},
	})

	return v.ValidateStruct()
}
