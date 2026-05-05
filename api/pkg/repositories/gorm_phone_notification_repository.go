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

// gormPhoneNotificationRepository persists entities.PhoneNotification records.
type gormPhoneNotificationRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormPhoneNotificationRepository creates a GORM-backed PhoneNotificationRepository.
func NewGormPhoneNotificationRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) PhoneNotificationRepository {
	return &gormPhoneNotificationRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormPhoneNotificationRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// DeleteAllForUser deletes all phone notifications that belong to a user.
func (repository *gormPhoneNotificationRepository) DeleteAllForUser(
	ctx context.Context,
	userID entities.UserID,
) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&entities.PhoneNotification{}).Error; err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(
				err,
				"cannot delete all [%T] for user with ID [%s]",
				&entities.PhoneNotification{},
				userID,
			),
		)
	}

	return nil
}

// DeleteByMessageID deletes all entities.PhoneNotification for a user and message ID.
func (repository *gormPhoneNotificationRepository) DeleteByMessageID(ctx context.Context, userID entities.UserID, messageID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).
		Where("user_id = ? AND message_id = ?", userID, messageID).
		Delete(&entities.PhoneNotification{}).Error
	if err != nil {
		msg := fmt.Sprintf("cannot delete [%T] for user [%s] and message with ID [%s]", &entities.PhoneNotification{}, userID, messageID)
		return repository.tracer.WrapErrorSpan(span,
			stacktrace.Propagate(err, msg),
		)
	}

	return nil
}

// UpdateStatus updates the status of a phone notification.
func (repository *gormPhoneNotificationRepository) UpdateStatus(ctx context.Context, notificationID uuid.UUID, status entities.PhoneNotificationStatus) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.
		WithContext(ctx).
		Model(&entities.PhoneNotification{ID: notificationID}).
		Update("status", status).
		Error
	if err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(
				err,
				"cannot update notification [%s] with status [%s]",
				notificationID,
				status,
			),
		)
	}

	return nil
}

// Schedule stores a phone notification and calculates its final scheduled time.
// The final time is determined by combining:
// 1. the next allowed time from the message send schedule
// 2. the phone send-rate limit derived from the latest scheduled notification
func (repository *gormPhoneNotificationRepository) Schedule(
	ctx context.Context,
	messagesPerMinute uint,
	schedule *entities.MessageSendSchedule,
	notification *entities.PhoneNotification,
) error {
	ctx, span, _ := repository.tracer.StartWithLogger(ctx, repository.logger)
	defer span.End()

	now := time.Now().UTC()

	if messagesPerMinute == 0 {
		notification.ScheduledAt = repository.resolveScheduledAt(schedule, now)
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
			return stacktrace.Propagate(
				err,
				"cannot fetch last notification with phone ID [%s]",
				notification.PhoneID,
			)
		}

		notification.ScheduledAt = repository.resolveScheduledAt(schedule, now)

		if err == nil {
			rateLimitedAt := lastNotification.ScheduledAt.Add(
				time.Duration(60/messagesPerMinute) * time.Second,
			)

			nextCandidate := repository.maxTime(notification.ScheduledAt, rateLimitedAt)
			notification.ScheduledAt = repository.resolveScheduledAt(schedule, nextCandidate)
		}

		if err = tx.WithContext(ctx).Create(notification).Error; err != nil {
			return stacktrace.Propagate(
				err,
				"cannot create new notification with id [%s] and schedule [%s]",
				notification.ID,
				notification.ScheduledAt.String(),
			)
		}

		return nil
	})
	if err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(
				err,
				"cannot schedule phone notification with ID [%s]",
				notification.ID,
			),
		)
	}

	return nil
}

// resolveScheduledAt returns the next time the notification is allowed to be sent.
// If no schedule is attached, the provided time is returned unchanged in UTC.
func (repository *gormPhoneNotificationRepository) resolveScheduledAt(
	schedule *entities.MessageSendSchedule,
	current time.Time,
) time.Time {
	if schedule == nil {
		return current.UTC()
	}

	return schedule.ResolveScheduledAt(current)
}

// maxTime returns the greater of the two time.Time.
func (repository *gormPhoneNotificationRepository) maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// insert stores a single phone notification.
func (repository *gormPhoneNotificationRepository) insert(
	ctx context.Context,
	notification *entities.PhoneNotification,
) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Create(notification).Error; err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(
				err,
				"cannot store notification with id [%s]",
				notification.ID,
			),
		)
	}

	return nil
}

// ScheduleExact stores a phone notification with an exact ScheduledAt time.
// It performs a dedupe check — if a pending notification for the same message already exists, it's a no-op.
func (repository *gormPhoneNotificationRepository) ScheduleExact(
	ctx context.Context,
	notification *entities.PhoneNotification,
) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	// Dedupe: check if a pending notification for this message already exists
	var count int64
	if err := repository.db.WithContext(ctx).
		Model(&entities.PhoneNotification{}).
		Where("message_id = ? AND status = ?", notification.MessageID, entities.PhoneNotificationStatusPending).
		Count(&count).Error; err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(err, "cannot check for existing notification for message [%s]", notification.MessageID),
		)
	}

	if count > 0 {
		return nil
	}

	if err := repository.db.WithContext(ctx).Create(notification).Error; err != nil {
		return repository.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(err, "cannot create exact-time notification with id [%s]", notification.ID),
		)
	}

	return nil
}
