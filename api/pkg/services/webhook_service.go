package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"

	"github.com/gofiber/fiber/v2"

	"github.com/NdoleStudio/httpsms/pkg/events"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/palantir/stacktrace"
)

// WebhookService is responsible for handling webhooks
type WebhookService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	client     *http.Client
	repository repositories.WebhookRepository
	dispatcher *EventDispatcher
}

// NewWebhookService creates a new WebhookService
func NewWebhookService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *http.Client,
	repository repositories.WebhookRepository,
	dispatcher *EventDispatcher,
) (s *WebhookService) {
	return &WebhookService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		client:     client,
		dispatcher: dispatcher,
		repository: repository,
	}
}

// DeleteAllForUser deletes all entities.Webhook for an entities.UserID.
func (service *WebhookService) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.repository.DeleteAllForUser(ctx, userID); err != nil {
		msg := fmt.Sprintf("could not delete all [entities.Webhook] for user with ID [%s]", userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted all [entities.Webhook] for user with ID [%s]", userID))
	return nil
}

// Index fetches the entities.Webhook for an entities.UserID
func (service *WebhookService) Index(ctx context.Context, userID entities.UserID, params repositories.IndexParams) ([]*entities.Webhook, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	webhooks, err := service.repository.Index(ctx, userID, params)
	if err != nil {
		msg := fmt.Sprintf("could not fetch webhooks with params [%+#v]", params)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] webhooks with prams [%+#v]", len(webhooks), params))
	return webhooks, nil
}

// Delete an entities.Webhook
func (service *WebhookService) Delete(ctx context.Context, userID entities.UserID, webhookID uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if _, err := service.repository.Load(ctx, userID, webhookID); err != nil {
		msg := fmt.Sprintf("cannot load webhook with userID [%s] and phoneID [%s]", userID, webhookID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	if err := service.repository.Delete(ctx, userID, webhookID); err != nil {
		msg := fmt.Sprintf("cannot delete webhook with id [%s] and user id [%s]", webhookID, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted webhook with id [%s] and user id [%s]", webhookID, userID))
	return nil
}

// WebhookStoreParams are parameters for creating a new entities.Webhook
type WebhookStoreParams struct {
	UserID       entities.UserID
	SigningKey   string
	URL          string
	PhoneNumbers pq.StringArray
	Events       pq.StringArray
}

// Store a new entities.Webhook
func (service *WebhookService) Store(ctx context.Context, params *WebhookStoreParams) (*entities.Webhook, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	webhook := &entities.Webhook{
		ID:           uuid.New(),
		UserID:       params.UserID,
		URL:          params.URL,
		PhoneNumbers: params.PhoneNumbers,
		SigningKey:   params.SigningKey,
		Events:       params.Events,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := service.repository.Save(ctx, webhook); err != nil {
		msg := fmt.Sprintf("cannot save webhook with id [%s]", webhook.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("webhook saved with id [%s] for user [%s] in the [%T]", webhook.ID, webhook.UserID, service.repository))
	return webhook, nil
}

// WebhookUpdateParams are parameters for updating an entities.Webhook
type WebhookUpdateParams struct {
	UserID       entities.UserID
	SigningKey   string
	URL          string
	Events       pq.StringArray
	PhoneNumbers pq.StringArray
	WebhookID    uuid.UUID
}

// Update an entities.Webhook
func (service *WebhookService) Update(ctx context.Context, params *WebhookUpdateParams) (*entities.Webhook, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	webhook, err := service.repository.Load(ctx, params.UserID, params.WebhookID)
	if err != nil {
		msg := fmt.Sprintf("cannot load webhook with userID [%s] and phoneID [%s]", params.UserID, params.WebhookID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	webhook.URL = params.URL
	webhook.SigningKey = params.SigningKey
	webhook.Events = params.Events
	webhook.PhoneNumbers = params.PhoneNumbers

	if err = service.repository.Save(ctx, webhook); err != nil {
		msg := fmt.Sprintf("cannot save webhook with id [%s] after update", webhook.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("webhook updated with id [%s] in the [%T]", webhook.ID, service.repository))
	return webhook, nil
}

// Send an event to a subscribed webhook
func (service *WebhookService) Send(ctx context.Context, userID entities.UserID, event cloudevents.Event, phoneNumber string) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	webhooks, err := service.repository.LoadByEvent(ctx, userID, event.Type(), phoneNumber)
	if err != nil {
		msg := fmt.Sprintf("cannot load webhooks for userID [%s] and event [%s]", userID, event.Type())
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	if len(webhooks) == 0 {
		ctxLogger.Info(fmt.Sprintf("user [%s] has no webhook subscription to event [%s]", userID, event.Type()))
		return nil
	}

	var wg sync.WaitGroup
	for _, webhook := range webhooks {
		wg.Add(1)
		go func(webhook *entities.Webhook) {
			defer wg.Done()
			service.sendNotification(ctx, event, phoneNumber, webhook)
		}(webhook)
	}
	wg.Wait()

	return nil
}

func (service *WebhookService) sendNotification(ctx context.Context, event cloudevents.Event, owner string, webhook *entities.Webhook) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	attempts := 0
	err := retry.Do(func() error {
		attempts++

		requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		request, err := service.createRequest(requestCtx, event, webhook)
		if err != nil {
			msg := fmt.Sprintf("cannot create [%s] event to webhook [%s] for user [%s] after [%d] attempts", event.Type(), webhook.URL, webhook.UserID, attempts)
			return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}

		response, err := service.client.Do(request)
		if err != nil {
			ctxLogger.Warn(stacktrace.Propagate(err, fmt.Sprintf("cannot send [%s] event to webhook [%s] for user [%s] after [%d] attempts", event.Type(), webhook.URL, webhook.UserID, attempts)))
			if attempts == 1 {
				return err
			}
			service.handleWebhookSendFailed(ctx, event, webhook, owner, err, nil)
			return nil
		}

		defer func() {
			err = response.Body.Close()
			if err != nil {
				ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot close response body for [%s] event with ID [%s] after [%d] attempts", event.Type(), event.ID(), attempts)))
			}
		}()

		if response.StatusCode >= 400 {
			ctxLogger.Info(fmt.Sprintf("cannot send [%s] event to webhook [%s] for user [%s] with response code [%d]", event.Type(), webhook.URL, webhook.UserID, response.StatusCode))
			if attempts == 1 {
				return stacktrace.NewError(http.StatusText(response.StatusCode))
			}
			service.handleWebhookSendFailed(ctx, event, webhook, owner, stacktrace.NewError(http.StatusText(response.StatusCode)), response)
			return nil
		}

		ctxLogger.Info(fmt.Sprintf("sent webhook to url [%s] for event [%s] with ID [%s] and response code [%d]", webhook.URL, event.Type(), event.ID(), response.StatusCode))
		return nil
	}, retry.Attempts(2))
	if err != nil {
		msg := fmt.Sprintf("cannot handle [%s] event to webhook [%s] for user [%s] after [%d] attempts", event.Type(), webhook.URL, webhook.UserID, attempts)
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
	}
}

func (service *WebhookService) createRequest(ctx context.Context, event cloudevents.Event, webhook *entities.Webhook) (*http.Request, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	payload, err := json.Marshal(service.getPayload(ctxLogger, event, webhook))
	if err != nil {
		msg := fmt.Sprintf("cannot marshal payload for user [%s] and webhook [%s] for event [%s]", webhook.UserID, webhook.ID, event.ID())
		return nil, stacktrace.Propagate(err, msg)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payload))
	if err != nil {
		msg := fmt.Sprintf("cannot create request for user [%s] and webhook [%s] for event [%s]", webhook.UserID, webhook.ID, event.ID())
		return nil, stacktrace.Propagate(err, msg)
	}

	request.Header.Add("X-Event-Type", event.Type())
	request.Header.Set("Content-Type", "application/json")

	if strings.TrimSpace(webhook.SigningKey) != "" {
		token, err := service.getAuthToken(webhook)
		if err != nil {
			msg := fmt.Sprintf("cannot generate auth token for user [%s] and webhook [%s]", webhook.UserID, webhook.ID)
			return nil, stacktrace.Propagate(err, msg)
		}
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	return request, nil
}

func (service *WebhookService) getPayload(ctxLogger telemetry.Logger, event cloudevents.Event, webhook *entities.Webhook) any {
	if event.Type() != events.EventTypeMessagePhoneReceived {
		return event
	}

	if !strings.HasPrefix(webhook.URL, "https://discord.com/api/webhooks/") {
		return event
	}

	payload := new(events.MessagePhoneReceivedPayload)

	err := event.DataAs(payload)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot unmarshal event [%s] with ID [%s] into [%T]", event.Type(), event.ID(), payload)))
		return event
	}

	return map[string]any{
		"avatar_url": "https://httpsms.com/avatar.png",
		"username":   "httpsms.com",
		"content":    "✉ new message received",
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

func (service *WebhookService) getAuthToken(webhook *entities.Webhook) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Audience:  webhook.URL,
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Unix(),
		IssuedAt:  time.Now().UTC().Unix(),
		Issuer:    "api.httpsms.com",
		NotBefore: time.Now().UTC().Add(-10 * time.Minute).Unix(),
		Subject:   string(webhook.UserID),
	})
	return token.SignedString([]byte(webhook.SigningKey))
}

func (service *WebhookService) handleWebhookSendFailed(ctx context.Context, event cloudevents.Event, webhook *entities.Webhook, owner string, err error, response *http.Response) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	payload := &events.WebhookSendFailedPayload{
		WebhookID:              webhook.ID,
		WebhookURL:             webhook.URL,
		UserID:                 webhook.UserID,
		EventID:                event.ID(),
		Owner:                  owner,
		EventType:              event.Type(),
		EventPayload:           string(event.Data()),
		HTTPResponseStatusCode: nil,
		ErrorMessage:           err.Error(),
	}

	if errors.Is(err, context.DeadlineExceeded) {
		payload.ErrorMessage = "TIMOUT after 10 seconds"
	}

	if response != nil {
		payload.HTTPResponseStatusCode = &response.StatusCode
		payload.ErrorMessage = http.StatusText(response.StatusCode)

		body, err := io.ReadAll(response.Body)
		if err == nil && len(body) > 0 {
			payload.ErrorMessage = string(body)
		}
	}

	event, err = service.createEvent(events.EventTypeWebhookSendFailed, event.Source(), payload)
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for user with id [%s]", events.EventTypeWebhookSendFailed, payload.UserID)
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for user with id [%s]", event.Type(), payload.UserID)
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return
	}

	ctxLogger.Info(fmt.Sprintf("dispatched [%s] event with ID [%s] for user with id [%s]", event.Type(), event.ID(), payload.UserID))
}
