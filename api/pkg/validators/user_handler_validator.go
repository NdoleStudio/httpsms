package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"github.com/thedevsaddam/govalidator"
)

// UserHandlerValidator validates models used in handlers.UserHandler
type UserHandlerValidator struct {
	validator
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.UserService
}

// NewUserHandlerValidator creates a new handlers.UserHandler validator
func NewUserHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.UserService,
) (v *UserHandlerValidator) {
	return &UserHandlerValidator{
		service: service,
		logger:  logger.WithService(fmt.Sprintf("%T", v)),
		tracer:  tracer,
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

// ValidatePaymentInvoice validates the requests.UserPaymentInvoice request
func (validator *UserHandlerValidator) ValidatePaymentInvoice(ctx context.Context, userID entities.UserID, request requests.UserPaymentInvoice) url.Values {
	ctx, span, ctxLogger := validator.tracer.StartWithLogger(ctx, validator.logger)
	defer span.End()

	rules := govalidator.MapData{
		"name": []string{
			"required",
			"min:1",
			"max:100",
		},
		"address": []string{
			"required",
			"min:1",
			"max:200",
		},
		"city": []string{
			"required",
			"min:1",
			"max:100",
		},
		"state": []string{
			"min:1",
			"max:100",
		},
		"country": []string{
			"required",
			"len:2",
		},
		"zip_code": []string{
			"required",
			"min:1",
			"max:20",
		},
		"notes": []string{
			"max:1000",
		},
	}
	if request.Country == "CA" {
		rules["state"] = []string{
			"required",
			"in:AB,BC,MB,NB,NL,NS,NT,NU,ON,PE,QC,SK,YT",
		}
	}

	if request.Country == "US" {
		rules["state"] = []string{
			"required",
			"in:AL,AK,AZ,AR,CA,CO,CT,DE,FL,GA,HI,ID,IL,IN,IA,KS,KY,LA,ME,MD,MA,MI,MN,MS,MO,MT,NE,NV,NH,NJ,NM,NY,NC,ND,OH,OK,OR,PA,RI,SC,SD,TN,TX,UT,VT,VA,WA,WV,WI,WY",
		}
	}

	v := govalidator.New(govalidator.Options{
		Data:  &request,
		Rules: rules,
	})

	validationErrors := v.ValidateStruct()
	if len(validationErrors) > 0 {
		return validationErrors
	}

	payments, err := validator.service.GetSubscriptionPayments(ctx, userID)
	if err != nil {
		msg := fmt.Sprintf("cannot get subscription payments for user with ID [%s]", userID)
		ctxLogger.Error(validator.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		validationErrors.Add("subscriptionInvoiceID", "failed to validate subscription payment invoice ID")
		return validationErrors
	}

	for _, payment := range payments {
		if payment.ID == request.SubscriptionInvoiceID {
			return validationErrors
		}
	}

	validationErrors.Add("subscriptionInvoiceID", "failed to validate the subscription payment invoice ID")
	return validationErrors
}
