package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

type SendScheduleHandlerValidator struct {
	validator
	logger telemetry.Logger
	tracer telemetry.Tracer
}

func NewSendScheduleHandlerValidator(logger telemetry.Logger, tracer telemetry.Tracer) *SendScheduleHandlerValidator {
	return &SendScheduleHandlerValidator{logger: logger.WithService(fmt.Sprintf("%T", &SendScheduleHandlerValidator{})), tracer: tracer}
}

func (validator *SendScheduleHandlerValidator) ValidateUpsert(_ context.Context, request requests.SendScheduleUpsert) url.Values {
	v := govalidator.New(govalidator.Options{Data: &request, Rules: govalidator.MapData{"name": {"required", "min:1", "max:100"}, "timezone": {"required", "min:1", "max:100"}}})
	errors := v.ValidateStruct()
	if len(request.Windows) == 0 {
		errors.Add("windows", "at least one send window is required")
	}
	return errors
}
