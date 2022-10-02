package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"
	"github.com/google/uuid"
	"github.com/jinzhu/now"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormBillingUsageRepository is responsible for persisting entities.BillingUsage
type gormBillingUsageRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormBillingUsageRepository creates the GORM version of the BillingUsageRepository
func NewGormBillingUsageRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) BillingUsageRepository {
	return &gormBillingUsageRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormBillingUsageRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// RegisterSentMessage registers a message as sent
func (repository *gormBillingUsageRepository) RegisterSentMessage(ctx context.Context, timestamp time.Time, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	return crdbgorm.ExecuteTx(ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			result := tx.WithContext(ctx).
				Model(&entities.BillingUsage{}).
				Where("start_timestamp = ?", now.New(timestamp).BeginningOfMonth()).
				Where("user_id = ?", userID).
				UpdateColumn("sent_messages", gorm.Expr("sent_messages + ?", 1))

			if result.Error == nil && result.RowsAffected == 0 {
				return tx.Create(repository.createBillingUsage(userID, timestamp, 1, 0)).Error
			}
			return result.Error
		},
	)
}

// RegisterReceivedMessage registers a message as received
func (repository *gormBillingUsageRepository) RegisterReceivedMessage(ctx context.Context, timestamp time.Time, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	return crdbgorm.ExecuteTx(ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			result := tx.WithContext(ctx).
				Model(&entities.BillingUsage{}).
				Where("start_timestamp = ?", now.New(timestamp).BeginningOfMonth()).
				Where("user_id = ?", userID).
				UpdateColumn("received_messages", gorm.Expr("received_messages + ?", 1))

			if result.Error == nil && result.RowsAffected == 0 {
				return tx.Create(repository.createBillingUsage(userID, timestamp, 0, 1)).Error
			}
			return result.Error
		},
	)
}

// GetCurrent returns the current billing usage by entities.UserID
func (repository *gormBillingUsageRepository) GetCurrent(ctx context.Context, userID entities.UserID) (*entities.BillingUsage, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	timestamp := time.Now().UTC()
	usage := repository.createBillingUsage(userID, timestamp, 0, 0)

	err := crdbgorm.ExecuteTx(ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			result := tx.WithContext(ctx).
				Where("start_timestamp = ?", now.New(timestamp).BeginningOfMonth()).
				First(usage)

			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return tx.WithContext(ctx).Create(usage).Error
			}
			return result.Error
		},
	)
	if err != nil {
		return usage, stacktrace.Propagate(err, fmt.Sprintf("cannot load billing usage for user [%s]", userID))
	}

	return usage, err
}

// GetHistory returns past billing usage by entities.UserID
func (repository *gormBillingUsageRepository) GetHistory(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.BillingUsage, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	usages := new([]entities.BillingUsage)

	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("start_timestamp != ?", now.BeginningOfMonth()).
		Order("start_timestamp DESC").
		Limit(params.Limit).
		Offset(params.Skip).
		Find(&usages).
		Error
	if err != nil {
		msg := fmt.Sprintf("cannot fetch billing usage history for userID [%s] and params [%+#v]", userID, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return usages, err
}

func (repository *gormBillingUsageRepository) createBillingUsage(userID entities.UserID, timestamp time.Time, sent uint, received uint) *entities.BillingUsage {
	return &entities.BillingUsage{
		ID:               uuid.New(),
		UserID:           userID,
		SentMessages:     sent,
		ReceivedMessages: received,
		StartTimestamp:   now.New(timestamp).BeginningOfMonth(),
		EndTimestamp:     now.New(timestamp).EndOfMonth(),
	}
}
