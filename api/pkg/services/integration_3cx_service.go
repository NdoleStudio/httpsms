package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"

	"github.com/gofiber/fiber/v2"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/carlmjohnson/requests"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"
)

// Integration3CXService is responsible for handling webhooks
type Integration3CXService struct {
	service
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	client     *http.Client
	repository repositories.Integration3CxRepository
}

// NewIntegration3CXService creates a new Integration3CXService
func NewIntegration3CXService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *http.Client,
	repository repositories.Integration3CxRepository,
) (s *Integration3CXService) {
	return &Integration3CXService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		client:     client,
		repository: repository,
	}
}

// Send an event to a 3CX webhook
func (service *Integration3CXService) Send(ctx context.Context, userID entities.UserID, event cloudevents.Event) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	webhooks, err := service.repository.Load(ctx, userID)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		ctxLogger.Info(fmt.Sprintf("user [%s] has no [3cx] integration to event [%s]", userID, event.Type()))
		return nil
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load [3cx] integration for user [%s] and event [%s]", userID, event.Type())
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	service.sendNotification(ctx, event, webhooks)
	return nil
}

func (service *Integration3CXService) sendNotification(ctx context.Context, event cloudevents.Event, integration *entities.Integration3CX) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload, err := service.getPayload(event)
	if err != nil {
		msg := fmt.Sprintf("cannot generate payload from [%s] event with ID [%s] for user [%s]", event.Type(), event.ID(), integration.UserID)
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
	}

	var response string
	err = requests.URL(integration.WebhookURL).
		Client(service.client).
		BodyJSON(payload).
		ToString(&response).
		Fetch(ctx)
	if err != nil {
		msg := fmt.Sprintf("cannot send [%s] event to [3cx] webhook [%s] for user [%s]", event.Type(), integration.WebhookURL, integration.UserID)
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
	}

	ctxLogger.Info(fmt.Sprintf("sent [3cx] webhook to url [%s] for event [%s] with ID [%s] and response [%s]", integration.WebhookURL, event.Type(), event.ID(), response))
}

func (service *Integration3CXService) getPayload(event cloudevents.Event) (fiber.Map, error) {
	switch event.Type() {
	case events.EventTypeMessagePhoneDelivered:
		return service.getEventDeliveredPayload(event)
	case events.EventTypeMessagePhoneReceived:
		return service.getMessageReceivedPayload(event)
	case events.EventTypeMessagePhoneSent:
		return service.getMessageSentPayload(event)
	case events.EventTypeMessageSendFailed:
		return service.getMessageSendFailedPayload(event)
	default:
		return nil, stacktrace.NewError(fmt.Sprintf("cannot generate [3cx] payload for event [%s] with ID [%s]", event.Type(), event.ID()))
	}
}

func (service *Integration3CXService) getEventDeliveredPayload(event cloudevents.Event) (fiber.Map, error) {
	payload := new(events.MessagePhoneDeliveredPayload)
	if err := event.DataAs(payload); err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot unmarshal event [%s] with ID [%s] into [%T]", event.Type(), event.ID(), payload))
	}

	return fiber.Map{
		"data": fiber.Map{
			"event_type":  "message.finalized",
			"id":          event.ID(),
			"occurred_at": event.Time(),
			"payload": fiber.Map{
				"completed_at": payload.Timestamp,
				"from": fiber.Map{
					"phone_number": payload.Owner,
				},
				"id": payload.ID,
				"to": []fiber.Map{
					{
						"status":       "delivered",
						"phone_number": payload.Contact,
					},
				},
				"type": "SMS",
			},
			"record_type": "event",
		},
	}, nil
}

func (service *Integration3CXService) getMessageSentPayload(event cloudevents.Event) (fiber.Map, error) {
	payload := new(events.MessagePhoneSentPayload)
	if err := event.DataAs(payload); err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot unmarshal event [%s] with ID [%s] into [%T]", event.Type(), event.ID(), payload))
	}

	return fiber.Map{
		"data": fiber.Map{
			"event_type":  "message.sent",
			"id":          event.ID(),
			"occurred_at": event.Time(),
			"payload": fiber.Map{
				"completed_at": payload.Timestamp,
				"from": fiber.Map{
					"phone_number": payload.Owner,
				},
				"id": payload.ID,
				"to": []fiber.Map{
					{
						"status":       "sent",
						"phone_number": payload.Contact,
					},
				},
				"type": "SMS",
			},
			"record_type": "event",
		},
	}, nil
}

func (service *Integration3CXService) getMessageSendFailedPayload(event cloudevents.Event) (fiber.Map, error) {
	payload := new(events.MessageSendFailedPayload)
	if err := event.DataAs(payload); err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot unmarshal event [%s] with ID [%s] into [%T]", event.Type(), event.ID(), payload))
	}

	return fiber.Map{
		"data": fiber.Map{
			"event_type":  "message.sent",
			"id":          event.ID(),
			"occurred_at": event.Time(),
			"payload": fiber.Map{
				"completed_at": payload.Timestamp,
				"from": fiber.Map{
					"phone_number": payload.Owner,
				},
				"id": payload.ID,
				"to": []fiber.Map{
					{
						"status":       "sending_failed",
						"phone_number": payload.Contact,
					},
				},
				"type": "SMS",
			},
			"record_type": "event",
		},
	}, nil
}

func (service *Integration3CXService) getMessageReceivedPayload(event cloudevents.Event) (fiber.Map, error) {
	payload := new(events.MessagePhoneReceivedPayload)
	if err := event.DataAs(payload); err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot unmarshal event [%s] with ID [%s] into [%T]", event.Type(), event.ID(), payload))
	}

	return fiber.Map{
		"data": fiber.Map{
			"event_type": "message.received",
			"id":         event.ID(),
			"payload": fiber.Map{
				"from": fiber.Map{
					"phone_number": payload.Contact,
				},
				"id":          payload.MessageID,
				"received_at": payload.Timestamp,
				"text":        payload.Content,
				"to": []fiber.Map{
					{
						"phone_number": payload.Owner,
					},
				},
				"type": "SMS",
			},
			"record_type": "event",
		},
	}, nil
}
