package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/repositories"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"
	"github.com/palantir/stacktrace"
)

// LemonsqueezyService is responsible for managing lemonsqueezy events
type LemonsqueezyService struct {
	service
	logger          telemetry.Logger
	tracer          telemetry.Tracer
	eventDispatcher *EventDispatcher
	userRepository  repositories.UserRepository
}

// NewLemonsqueezyService creates a new LemonsqueezyService
func NewLemonsqueezyService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.UserRepository,
	eventDispatcher *EventDispatcher,
) (s *LemonsqueezyService) {
	return &LemonsqueezyService{
		logger:          logger.WithService(fmt.Sprintf("%T", s)),
		tracer:          tracer,
		userRepository:  repository,
		eventDispatcher: eventDispatcher,
	}
}

// HandleSubscriptionCreatedEvent handles the subscription_created lemonsqueezy event
func (service *LemonsqueezyService) HandleSubscriptionCreatedEvent(ctx context.Context, source string, request *lemonsqueezy.WebhookRequestSubscription) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	payload := &events.UserSubscriptionCreatedPayload{
		UserID:                entities.UserID(request.Meta.CustomData["user_id"].(string)),
		SubscriptionCreatedAt: request.Data.Attributes.CreatedAt,
		SubscriptionID:        request.Data.ID,
		SubscriptionName:      service.subscriptionName(request.Data.Attributes.VariantName),
		SubscriptionRenewsAt:  request.Data.Attributes.RenewsAt,
		SubscriptionStatus:    request.Data.Attributes.Status,
	}

	event, err := service.createEvent(events.UserSubscriptionCreated, source, payload)
	if err != nil {
		msg := fmt.Sprintf("cannot create [%s] event for user [%s]", events.UserSubscriptionCreated, payload.UserID)
		return stacktrace.Propagate(err, msg)
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch [%s] event for user [%s]", events.UserSubscriptionCreated, payload.UserID)
		return stacktrace.Propagate(err, msg)
	}
	ctxLogger.Info(fmt.Sprintf("[%s] subscription [%s] created for user [%s]", payload.SubscriptionName, payload.SubscriptionID, payload.UserID))
	return nil
}

// HandleSubscriptionCanceledEvent handles the subscription_cancelled lemonsqueezy event
func (service *LemonsqueezyService) HandleSubscriptionCanceledEvent(ctx context.Context, source string, request *lemonsqueezy.WebhookRequestSubscription) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.userRepository.LoadBySubscriptionID(ctx, request.Data.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot load user with subscription ID [%s]", request.Data.ID)
		return stacktrace.Propagate(err, msg)
	}

	payload := &events.UserSubscriptionCancelledPayload{
		UserID:                  user.ID,
		SubscriptionCancelledAt: request.Data.Attributes.CreatedAt,
		SubscriptionID:          request.Data.ID,
		SubscriptionName:        service.subscriptionName(request.Data.Attributes.VariantName),
		SubscriptionEndsAt:      *request.Data.Attributes.EndsAt,
		SubscriptionStatus:      request.Data.Attributes.Status,
	}

	event, err := service.createEvent(events.UserSubscriptionCancelled, source, payload)
	if err != nil {
		msg := fmt.Sprintf("cannot created [%s] event for user [%s]", events.UserSubscriptionCancelled, payload.UserID)
		return stacktrace.Propagate(err, msg)
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch [%s] event for user [%s]", events.UserSubscriptionCancelled, payload.UserID)
		return stacktrace.Propagate(err, msg)
	}
	ctxLogger.Info(fmt.Sprintf("[%s] subscription [%s] cancelled for user [%s]", payload.SubscriptionName, payload.SubscriptionID, payload.UserID))
	return nil
}

// HandleSubscriptionUpdatedEvent handles the subscription_cancelled lemonsqueezy event
func (service *LemonsqueezyService) HandleSubscriptionUpdatedEvent(ctx context.Context, source string, request *lemonsqueezy.WebhookRequestSubscription) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.userRepository.LoadBySubscriptionID(ctx, request.Data.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot load user with subscription ID [%s]", request.Data.ID)
		return stacktrace.Propagate(err, msg)
	}

	payload := &events.UserSubscriptionUpdatedPayload{
		UserID:                user.ID,
		SubscriptionUpdatedAt: request.Data.Attributes.UpdatedAt,
		SubscriptionID:        request.Data.ID,
		SubscriptionName:      service.subscriptionName(request.Data.Attributes.VariantName),
		SubscriptionEndsAt:    *request.Data.Attributes.EndsAt,
		SubscriptionRenewsAt:  request.Data.Attributes.RenewsAt,
		SubscriptionStatus:    request.Data.Attributes.Status,
	}

	event, err := service.createEvent(events.UserSubscriptionUpdated, source, payload)
	if err != nil {
		msg := fmt.Sprintf("cannot created [%s] event for user [%s]", events.UserSubscriptionUpdated, payload.UserID)
		return stacktrace.Propagate(err, msg)
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch [%s] event for user [%s]", event.Type(), payload.UserID)
		return stacktrace.Propagate(err, msg)
	}
	ctxLogger.Info(fmt.Sprintf("[%s] subscription [%s] updated for user [%s]", payload.SubscriptionName, payload.SubscriptionID, payload.UserID))
	return nil
}

// HandleSubscriptionExpiredEvent handles the subscription_expired lemonsqueezy event
func (service *LemonsqueezyService) HandleSubscriptionExpiredEvent(ctx context.Context, source string, request *lemonsqueezy.WebhookRequestSubscription) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.userRepository.LoadBySubscriptionID(ctx, request.Data.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot load user with subscription ID [%s]", request.Data.ID)
		return stacktrace.Propagate(err, msg)
	}

	payload := &events.UserSubscriptionExpiredPayload{
		UserID:                user.ID,
		SubscriptionExpiredAt: time.Now().UTC(),
		SubscriptionID:        request.Data.ID,
		IsCancelled:           request.Data.Attributes.Cancelled,
		SubscriptionName:      service.subscriptionName(request.Data.Attributes.VariantName),
		SubscriptionEndsAt:    *request.Data.Attributes.EndsAt,
		SubscriptionStatus:    request.Data.Attributes.Status,
	}

	event, err := service.createEvent(events.UserSubscriptionExpired, source, payload)
	if err != nil {
		msg := fmt.Sprintf("cannot created [%s] event for user [%s]", events.UserSubscriptionExpired, payload.UserID)
		return stacktrace.Propagate(err, msg)
	}

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch [%s] event for user [%s]", event.Type(), payload.UserID)
		return stacktrace.Propagate(err, msg)
	}
	ctxLogger.Info(fmt.Sprintf("[%s] subscription [%s] expired for user [%s]", payload.SubscriptionName, payload.SubscriptionID, payload.UserID))
	return nil
}

func (service *LemonsqueezyService) subscriptionName(variant string) entities.SubscriptionName {
	if strings.Contains(strings.ToLower(variant), "pro") {
		if strings.Contains(strings.ToLower(variant), "monthly") {
			return entities.SubscriptionNameProMonthly
		}
		return entities.SubscriptionNameProYearly
	}

	if strings.Contains(strings.ToLower(variant), "ultra") {
		if strings.Contains(strings.ToLower(variant), "monthly") {
			return entities.SubscriptionNameUltraMonthly
		}
		return entities.SubscriptionNameUltraYearly
	}

	if strings.Contains(strings.ToLower(variant), "20k") {
		if strings.Contains(strings.ToLower(variant), "monthly") {
			return entities.SubscriptionName20KMonthly
		}
		return entities.SubscriptionName20KYearly
	}

	if strings.Contains(strings.ToLower(variant), "100k") {
		if strings.Contains(strings.ToLower(variant), "monthly") {
			return entities.SubscriptionName100KMonthly
		}
		return entities.SubscriptionName100KYearly
	}

	return entities.SubscriptionNameFree
}
