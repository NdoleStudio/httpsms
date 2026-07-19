package services

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/cache"
	"github.com/NdoleStudio/httpsms/pkg/emails"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
)

// BillingService is responsible for tracking usages and billing users
type BillingService struct {
	service
	logger                 telemetry.Logger
	tracer                 telemetry.Tracer
	cache                  cache.Cache
	emailFactory           emails.UserEmailFactory
	mailer                 emails.Mailer
	userRepository         repositories.UserRepository
	billingUsageRepository repositories.BillingUsageRepository
}

// NewBillingService creates a new BillingService
func NewBillingService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	cache cache.Cache,
	mailer emails.Mailer,
	emailFactory emails.UserEmailFactory,
	usageRepository repositories.BillingUsageRepository,
	userRepository repositories.UserRepository,
) (s *BillingService) {
	return &BillingService{
		logger:                 logger.WithService(fmt.Sprintf("%T", s)),
		tracer:                 tracer,
		cache:                  cache,
		emailFactory:           emailFactory,
		mailer:                 mailer,
		userRepository:         userRepository,
		billingUsageRepository: usageRepository,
	}
}

// IsEntitledWithCount checks if a user can send or receive and SMS message
func (service *BillingService) IsEntitledWithCount(ctx context.Context, userID entities.UserID, count uint) *string {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.userRepository.Load(ctx, userID)
	if err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load user with ID [%s], entitlement successful", userID)))
		return nil
	}

	usage, err := service.billingUsageRepository.GetCurrent(ctx, userID)
	if err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load billing usage for user with ID [%s], entitlement successful", userID)))
		return nil
	}

	if !usage.IsEntitled(count, user.SubscriptionName.Limit()) {
		return service.handleLimitExceeded(ctx, user, usage)
	}

	return nil
}

// IsEntitled checks if a user can send or receive and SMS message
func (service *BillingService) IsEntitled(ctx context.Context, userID entities.UserID) *string {
	return service.IsEntitledWithCount(ctx, userID, 1)
}

func (service *BillingService) handleLimitExceeded(ctx context.Context, user *entities.User, usage *entities.BillingUsage) *string {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	service.sendLimitExceededEmail(ctx, user, usage)

	message := fmt.Sprintf(
		"You have exceeded your limit of [%d] messages on your [%s] plan. Upgrade to send more messages on https://httpsms.com/billing",
		user.SubscriptionName.Limit(),
		user.SubscriptionName,
	)
	return &message
}

func (service *BillingService) sendLimitExceededEmail(ctx context.Context, user *entities.User, usage *entities.BillingUsage) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	key := fmt.Sprintf("user.limit.exceeded.%s", user.ID)
	if _, err := service.cache.Get(ctx, key); err == nil {
		return
	}

	email, err := service.emailFactory.UsageLimitExceeded(user, usage)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot create usage limit email for user [%s]", user.ID))
		return
	}

	if err = service.mailer.Send(ctx, email); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "canot send usage limit exceeded notification to user [%s]", user.ID))
		return
	}

	ctxLogger.Info(fmt.Sprintf("usage limit exceeded email sent to user [%s]", user.ID))
	if err = service.cache.Set(ctx, key, "", time.Hour*12); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot set item in redis with key [%s]", key))
	}
}

// GetCurrentUsage gets the current billing usage for a user
func (service *BillingService) GetCurrentUsage(ctx context.Context, userID entities.UserID) (*entities.BillingUsage, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	return service.billingUsageRepository.GetCurrent(ctx, userID)
}

// GetUsageHistory gets the billing usage history for a user
func (service *BillingService) GetUsageHistory(ctx context.Context, userID entities.UserID, params repositories.IndexParams) (*[]entities.BillingUsage, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	return service.billingUsageRepository.GetHistory(ctx, userID, params)
}

// RegisterSentMessage records the billing usage for a sent message
func (service *BillingService) RegisterSentMessage(ctx context.Context, messageID uuid.UUID, timestamp time.Time, userID entities.UserID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if err := service.billingUsageRepository.RegisterSentMessage(ctx, timestamp, userID); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "could not register [sent] message with ID [%s] for user with ID [%s]", messageID, userID))
	}

	ctxLogger.Info(fmt.Sprintf("registered [sent] message with ID [%s] for user [%s]", messageID, userID))
	service.sendUsageAlert(ctx, userID)
	return nil
}

// RegisterReceivedMessage records the billing usage for a received message
func (service *BillingService) RegisterReceivedMessage(ctx context.Context, messageID uuid.UUID, timestamp time.Time, userID entities.UserID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if err := service.billingUsageRepository.RegisterReceivedMessage(ctx, timestamp, userID); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "could not register [received] message with ID [%s] for user with ID [%s]", messageID, userID))
	}

	ctxLogger.Info(fmt.Sprintf("registered [received] message with ID [%s] for user [%s]", messageID, userID))
	service.sendUsageAlert(ctx, userID)
	return nil
}

// DeleteAllForUser deletes all entities.BillingUsage for an entities.UserID.
func (service *BillingService) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.billingUsageRepository.DeleteAllForUser(ctx, userID); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "could not delete [entities.BillingUsage] for user with ID [%s]", userID))
	}

	ctxLogger.Info(fmt.Sprintf("deleted all [entities.BillingUsage] for user with ID [%s]", userID))
	return nil
}

func (service *BillingService) sendUsageAlert(ctx context.Context, userID entities.UserID) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.userRepository.Load(ctx, userID)
	if err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load user with ID [%s]", userID)))
		return
	}

	billingUsage, err := service.billingUsageRepository.GetCurrent(ctx, userID)
	if err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load billing usage for user with ID [%s]", userID)))
		return
	}

	if !service.shouldSendAlert(user, billingUsage) {
		return
	}

	email, err := service.emailFactory.UsageLimitAlert(user, billingUsage)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot create usage alert email for user [%s]", user.ID))
		return
	}

	if err = service.mailer.Send(ctx, email); err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "canot send usage alert notification to user [%s]", user.ID))
	}

	ctxLogger.Info(fmt.Sprintf("usage alert email sent to user [%s]", user.ID))
}

func (service *BillingService) shouldSendAlert(user *entities.User, usage *entities.BillingUsage) bool {
	if user.IsOnFreePlan() && (usage.TotalMessages() == 160 || usage.TotalMessages() == 180 || usage.TotalMessages() == 190) {
		return true
	}

	if user.IsOnProPlan() && (usage.TotalMessages() == 4000 || usage.TotalMessages() == 4500 || usage.TotalMessages() == 4750) {
		return true
	}

	if user.IsOnUltraPlan() && (usage.TotalMessages() == 8000 || usage.TotalMessages() == 9000 || usage.TotalMessages() == 9500) {
		return true
	}

	if user.IsOn20kPlan() && (usage.TotalMessages() == 16000 || usage.TotalMessages() == 18000 || usage.TotalMessages() == 19000) {
		return true
	}

	return false
}
