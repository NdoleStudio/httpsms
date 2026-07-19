package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"
	"github.com/google/uuid"
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

// DeleteForUser deletes all billing usages for a user
func (repository *gormBillingUsageRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.BillingUsage{}).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete all [%T] for user with ID [%s]", &entities.BillingUsage{}, userID))
	}

	return nil
}

// RegisterSentMessage registers a message as sent
func (repository *gormBillingUsageRepository) RegisterSentMessage(ctx context.Context, timestamp time.Time, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	return crdbgorm.ExecuteTx(
		ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			result := tx.WithContext(ctx).
				Model(&entities.BillingUsage{}).
				Where("user_id = ?", userID).
				Where("start_timestamp <= ?", timestamp).
				Where("end_timestamp >= ?", timestamp).
				UpdateColumn("sent_messages", gorm.Expr("sent_messages + ?", 1))

			if result.Error == nil && result.RowsAffected == 0 {
				usage, err := repository.createBillingUsageForUser(ctx, tx, userID, timestamp, 1, 0)
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

	return crdbgorm.ExecuteTx(
		ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			result := tx.WithContext(ctx).
				Model(&entities.BillingUsage{}).
				Where("user_id = ?", userID).
				Where("start_timestamp <= ?", timestamp).
				Where("end_timestamp >= ?", timestamp).
				UpdateColumn("received_messages", gorm.Expr("received_messages + ?", 1))

			if result.Error == nil && result.RowsAffected == 0 {
				usage, err := repository.createBillingUsageForUser(ctx, tx, userID, timestamp, 0, 1)
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
	err := crdbgorm.ExecuteTx(
		ctx, repository.db, nil,
		func(tx *gorm.DB) error {
			result := tx.WithContext(ctx).
				Where("user_id = ?", userID).
				Where("start_timestamp <= ?", timestamp).
				Where("end_timestamp >= ?", timestamp).
				First(&usage)

			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				newUsage, createErr := repository.createBillingUsageForUser(ctx, tx, userID, timestamp, 0, 0)
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
		return &usage, stacktrace.Propagatef(err, "cannot load billing usage for user [%s]", userID)
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
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch billing usage history for userID [%s] and params [%+#v]", userID, params))
	}

	return usages, nil
}

// createBillingUsageForUser loads the user to determine anchor day and computes cycle boundaries.
// It accepts a tx to ensure the user read is part of the same transaction snapshot.
func (repository *gormBillingUsageRepository) createBillingUsageForUser(ctx context.Context, tx *gorm.DB, userID entities.UserID, timestamp time.Time, sent uint, received uint) (*entities.BillingUsage, error) {
	user := new(entities.User)
	if err := tx.WithContext(ctx).First(user, userID).Error; err != nil {
		return nil, stacktrace.Propagatef(err, "cannot load user [%s] to compute billing cycle", userID)
	}

	start, end := computeBillingCycle(timestamp, user.GetBillingAnchorDay())

	return &entities.BillingUsage{
		ID:               uuid.New(),
		UserID:           userID,
		SentMessages:     sent,
		ReceivedMessages: received,
		StartTimestamp:   start,
		EndTimestamp:     end,
	}, nil
}

// computeBillingCycle returns the start and end timestamps of the billing cycle
// that contains `now`, given the user's anchor day (1–31). The anchor day is
// dynamically clamped to the number of days in the relevant month.
func computeBillingCycle(now time.Time, anchorDay int) (start, end time.Time) {
	clampedDay := min(anchorDay, daysInMonth(now.Year(), now.Month()))

	if now.Day() >= clampedDay {
		start = time.Date(now.Year(), now.Month(), clampedDay, 0, 0, 0, 0, time.UTC)
	} else {
		prev := now.AddDate(0, -1, 0)
		prevClamped := min(anchorDay, daysInMonth(prev.Year(), prev.Month()))
		start = time.Date(prev.Year(), prev.Month(), prevClamped, 0, 0, 0, 0, time.UTC)
	}

	nextMonth := start.Month() + 1
	nextYear := start.Year()
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}

	nextClamped := min(anchorDay, daysInMonth(nextYear, nextMonth))
	nextCycleStart := time.Date(nextYear, nextMonth, nextClamped, 0, 0, 0, 0, time.UTC)

	end = nextCycleStart.Add(-time.Second)

	return start, end
}

// daysInMonth returns the number of days in the given month/year.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
