package validators

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// PhoneHandlerValidator validates models used in handlers.PhoneHandler
type PhoneHandlerValidator struct {
	validator
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
				"min:0",
				"max:1000",
			},
			"messages_per_minute": []string{
				"min:0",
				"max:60",
			},
			"max_send_attempts": []string{
				"min:0",
				"max:5",
			},
			"sim": []string{
				"required",
				"in:" + strings.Join([]string{entities.SIM1.String(), entities.SIM2.String()}, ","),
			},
			"message_expiration_seconds": []string{
				"min:60",
				"max:3600",
			},
		},
	})

	result := v.ValidateStruct()
	if len(result) > 0 {
		return result
	}

	if request.MaxSendAttempts > 0 && request.MessageExpirationSeconds == 0 {
		result.Add("message_expiration_seconds", "message_expiration_seconds cannot be 0 when max_send_attempts is greater than 0")
	}

	return result
}

// ValidateFCMToken validates requests.PhoneFCMToken
func (validator *PhoneHandlerValidator) ValidateFCMToken(_ context.Context, request requests.PhoneFCMToken) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"phone_number": []string{
				"required",
				phoneNumberRule,
			},
			"fcm_token": []string{
				"min:0",
				"max:1000",
			},
			"sim": []string{
				"required",
				"in:" + strings.Join([]string{entities.SIM1.String(), entities.SIM2.String()}, ","),
			},
		},
	})

	return v.ValidateStruct()
}

// ValidateDelete ValidateUpsert validates requests.PhoneDelete
func (validator *PhoneHandlerValidator) ValidateDelete(_ context.Context, request requests.PhoneDelete) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"phoneID": []string{
				"required",
				"uuid",
			},
		},
	})

	return v.ValidateStruct()
}
