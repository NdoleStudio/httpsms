package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/http-sms-manager/pkg/requests"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
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
