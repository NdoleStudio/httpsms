package validators

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// MessageHandlerValidator validates models used in handlers.MessageHandler
type MessageHandlerValidator struct {
	validator
	logger       telemetry.Logger
	tracer       telemetry.Tracer
	phoneService *services.PhoneService
}

// NewMessageHandlerValidator creates a new handlers.MessageHandler validator
func NewMessageHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	phoneService *services.PhoneService,
) (v *MessageHandlerValidator) {
	return &MessageHandlerValidator{
		logger:       logger.WithService(fmt.Sprintf("%T", v)),
		tracer:       tracer,
		phoneService: phoneService,
	}
}

// ValidateMessageReceive validates the requests.MessageReceive request
func (validator MessageHandlerValidator) ValidateMessageReceive(_ context.Context, request requests.MessageReceive) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"to": []string{
				"required",
				phoneNumberRule,
			},
			"from": []string{
				"required",
			},
			"content": []string{
				"required",
				"min:1",
				"max:2048",
			},
			"sim": []string{
				"required",
				"in:" + strings.Join([]string{
					string(entities.SIM1),
					string(entities.SIM2),
				}, ","),
			},
		},
	})

	return v.ValidateStruct()
}

// ValidateMessageSend validates the requests.MessageSend request
func (validator MessageHandlerValidator) ValidateMessageSend(ctx context.Context, userID entities.UserID, request requests.MessageSend) url.Values {
	ctx, span := validator.tracer.Start(ctx)
	defer span.End()

	ctxLogger := validator.tracer.CtxLogger(validator.logger, span)

	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"to": []string{
				"required",
				contactPhoneNumberRule,
			},
			"request_id": []string{
				"max:255",
			},
			"from": []string{
				"required",
				phoneNumberRule,
			},
			"content": []string{
				"required",
				"min:1",
				"max:2048",
			},
		},
	})

	result := v.ValidateStruct()
	if len(result) != 0 {
		return result
	}

	_, err := validator.phoneService.Load(ctx, userID, request.From)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		result.Add("from", fmt.Sprintf("no phone found with with 'from' number [%s]. install the android app on your phone to start sending messages", request.From))
	}

	if err != nil {
		ctxLogger.Error(validator.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("could not load phone for user [%s] and phone [%s]", userID, request.From))))
		result.Add("from", fmt.Sprintf("could not validate 'from' number [%s], please try again later", request.From))
	}

	return result
}

// ValidateMessageBulkSend validates the requests.MessageBulkSend request
func (validator MessageHandlerValidator) ValidateMessageBulkSend(ctx context.Context, userID entities.UserID, request requests.MessageBulkSend) url.Values {
	ctx, span := validator.tracer.Start(ctx)
	defer span.End()

	ctxLogger := validator.tracer.CtxLogger(validator.logger, span)

	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"to": []string{
				"required",
				"max:1000",
				"min:1",
				multipleContactPhoneNumberRule,
			},
			"from": []string{
				"required",
				phoneNumberRule,
			},
			"content": []string{
				"required",
				"min:1",
				"max:1024",
			},
		},
	})

	result := v.ValidateStruct()
	if len(result) != 0 {
		return result
	}

	_, err := validator.phoneService.Load(ctx, userID, request.From)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		result.Add("from", fmt.Sprintf("no phone found with with 'from' number [%s]. Install the android app on your phone to start sending messages", request.From))
	}

	if err != nil {
		ctxLogger.Error(validator.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("could not load phone for user [%s] and phone [%s]", userID, request.From))))
		result.Add("from", fmt.Sprintf("could not validate 'from' number [%s], please try again later", request.From))
	}

	return result
}

// ValidateMessageOutstanding validates the requests.MessageOutstanding request
func (validator MessageHandlerValidator) ValidateMessageOutstanding(_ context.Context, request requests.MessageOutstanding) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"message_id": []string{
				"required",
				"uuid",
			},
		},
	})
	return v.ValidateStruct()
}

// ValidateMessageIndex validates the requests.MessageIndex request
func (validator MessageHandlerValidator) ValidateMessageIndex(_ context.Context, request requests.MessageIndex) url.Values {
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
			"contact": []string{
				"required",
				"min:1",
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

// ValidateMessageSearch validates the requests.MessageSearch request
func (validator MessageHandlerValidator) ValidateMessageSearch(_ context.Context, request requests.MessageSearch) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"owners": []string{
				multipleContactPhoneNumberRule,
			},
			"types": []string{
				multipleInRule + ":" + strings.Join([]string{
					entities.MessageTypeCallMissed,
					entities.MessageTypeMobileOriginated,
					entities.MessageTypeMobileTerminated,
				}, ","),
			},
			"statuses": []string{
				multipleInRule + ":" + strings.Join([]string{
					entities.MessageStatusPending,
					entities.MessageStatusSent,
					entities.MessageStatusDelivered,
					entities.MessageStatusFailed,
					entities.MessageStatusExpired,
					entities.MessageStatusReceived,
				}, ","),
			},
			"sort_by": []string{
				"in:" + strings.Join([]string{
					"created_at",
					"owner",
					"contact",
					"type",
					"status",
				}, ","),
			},
			"limit": []string{
				"required",
				"numeric",
				"min:1",
				"max:200",
			},
			"skip": []string{
				"required",
				"numeric",
				"min:0",
			},
			"query": []string{
				"max:20",
			},
		},
	})
	return v.ValidateStruct()
}

// ValidateMessageEvent validates the requests.MessageEvent request
func (validator MessageHandlerValidator) ValidateMessageEvent(_ context.Context, request requests.MessageEvent) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"event_name": []string{
				"required",
				"in:" + strings.Join([]string{
					string(entities.MessageEventNameSent),
					string(entities.MessageEventNameFailed),
					string(entities.MessageEventNameDelivered),
				}, ","),
			},
			"messageID": []string{
				"required",
				"uuid",
			},
		},
	})
	return v.ValidateStruct()
}

// ValidateCallMissed validates the requests.MessageCallMissed request
func (validator MessageHandlerValidator) ValidateCallMissed(_ context.Context, request requests.MessageCallMissed) url.Values {
	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"to": []string{
				"required",
				phoneNumberRule,
			},
			"from": []string{
				"required",
			},
			"sim": []string{
				"required",
				"in:" + strings.Join([]string{
					string(entities.SIM1),
					string(entities.SIM2),
				}, ","),
			},
		},
	})

	return v.ValidateStruct()
}
