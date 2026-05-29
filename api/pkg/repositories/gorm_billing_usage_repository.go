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
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormBillingUsageRepository is responsible for persisting entities.BillingUsage
type gormBillingUsageRepository struct {
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	db             *gorm.DB
	userRepository UserRepository
}

// NewGormBillingUsageRepository creates the GORM version of the BillingUsageRepository
func NewGormBillingUsageRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
	userRepository UserRepository,
) BillingUsageRepository {
	return &gormBillingUsageRepository{
		logger:         logger.WithService(fmt.Sprintf("%T", &gormBillingUsageRepository{})),
		tracer:         tracer,
		db:             db,
		userRepository: userRepository,
	}
}

// DeleteForUser deletes all billing usages for a user
func (repository *gormBillingUsageRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.BillingUsage{}).Error; err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.BillingUsage{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// RegisterSentMessage registers a message as sent
func (repository *gormBillingUsageRepository) RegisterSentMessage(ctx context.Context, timestamp time.Time, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	return crdbgorm.ExecuteTx(ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			result := tx.WithContext(ctx).
				Model(&entities.BillingUsage{}).
				Where("user_id = ?", userID).
				Where("start_timestamp <= ?", timestamp).
				Where("end_timestamp >= ?", timestamp).
				UpdateColumn("sent_messages", gorm.Expr("sent_messages + ?", 1))

			if result.Error == nil && result.RowsAffected == 0 {
				usage, err := repository.createBillingUsageForUser(ctx, userID, timestamp, 1, 0)
				if err != nil {
					return err
				}
				return tx.Create(usage).Error
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
				Where("user_id = ?", userID).
				Where("start_timestamp <= ?", timestamp).
				Where("end_timestamp >= ?", timestamp).
				UpdateColumn("received_messages", gorm.Expr("received_messages + ?", 1))

			if result.Error == nil && result.RowsAffected == 0 {
				usage, err := repository.createBillingUsageForUser(ctx, userID, timestamp, 0, 1)
				if err != nil {
					return err
				}
				return tx.Create(usage).Error
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

	var usage entities.BillingUsage
	err := crdbgorm.ExecuteTx(ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			result := tx.WithContext(ctx).
				Where("user_id = ?", userID).
				Where("start_timestamp <= ?", timestamp).
				Where("end_timestamp >= ?", timestamp).
				First(&usage)

			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				newUsage, createErr := repository.createBillingUsageForUser(ctx, userID, timestamp, 0, 0)
				if createErr != nil {
					return createErr
				}
				if err := tx.WithContext(ctx).Create(newUsage).Error; err != nil {
					return err
				}
				usage = *newUsage
				return nil
			}

			return result.Error
		},
	)
	if err != nil {
		return &usage, stacktrace.Propagate(err, fmt.Sprintf("cannot load billing usage for user [%s]", userID))
	}

	return &usage, nil
}

// GetHistory returns past billing usage by entities.UserID
func (repository *gormBillingUsageRepository) GetHistory(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.BillingUsage, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	timestamp := time.Now().UTC()
	usages := new([]entities.BillingUsage)

	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("end_timestamp < ?", timestamp).
		Order("start_timestamp DESC").
		Limit(params.Limit).
		Offset(params.Skip).
		Find(&usages).
		Error
	if err != nil {
		msg := fmt.Sprintf("cannot fetch billing usage history for userID [%s] and params [%+#v]", userID, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return usages, nil
}

// createBillingUsageForUser loads the user to determine anchor day and computes cycle boundaries.
func (repository *gormBillingUsageRepository) createBillingUsageForUser(ctx context.Context, userID entities.UserID, timestamp time.Time, sent uint, received uint) (*entities.BillingUsage, error) {
	user, err := repository.userRepository.Load(ctx, userID)
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot load user [%s] to compute billing cycle", userID))
	}

	start, end := entities.ComputeBillingCycle(timestamp, user.GetBillingAnchorDay())

	return &entities.BillingUsage{
		ID:               uuid.New(),
		UserID:           userID,
		SentMessages:     sent,
		ReceivedMessages: received,
		StartTimestamp:   start,
		EndTimestamp:     end,
	}, nil
}
