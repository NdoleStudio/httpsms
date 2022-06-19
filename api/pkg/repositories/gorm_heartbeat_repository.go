package repositories

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
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

// Index entities.Message between 2 parties
func (repository *gormHeartbeatRepository) Index(ctx context.Context, owner string, params IndexParams) (*[]entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.Where("owner = ?", owner)
	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query.Where("quantity ILIKE ?", queryPattern)
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

	if err := repository.db.Create(heartbeat).Error; err != nil {
		msg := fmt.Sprintf("cannot save heartbeat with ID [%s]", heartbeat.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
