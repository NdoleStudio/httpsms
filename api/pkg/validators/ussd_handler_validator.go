package validators

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// USSDHandlerValidator validates USSD requests
type USSDHandlerValidator struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewUSSDHandlerValidator creates a new USSDHandlerValidator
func NewUSSDHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *USSDHandlerValidator) {
	return &USSDHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateReceive validates a USSD receive request
func (v *USSDHandlerValidator) ValidateReceive(ctx context.Context, request requests.USSDReceive) url.Values {
	_, span := v.tracer.Start(ctx)
	defer span.End()

	errs := url.Values{}

	if len(request.From) == 0 {
		errs.Add("from", "cannot be blank")
	}

	if len(request.To) == 0 {
		errs.Add("to", "cannot be blank")
	}

	if len(request.Content) == 0 {
		errs.Add("content", "cannot be blank")
	}

	if len(request.SessionID) == 0 {
		errs.Add("session_id", "cannot be blank")
	}

	if request.Timestamp.IsZero() {
		errs.Add("timestamp", "cannot be blank")
	}

	return errs
}

// ValidateSend validates a USSD send request
func (v *USSDHandlerValidator) ValidateSend(ctx context.Context, request requests.USSDSend) url.Values {
	_, span := v.tracer.Start(ctx)
	defer span.End()

	errs := url.Values{}

	if len(request.From) == 0 {
		errs.Add("from", "cannot be blank")
	}

	if len(request.To) == 0 {
		errs.Add("to", "cannot be blank")
	}

	if len(request.Content) == 0 {
		errs.Add("content", "cannot be blank")
	}

	if len(request.SessionID) == 0 {
		errs.Add("session_id", "cannot be blank")
	}

	return errs
}

// ValidateIndex validates a USSD index request
func (v *USSDHandlerValidator) ValidateIndex(ctx context.Context, request requests.USSDIndex) url.Values {
	_, span := v.tracer.Start(ctx)
	defer span.End()

	errs := url.Values{}

	if request.Limit < 0 || request.Limit > 20 {
		errs.Add("limit", "must be between 0 and 20")
	}

	if request.Skip < 0 {
		errs.Add("skip", "cannot be less than 0")
	}

	if len(request.PhoneID) > 0 {
		if matched, _ := regexp.MatchString("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$", request.PhoneID); !matched {
			errs.Add("phone_id", "must be a valid UUID")
		}
	}

	return errs
}

// ValidateDelete validates a USSD delete request
func (v *USSDHandlerValidator) ValidateDelete(ctx context.Context, request requests.USSDDelete) url.Values {
	_, span := v.tracer.Start(ctx)
	defer span.End()

	errs := url.Values{}

	if matched, _ := regexp.MatchString("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$", request.USSDID); !matched {
		errs.Add("ussd_id", "must be a valid UUID")
	}

	return errs
}