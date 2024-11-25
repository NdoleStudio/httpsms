package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/requests"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// HeartbeatHandlerValidator validates models used in handlers.HeartbeatHandler
type HeartbeatHandlerValidator struct {
	validator
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewHeartbeatHandlerValidator creates a new handlers.MessageHandler validator
func NewHeartbeatHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *HeartbeatHandlerValidator) {
	return &HeartbeatHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateIndex validates the requests.HeartbeatIndex request
func (validator *HeartbeatHandlerValidator) ValidateIndex(_ context.Context, request requests.HeartbeatIndex) url.Values {
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
			"owner": []string{
				"required",
				phoneNumberRule,
			},
		},
	})
	return v.ValidateStruct()
}

// ValidateStore validates the requests.HeartbeatStore request
func (validator *HeartbeatHandlerValidator) ValidateStore(_ context.Context, request requests.HeartbeatStore) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"phone_numbers": []string{
				"required",
				"max:2",
				"min:1",
				multiplePhoneNumberRule,
			},
		},
	})
	return v.ValidateStruct()
}
