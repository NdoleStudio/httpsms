package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"
	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormPhoneNotificationRepository is responsible for persisting entities.PhoneNotification
type gormPhoneNotificationRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormPhoneNotificationRepository creates the GORM version of the PhoneNotificationRepository
func NewGormPhoneNotificationRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) PhoneNotificationRepository {
	return &gormPhoneNotificationRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormHeartbeatRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *gormPhoneNotificationRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.PhoneNotification{}).Error; err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.PhoneNotification{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// UpdateStatus of an entities.PhoneNotification
func (repository *gormPhoneNotificationRepository) UpdateStatus(ctx context.Context, notificationID uuid.UUID, status entities.PhoneNotificationStatus) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.
		WithContext(ctx).
		Model(&entities.PhoneNotification{ID: notificationID}).
		Update("status", status).
		Error
	if err != nil {
		msg := fmt.Sprintf("cannot update notification [%s] with status [%s]", notificationID, status)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Schedule a notification to be sent in the future
func (repository *gormPhoneNotificationRepository) Schedule(ctx context.Context, messagesPerMinute uint, notification *entities.PhoneNotification) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if messagesPerMinute == 0 {
		return repository.insert(ctx, notification)
	}

	err := crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error {
		lastNotification := new(entities.PhoneNotification)
		err := tx.WithContext(ctx).
			Where("phone_id = ?", notification.PhoneID).
			Order("scheduled_at desc").
			First(lastNotification).
			Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			msg := fmt.Sprintf("cannot fetch last notification with phone ID [%s]", notification.PhoneID)
			return stacktrace.Propagate(err, msg)
		}

		notification.ScheduledAt = time.Now().UTC()
		if err == nil {
			notification.ScheduledAt = repository.maxTime(
				time.Now().UTC(),
				lastNotification.ScheduledAt.Add(time.Duration(60/messagesPerMinute)*time.Second),
			)
		}

		if err = tx.WithContext(ctx).Create(notification).Error; err != nil {
			msg := fmt.Sprintf("cannot create new notification with id [%s] and schedule [%s]", notification.ID, notification.ScheduledAt.String())
			return stacktrace.Propagate(err, msg)
		}
		return nil
	})
	if err != nil {
		msg := fmt.Sprintf("cannot schedule phone notification with ID [%s]", notification.ID)
		return stacktrace.Propagate(err, msg)
	}

	return nil
}

func (repository *gormPhoneNotificationRepository) maxTime(a, b time.Time) time.Time {
	if a.Unix() > b.Unix() {
		return a
	}
	return b
}

func (repository *gormPhoneNotificationRepository) insert(ctx context.Context, notification *entities.PhoneNotification) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Create(notification).Error
	if err != nil {
		msg := fmt.Sprintf("cannot store notification with id [%s]", notification.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}
