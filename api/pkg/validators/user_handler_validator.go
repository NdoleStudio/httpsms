package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// UserHandlerValidator validates models used in handlers.UserHandler
type UserHandlerValidator struct {
	validator
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewUserHandlerValidator creates a new handlers.UserHandler validator
func NewUserHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *UserHandlerValidator) {
	return &UserHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateUpdate validates requests.UserUpdate
func (validator *UserHandlerValidator) ValidateUpdate(_ context.Context, request requests.UserUpdate) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"active_phone_id": []string{
				"uuid",
			},
		},
	})

	return v.ValidateStruct()
}
