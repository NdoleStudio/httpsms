package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// PhoneAPIKeyHandlerValidator validates models used in handlers.PhoneAPIKeyHandler
type PhoneAPIKeyHandlerValidator struct {
	validator
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewPhoneAPIKeyHandlerValidator creates a new handlers.PhoneAPIKeyHandler validator
func NewPhoneAPIKeyHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *PhoneAPIKeyHandlerValidator) {
	return &PhoneAPIKeyHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateStore validates requests.PhoneAPIKeyStoreRequest
func (validator *PhoneAPIKeyHandlerValidator) ValidateStore(_ context.Context, request requests.PhoneAPIKeyStoreRequest) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"name": []string{
				"required",
				"min:1",
				"max:60",
			},
		},
	})

	return v.ValidateStruct()
}

// ValidateIndex validates the requests.HeartbeatIndex request
func (validator *PhoneAPIKeyHandlerValidator) ValidateIndex(_ context.Context, request requests.PhoneAPIKeyIndex) url.Values {
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
