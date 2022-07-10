package repositories

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"
	"github.com/google/uuid"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
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

// UpdateStatus of an entities.PhoneNotification
func (repository gormPhoneNotificationRepository) UpdateStatus(ctx context.Context, notificationID uuid.UUID, status entities.PhoneNotificationStatus) error {
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
func (repository gormPhoneNotificationRepository) Schedule(ctx context.Context, messagesPerMinute uint, notification *entities.PhoneNotification) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if messagesPerMinute == 0 {
		return repository.insert(ctx, notification)
	}

	err := crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error {
		var messagesCount int64
		err := tx.WithContext(ctx).
			Where("phone_id = ?", notification.PhoneID).
			Where("status = ?", entities.PhoneNotificationStatusPending).
			Count(&messagesCount).
			Error
		if err != nil {
			msg := fmt.Sprintf("cannot count messages with phoneID [%s] and status [%s]", notification.PhoneID, entities.PhoneNotificationStatusPending)
			return stacktrace.Propagate(err, msg)
		}

		timeout := int(math.Ceil(float64(messagesCount) / float64(messagesPerMinute))) // how many minutes to wait
		notification.ScheduledAt = time.Now().UTC().Add(time.Minute * time.Duration(timeout))

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
