package services

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// BillingService is responsible for tracking usages and billing users
type BillingService struct {
	service
	logger          telemetry.Logger
	tracer          telemetry.Tracer
	usageRepository repositories.BillingUsageRepository
}

// NewBillingService creates a new BillingService
func NewBillingService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	usageRepository repositories.BillingUsageRepository,
) (s *BillingService) {
	return &BillingService{
		logger:          logger.WithService(fmt.Sprintf("%T", s)),
		tracer:          tracer,
		usageRepository: usageRepository,
	}
}

// GetCurrentUsage gets the current billing usage for a user
func (service *BillingService) GetCurrentUsage(ctx context.Context, userID entities.UserID) (*entities.BillingUsage, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	return service.usageRepository.GetCurrent(ctx, userID)
}

// GetUsageHistory gets the billing usage history for a user
func (service *BillingService) GetUsageHistory(ctx context.Context, userID entities.UserID, params repositories.IndexParams) (*[]entities.BillingUsage, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	return service.usageRepository.GetHistory(ctx, userID, params)
}

// RegisterSentMessage records the billing usage for a sent message
func (service *BillingService) RegisterSentMessage(ctx context.Context, messageID uuid.UUID, timestamp time.Time, userID entities.UserID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if err := service.usageRepository.RegisterSentMessage(ctx, timestamp, userID); err != nil {
		msg := fmt.Sprintf("could not register [sent] message with ID [%s] for user with ID [%s]", messageID, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("registered [sent] message with ID [%s] for user [%s]", messageID, userID))
	return nil
}

// RegisterReceivedMessage records the billing usage for a received message
func (service *BillingService) RegisterReceivedMessage(ctx context.Context, messageID uuid.UUID, timestamp time.Time, userID entities.UserID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if err := service.usageRepository.RegisterReceivedMessage(ctx, timestamp, userID); err != nil {
		msg := fmt.Sprintf("could not register [received] message with ID [%s] for user with ID [%s]", messageID, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("registered [received] message with ID [%s] for user [%s]", messageID, userID))
	return nil
}
