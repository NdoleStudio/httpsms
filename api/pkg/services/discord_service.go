package services

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/discord"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/palantir/stacktrace"
)

// DiscordService is responsible for handling discordIntegrations
type DiscordService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	client     *discord.Client
	repository repositories.DiscordRepository
}

// NewDiscordService creates a new DiscordService
func NewDiscordService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *discord.Client,
	repository repositories.DiscordRepository,
) (s *DiscordService) {
	return &DiscordService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		client:     client,
		repository: repository,
	}
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
func (service *DiscordService) Delete(ctx context.Context, userID entities.UserID, webhookID uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if _, err := service.repository.Load(ctx, userID, webhookID); err != nil {
		msg := fmt.Sprintf("cannot load discord integration with userID [%s] and phoneID [%s]", userID, webhookID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	if err := service.repository.Delete(ctx, userID, webhookID); err != nil {
		msg := fmt.Sprintf("cannot delete discord integration with id [%s] and user id [%s]", webhookID, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted discord integration with id [%s] and user id [%s]", webhookID, userID))
	return nil
}

// DiscordStoreParams are parameters for creating a new entities.Discord
type DiscordStoreParams struct {
	UserID            entities.UserID
	Name              string
	ServerID          string
	IncomingChannelID string
	Events            pq.StringArray
}

// Store a new entities.Discord
func (service *DiscordService) Store(ctx context.Context, params *DiscordStoreParams) (*entities.Discord, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	discord := &entities.Discord{
		ID:                uuid.New(),
		UserID:            params.UserID,
		Name:              params.Name,
		ServerID:          params.ServerID,
		IncomingChannelID: params.IncomingChannelID,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	if err := service.repository.Save(ctx, discord); err != nil {
		msg := fmt.Sprintf("cannot save discord integration with id [%s]", discord.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("discord integration saved with id [%s] in the [%T]", discord.ID, service.repository))
	return discord, nil
}

//// DiscordUpdateParams are parameters for updating an entities.Discord
//type DiscordUpdateParams struct {
//	UserID     entities.UserID
//	SigningKey string
//	URL        string
//	Events     pq.StringArray
//	DiscordID  uuid.UUID
//}
//
//// Update an entities.Discord
//func (service *DiscordService) Update(ctx context.Context, params *DiscordUpdateParams) (*entities.Discord, error) {
//	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
//	defer span.End()
//
//	webhook, err := service.repository.Load(ctx, params.UserID, params.DiscordID)
//	if err != nil {
//		msg := fmt.Sprintf("cannot load webhook with userID [%s] and phoneID [%s]", params.UserID, params.DiscordID)
//		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
//	}
//
//	webhook.URL = params.URL
//	webhook.SigningKey = params.SigningKey
//	webhook.Events = params.Events
//
//	if err = service.repository.Save(ctx, webhook); err != nil {
//		msg := fmt.Sprintf("cannot save webhook with id [%s] after update", webhook.ID)
//		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
//	}
//
//	ctxLogger.Info(fmt.Sprintf("webhook updated with id [%s] in the [%T]", webhook.ID, service.repository))
//	return webhook, nil
//}
//
//// Send an event to a subscribed webhook
//func (service *DiscordService) Send(ctx context.Context, userID entities.UserID, event cloudevents.Event) error {
//	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
//	defer span.End()
//
//	discordIntegrations, err := service.repository.LoadByEvent(ctx, userID, event.Type())
//	if err != nil {
//		msg := fmt.Sprintf("cannot load discordIntegrations for userID [%s] and event [%s]", userID, event.Type())
//		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
//	}
//
//	if len(discordIntegrations) == 0 {
//		ctxLogger.Info(fmt.Sprintf("user [%s] has no webhook subscription to event [%s]", userID, event.Type()))
//		return nil
//	}
//
//	var wg sync.WaitGroup
//	for _, webhook := range discordIntegrations {
//		wg.Add(1)
//		go func(webhook *entities.Discord) {
//			defer wg.Done()
//			service.sendNotification(ctx, event, webhook)
//		}(webhook)
//	}
//	wg.Wait()
//
//	return nil
//}
//
//func (service *DiscordService) sendNotification(ctx context.Context, event cloudevents.Event, webhook *entities.Discord) {
//	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
//	defer span.End()
//
//	token, err := service.getAuthToken(webhook)
//	if err != nil {
//		msg := fmt.Sprintf("cannot generate auth token for user [%s] and webhook [%s]", webhook.UserID, webhook.ID)
//		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
//	}
//
//	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
//	defer cancel()
//
//	var response string
//	err = requests.URL(webhook.URL).
//		Client(service.client).
//		Bearer(token).
//		Header("X-Event-Type", event.Type()).
//		BodyJSON(service.getPayload(ctxLogger, event, webhook)).
//		ToString(&response).
//		Fetch(ctx)
//	if err != nil {
//		msg := fmt.Sprintf("cannot send [%s] event to webhook [%s] for user [%s]", event.Type(), webhook.URL, webhook.UserID)
//		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
//	}
//
//	ctxLogger.Info(fmt.Sprintf("sent webhook to url [%s] for event [%s] with ID [%s] and response [%s]", webhook.URL, event.Type(), event.ID(), response))
//}
//
//func (service *DiscordService) getPayload(ctxLogger telemetry.Logger, event cloudevents.Event, webhook *entities.Discord) any {
//	if event.Type() != events.EventTypeMessagePhoneReceived {
//		return event
//	}
//
//	if !strings.HasPrefix(webhook.URL, "https://discord.com/api/discordIntegrations/") {
//		return event
//	}
//
//	payload := new(events.MessagePhoneReceivedPayload)
//	err := event.DataAs(payload)
//	if err != nil {
//		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot unmarshal event [%s] with ID [%s] into [%T]", event.Type(), event.ID(), payload)))
//		return event
//	}
//
//	return map[string]string{
//		"avatar_url": "https://httpsms.com/avatar.png",
//		"username":   service.getFormattedContact(ctxLogger, payload.Contact),
//		"content":    payload.Content,
//	}
//}
//
//func (service *DiscordService) getFormattedContact(ctxLogger telemetry.Logger, contact string) string {
//	matched, err := regexp.MatchString("^\\+?[1-9]\\d{10,14}$", contact)
//	if err != nil {
//		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("error while matching contact [%s] with regex [%s]", contact, "^\\+?[1-9]\\d{10,14}$")))
//		return contact
//	}
//	if !matched {
//		return contact
//	}
//
//	number, err := phonenumbers.Parse(contact, phonenumbers.UNKNOWN_REGION)
//	if err != nil {
//		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot parse number [%s]", contact)))
//		return contact
//	}
//
//	return phonenumbers.Format(number, phonenumbers.INTERNATIONAL)
//}
//
//func (service *DiscordService) getAuthToken(webhook *entities.Discord) (string, error) {
//	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
//		Audience:  webhook.URL,
//		ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Unix(),
//		IssuedAt:  time.Now().UTC().Unix(),
//		Issuer:    "api.httpsms.com",
//		NotBefore: time.Now().UTC().Add(-10 * time.Minute).Unix(),
//		Subject:   string(webhook.UserID),
//	})
//	return token.SignedString([]byte(webhook.SigningKey))
//}
