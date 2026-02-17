package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/pkg/errors"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormHeartbeatRepository is responsible for persisting entities.Heartbeat
type gormHeartbeatMonitorRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormHeartbeatMonitorRepository creates the GORM version of the HeartbeatMonitorRepository
func NewGormHeartbeatMonitorRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) HeartbeatMonitorRepository {
	return &gormHeartbeatMonitorRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormHeartbeatRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *gormHeartbeatMonitorRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	return executeWithRetry(func() error {
		if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.HeartbeatMonitor{}).Error; err != nil {
			msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.HeartbeatMonitor{}, userID)
			return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}
		return nil
	})
}

// UpdatePhoneOnline updates the online status of a phone
func (repository *gormHeartbeatMonitorRepository) UpdatePhoneOnline(ctx context.Context, userID entities.UserID, monitorID uuid.UUID, isOnline bool) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	err := executeWithRetry(func() error {
		return repository.db.
			Model(&entities.HeartbeatMonitor{}).
			Where("id = ?", monitorID).
			Where("user_id = ?", userID).
			Updates(map[string]any{
				"phone_online": isOnline,
				"updated_at":   time.Now().UTC(),
			}).Error
	})
	if err != nil {
		msg := fmt.Sprintf("cannot update heartbeat monitor ID [%s] for user [%s]", monitorID, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}

// UpdateQueueID updates the queueID of a monitor
func (repository *gormHeartbeatMonitorRepository) UpdateQueueID(ctx context.Context, monitorID uuid.UUID, queueID string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	err := executeWithRetry(func() error {
		return repository.db.
			Model(&entities.HeartbeatMonitor{}).
			Where("id = ?", monitorID).
			Updates(map[string]any{
				"queue_id":   queueID,
				"updated_at": time.Now().UTC(),
			}).Error
	})
	if err != nil {
		msg := fmt.Sprintf("cannot update heartbeat monitor ID [%s]", monitorID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}

func (repository *gormHeartbeatMonitorRepository) Delete(ctx context.Context, userID entities.UserID, owner string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	err := executeWithRetry(func() error {
		return repository.db.WithContext(ctx).
			Where("user_id = ?", userID).
			Where("owner = ?", owner).
			Delete(&entities.HeartbeatMonitor{}).Error
	})
	if err != nil {
		msg := fmt.Sprintf("cannot delete heartbeat monitor with owner [%s] and userID [%s]", owner, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Index entities.Message between 2 parties
func (repository *gormHeartbeatMonitorRepository) Index(ctx context.Context, userID entities.UserID, owner string, params IndexParams) (*[]entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	query := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("owner = ?", owner)
	heartbeats := new([]entities.Heartbeat)
	if err := executeWithRetry(func() error {
		return query.Order("timestamp DESC").Limit(params.Limit).Offset(params.Skip).Find(&heartbeats).Error
	}); err != nil {
		msg := fmt.Sprintf("cannot fetch heartbeats with owner [%s] and params [%+#v]", owner, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return heartbeats, nil
}

// Store a new heartbeat monitor
func (repository *gormHeartbeatMonitorRepository) Store(ctx context.Context, heartbeatMonitor *entities.HeartbeatMonitor) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	if err := executeWithRetry(func() error { return repository.db.WithContext(ctx).Create(heartbeatMonitor).Error }); err != nil {
		msg := fmt.Sprintf("cannot save heartbeatMonitor monitor with ID [%s]", heartbeatMonitor.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Load a heartbeat monitor by userID and owner
func (repository *gormHeartbeatMonitorRepository) Load(ctx context.Context, userID entities.UserID, owner string) (*entities.HeartbeatMonitor, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	phone := new(entities.HeartbeatMonitor)
	err := executeWithRetry(func() error {
		return repository.db.WithContext(ctx).
			Where("user_id = ?", userID).
			Where("owner = ?", owner).
			First(&phone).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("heartbeat monitor with userID [%s] and owner [%s] does not exist", userID, owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load heartbeat monitor with userID [%s] and owner [%s]", userID, owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return phone, nil
}

// Exists checks of a heartbeat monitor exists for the userID and owner
func (repository *gormHeartbeatMonitorRepository) Exists(ctx context.Context, userID entities.UserID, monitorID uuid.UUID) (bool, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	var exists bool
	err := executeWithRetry(func() error {
		return repository.db.WithContext(ctx).
			Model(&entities.HeartbeatMonitor{}).
			Select("count(*) > 0").
			Where("user_id = ?", userID).
			Where("id = ?", monitorID).
			Find(&exists).Error
	})
	if err != nil {
		msg := fmt.Sprintf("cannot check if heartbeat monitor exists with userID [%s] and montior ID [%s]", userID, monitorID)
		return exists, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return exists, nil
}
