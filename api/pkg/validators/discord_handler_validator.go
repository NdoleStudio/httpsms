package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/discord"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/requests"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/thedevsaddam/govalidator"
)

// DiscordHandlerValidator validates models used in handlers.DiscordHandler
type DiscordHandlerValidator struct {
	validator
	client *discord.Client
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewDiscordHandlerValidator creates a new handlers.DiscordHandler validator
func NewDiscordHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *discord.Client,
) (v *DiscordHandlerValidator) {
	return &DiscordHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
		client: client,
	}
}

//// ValidateIndex validates the requests.HeartbeatIndex request
//func (validator *DiscordHandlerValidator) ValidateIndex(_ context.Context, request requests.DiscordIndex) url.Values {
//	v := govalidator.New(govalidator.Options{
//		Data: &request,
//		Rules: govalidator.MapData{
//			"limit": []string{
//				"required",
//				"numeric",
//				"min:1",
//				"max:100",
//			},
//			"skip": []string{
//				"required",
//				"numeric",
//				"min:0",
//			},
//			"query": []string{
//				"max:100",
//			},
//		},
//	})
//	return v.ValidateStruct()
//}

// ValidateStore validates the requests.DiscordStore request
func (validator *DiscordHandlerValidator) ValidateStore(ctx context.Context, request requests.DiscordStore) url.Values {
	ctx, span, ctxLogger := validator.tracer.StartWithLogger(ctx, validator.logger)
	defer span.End()

	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			"name": []string{
				"required",
				"min:1",
				"max:255",
			},
			"server_id": []string{
				"required",
				"numeric",
				"max:255",
			},
			"incoming_channel_id": []string{
				"required",
				"max:255",
				"numeric",
			},
		},
	})

	result := v.ValidateStruct()
	if len(result) > 0 {
		return result
	}

	if _, _, err := validator.client.Channel.Get(ctx, request.IncomingChannelID); err != nil {
		msg := fmt.Sprintf("cannot fetch discord channel with ID [%s]", request.IncomingChannelID)
		ctxLogger.Error(validator.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		result.Add("incoming_channel_id", fmt.Sprintf("cannot fetch discord channel with ID [%s] make sure the bot has access to the channel"))
	}

	if _, _, err := validator.client.Guild.Get(ctx, request.ServerID); err != nil {
		msg := fmt.Sprintf("cannot fetch discord channel with ID [%s]", request.IncomingChannelID)
		ctxLogger.Error(validator.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		result.Add("server_id", fmt.Sprintf("cannot fetch discord channel with ID [%s] make sure the bot has access to the channel"))
	}

	return result
}

//// ValidateUpdate validates the requests.DiscordUpdate request
//func (validator *DiscordHandlerValidator) ValidateUpdate(_ context.Context, request requests.DiscordUpdate) url.Values {
//	v := govalidator.New(govalidator.Options{
//		Data: &request,
//		Rules: govalidator.MapData{
//			"signing_key": []string{
//				"required",
//				"min:1",
//				"max:255",
//			},
//			"webhookID": []string{
//				"required",
//				"uuid",
//			},
//			"url": []string{
//				"required",
//				"url",
//				"max:255",
//			},
//			"events": []string{
//				"required",
//				webhookEventsRule,
//			},
//		},
//	})
//	return v.ValidateStruct()
//}
