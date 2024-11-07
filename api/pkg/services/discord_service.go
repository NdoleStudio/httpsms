package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/gofiber/fiber/v2"

	"github.com/NdoleStudio/httpsms/pkg/discord"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// DiscordService is responsible for handling discordIntegrations
type DiscordService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	client     *discord.Client
	dispatcher *EventDispatcher
	repository repositories.DiscordRepository
}

// NewDiscordService creates a new DiscordService
func NewDiscordService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *discord.Client,
	repository repositories.DiscordRepository,
	dispatcher *EventDispatcher,
) (s *DiscordService) {
	return &DiscordService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		client:     client,
		dispatcher: dispatcher,
		repository: repository,
	}
}

// GetByServerID fetches the entities.Discord by the serverID
func (service *DiscordService) GetByServerID(ctx context.Context, serverID string) (*entities.Discord, error) {
	ctx, span, _ := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()
	return service.repository.FindByServerID(ctx, serverID)
}

// DeleteAllForUser deletes all entities.Discord for an entities.UserID.
func (service *DiscordService) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.repository.DeleteAllForUser(ctx, userID); err != nil {
		msg := fmt.Sprintf("could not delete all [entities.Discord] for user with ID [%s]", userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted all [entities.Discord] for user with ID [%s]", userID))
	return nil
}

// Index fetches the entities.Discord for an entities.UserID
func (service *DiscordService) Index(ctx context.Context, userID entities.UserID, params repositories.IndexParams) ([]*entities.Discord, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	discordIntegrations, err := service.repository.Index(ctx, userID, params)
	if err != nil {
		msg := fmt.Sprintf("could not fetch discord integrations with params [%+#v]", params)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] discord integrations with prams [%+#v]", len(discordIntegrations), params))
	return discordIntegrations, nil
}

// Delete an entities.Discord
func (service *DiscordService) Delete(ctx context.Context, userID entities.UserID, discordID uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if _, err := service.repository.Load(ctx, userID, discordID); err != nil {
		msg := fmt.Sprintf("cannot load discord integration with userID [%s] and discordID [%s]", userID, discordID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	if err := service.repository.Delete(ctx, userID, discordID); err != nil {
		msg := fmt.Sprintf("cannot delete discord integration with id [%s] and discordID [%s]", discordID, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted discord integration with id [%s] and user id [%s]", discordID, userID))
	return nil
}

// DiscordStoreParams are parameters for creating a new entities.Discord
type DiscordStoreParams struct {
	UserID            entities.UserID
	Name              string
	ServerID          string
	IncomingChannelID string
}

// Store a new entities.Discord
func (service *DiscordService) Store(ctx context.Context, params *DiscordStoreParams) (*entities.Discord, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.createSlashCommand(ctx, params.ServerID); err != nil {
		msg := fmt.Sprintf("cannot create slash command for server [%s]", params.ServerID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	discordIntegration := &entities.Discord{
		ID:                uuid.New(),
		UserID:            params.UserID,
		Name:              params.Name,
		ServerID:          params.ServerID,
		IncomingChannelID: params.IncomingChannelID,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	if err := service.repository.Save(ctx, discordIntegration); err != nil {
		msg := fmt.Sprintf("cannot save discord integration with id [%s]", discordIntegration.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("discord integration saved with id [%s] in the [%T]", discordIntegration.ID, service.repository))
	return discordIntegration, nil
}

func (service *DiscordService) createSlashCommand(ctx context.Context, serverID string) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	command, _, err := service.client.Application.CreateCommand(ctx, serverID, &discord.CommandCreateRequest{
		Name:        "httpsms",
		Type:        1,
		Description: "Send an SMS via httpsms.com",
		Options: []discord.CommandCreateRequestOption{
			{
				Name:        "from",
				Description: "Sender phone number",
				Type:        3,
				Required:    true,
			},
			{
				Name:        "to",
				Description: "Recipient phone number",
				Type:        3,
				Required:    true,
			},
			{
				Name:        "message",
				Description: "Text message content",
				Type:        3,
				Required:    true,
			},
		},
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create slash command for server [%s]", serverID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("upserted a slash command with ID [%s] for discord server [%s] and applicationID [%s]", command.ID, serverID, command.ApplicationID))
	return nil
}

// DiscordUpdateParams are parameters for updating an entities.Discord
type DiscordUpdateParams struct {
	UserID            entities.UserID
	Name              string
	ServerID          string
	IncomingChannelID string
	DiscordID         uuid.UUID
}

// Update an entities.Discord
func (service *DiscordService) Update(ctx context.Context, params *DiscordUpdateParams) (*entities.Discord, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	discordIntegration, err := service.repository.Load(ctx, params.UserID, params.DiscordID)
	if err != nil {
		msg := fmt.Sprintf("cannot load discord integration with userID [%s] and discordID [%s]", params.UserID, params.DiscordID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	if err = service.createSlashCommand(ctx, params.ServerID); err != nil {
		msg := fmt.Sprintf("cannot create slash command for server [%s]", params.ServerID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	discordIntegration.Name = params.Name
	discordIntegration.ServerID = params.ServerID
	discordIntegration.IncomingChannelID = params.IncomingChannelID

	if err = service.repository.Save(ctx, discordIntegration); err != nil {
		msg := fmt.Sprintf("cannot save discord integration with id [%s] after update", discordIntegration.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("discord integration updated with id [%s] in the [%T]", discordIntegration.ID, service.repository))
	return discordIntegration, nil
}

// HandleMessageReceived sends an incoming SMS to a discord channel
func (service *DiscordService) HandleMessageReceived(ctx context.Context, userID entities.UserID, event cloudevents.Event) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	discordIntegrations, err := service.repository.FetchHavingIncomingChannel(ctx, userID)
	if err != nil {
		msg := fmt.Sprintf("cannot load discord integrations for user with ID [%s]", userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	if len(discordIntegrations) == 0 {
		ctxLogger.Info(fmt.Sprintf("user [%s] has no discord integration for event [%s]", userID, event.Type()))
		return nil
	}

	var wg sync.WaitGroup
	for _, discordIntegration := range discordIntegrations {
		wg.Add(1)
		go func(webhook *entities.Discord) {
			defer wg.Done()
			service.sendMessage(ctx, event, webhook)
		}(discordIntegration)
	}
	wg.Wait()

	return nil
}

func (service *DiscordService) sendMessage(ctx context.Context, event cloudevents.Event, discord *entities.Discord) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	payload := new(events.MessagePhoneReceivedPayload)
	if err := event.DataAs(payload); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot unmarshal event [%s] with ID [%s] into [%T]", event.Type(), event.ID(), payload)))
		return
	}

	request := service.createDiscordMessage(ctxLogger, payload)
	message, response, err := service.client.Channel.CreateMessage(ctx, discord.IncomingChannelID, request)
	if err != nil {
		msg := fmt.Sprintf("cannot send [%s] event to discord channel [%s] for user [%s]", event.Type(), discord.IncomingChannelID, discord.UserID)
		ctxLogger.Warn(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))

		eventPayload := &events.DiscordSendFailedPayload{
			DiscordID:        discord.ID,
			UserID:           discord.UserID,
			MessageID:        payload.MessageID,
			Owner:            payload.Owner,
			EventType:        event.Type(),
			ErrorMessage:     err.Error(),
			DiscordChannelID: discord.IncomingChannelID,
		}

		if response != nil {
			eventPayload.HTTPResponseStatusCode = &response.HTTPResponse.StatusCode
			eventPayload.ErrorMessage = string(*response.Body)
		}

		service.handleDiscordMessageFailed(ctx, event.Source(), eventPayload)
		return
	}

	ctxLogger.Info(fmt.Sprintf("sent discord message [%s] to channel [%s] for [%s] event with ID [%s]", message["id"].(string), discord.IncomingChannelID, event.Type(), event.ID()))
}

func (service *DiscordService) createDiscordMessage(ctxLogger telemetry.Logger, payload *events.MessagePhoneReceivedPayload) fiber.Map {
	return fiber.Map{
		"content": "✉ new message received",
		"embeds": []fiber.Map{
			{
				"fields": []fiber.Map{
					{
						"name":   "From:",
						"value":  service.getFormattedNumber(ctxLogger, payload.Contact),
						"inline": true,
					},
					{
						"name":   "To:",
						"value":  service.getFormattedNumber(ctxLogger, payload.Owner),
						"inline": true,
					},
					{
						"name":  "Content:",
						"value": payload.Content,
					},
					{
						"name":  "MessageID:",
						"value": payload.MessageID,
					},
				},
			},
		},
	}
}

func (service *DiscordService) handleDiscordMessageFailed(ctx context.Context, source string, payload *events.DiscordSendFailedPayload) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	event, err := service.createEvent(events.EventTypeDiscordSendFailed, source, payload)
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for user with id [%s]", events.EventTypeDiscordSendFailed, payload.UserID)
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for user with id [%s]", event.Type(), payload.UserID)
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return
	}

	ctxLogger.Info(fmt.Sprintf("dispatched event [%s] for user with id [%s]", event.Type(), payload.UserID))
}
