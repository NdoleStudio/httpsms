package validators

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/cache"
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
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	phoneService   *services.PhoneService
	tokenValidator *TurnstileTokenValidator
	cache          cache.Cache
}

// NewMessageHandlerValidator creates a new handlers.MessageHandler validator
func NewMessageHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	phoneService *services.PhoneService,
	tokenValidator *TurnstileTokenValidator,
	appCache cache.Cache,
) (v *MessageHandlerValidator) {
	return &MessageHandlerValidator{
		logger:         logger.WithService(fmt.Sprintf("%T", v)),
		tracer:         tracer,
		phoneService:   phoneService,
		tokenValidator: tokenValidator,
		cache:          appCache,
	}
}

const (
	maxAttachmentCount     = 10
	maxAttachmentSize      = (3 * 1024 * 1024) / 2 // 1.5 MB per attachment
	maxTotalAttachmentSize = 3 * 1024 * 1024       // 3 MB total
)

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
			"content": func() []string {
				if len(request.Attachments) > 0 {
					return []string{"max:2048"}
				}
				return []string{"required", "min:1", "max:2048"}
			}(),
			"sim": []string{
				"required",
				"in:" + strings.Join([]string{
					string(entities.SIM1),
					string(entities.SIM2),
				}, ","),
			},
		},
	})

	errors := v.ValidateStruct()

	if len(request.Attachments) > 0 {
		attachmentErrors := validator.validateAttachments(request.Attachments)
		for key, values := range attachmentErrors {
			for _, value := range values {
				errors.Add(key, value)
			}
		}
	}

	return errors
}

func (validator MessageHandlerValidator) validateAttachments(attachments []requests.MessageAttachment) url.Values {
	errors := url.Values{}
	allowedTypes := repositories.AllowedContentTypes()

	if len(attachments) > maxAttachmentCount {
		errors.Add("attachments", fmt.Sprintf("attachment count [%d] exceeds maximum of [%d]", len(attachments), maxAttachmentCount))
		return errors
	}

	totalSize := 0
	for i, attachment := range attachments {
		if !allowedTypes[attachment.ContentType] {
			errors.Add("attachments", fmt.Sprintf("attachment [%d] has unsupported content type [%s]", i, attachment.ContentType))
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(attachment.Content)
		if err != nil {
			errors.Add("attachments", fmt.Sprintf("attachment [%d] has invalid base64 content", i))
			continue
		}

		if len(decoded) > maxAttachmentSize {
			errors.Add("attachments", fmt.Sprintf("attachment [%d] size [%d] exceeds maximum of [%d] bytes", i, len(decoded), maxAttachmentSize))
		}

		totalSize += len(decoded)
	}

	if totalSize > maxTotalAttachmentSize {
		errors.Add("attachments", fmt.Sprintf("total attachment size [%d] exceeds maximum of [%d] bytes", totalSize, maxTotalAttachmentSize))
	}

	return errors
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
			"attachments": []string{
				"max:10",
				multipleAttachmentURLRule,
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

	if request.SendAt != nil && request.SendAt.After(time.Now().Add(480*time.Hour)) {
		result.Add("send_at", "the scheduled time cannot be more than 20 days (480 hours) in the future")
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
			"attachments": []string{
				"max:10",
				multipleAttachmentURLRule,
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
func (validator MessageHandlerValidator) ValidateMessageSearch(ctx context.Context, request requests.MessageSearch) url.Values {
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
			"token": []string{
				"required",
			},
		},
	})

	errors := v.ValidateStruct()
	if len(errors) > 0 {
		return errors
	}

	if !validator.tokenValidator.ValidateToken(ctx, request.IPAddress, request.Token) {
		errors.Add("token", "The captcha token from turnstile is invalid")
	}

	return errors
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
