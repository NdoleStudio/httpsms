package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/requests"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// BillingHandlerValidator validates models used in handlers.BillingHandler
type BillingHandlerValidator struct {
	validator
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewBillingHandlerValidator creates a new handlers.BillingHandler validator
func NewBillingHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *BillingHandlerValidator) {
	return &BillingHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateHistory validates the requests.BillingUsageHistory request
func (validator *BillingHandlerValidator) ValidateHistory(_ context.Context, request requests.BillingUsageHistory) url.Values {
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
		},
	})
	return v.ValidateStruct()
}
