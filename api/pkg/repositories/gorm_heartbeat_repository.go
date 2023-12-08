package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormHeartbeatRepository is responsible for persisting entities.Heartbeat
type gormHeartbeatRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormHeartbeatRepository creates the GORM version of the HeartbeatRepository
func NewGormHeartbeatRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) HeartbeatRepository {
	return &gormHeartbeatRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormHeartbeatRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *gormHeartbeatRepository) Last(ctx context.Context, userID entities.UserID, owner string) (*entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	return nil, stacktrace.NewError("not implemented")

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	heartbeat := new(entities.Heartbeat)
	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("owner = ?", owner).
		Order("timestamp DESC").
		First(&heartbeat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("heartbeat with userID [%s] and owner [%s] does not exist", userID, owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load heartbeat with userID [%s] and owner [%s]", userID, owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return heartbeat, nil
}

// Index entities.Message between 2 parties
func (repository *gormHeartbeatRepository) Index(ctx context.Context, userID entities.UserID, owner string, params IndexParams) (*[]entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	return nil, stacktrace.NewError("not implemented")

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	query := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("owner = ?", owner)
	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query.Where("version LIKE ?", queryPattern)
	}

	heartbeats := new([]entities.Heartbeat)
	if err := query.Order("timestamp DESC").Limit(params.Limit).Offset(params.Skip).Find(&heartbeats).Error; err != nil {
		msg := fmt.Sprintf("cannot fetch heartbeats with owner [%s] and params [%+#v]", owner, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return heartbeats, nil
}

// Store a new entities.Message
func (repository *gormHeartbeatRepository) Store(ctx context.Context, heartbeat *entities.Heartbeat) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	return stacktrace.NewError("not implemented")

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := repository.db.WithContext(ctx).Create(heartbeat).Error; err != nil {
		msg := fmt.Sprintf("cannot save heartbeat with ID [%s]", heartbeat.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
