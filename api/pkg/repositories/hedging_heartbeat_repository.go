package repositories

import (
	"context"
	"fmt"

	otelMetric "go.opentelemetry.io/otel/metric"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// hedgingHeartbeatRepository writes to both primary and secondary repositories.
// Reads only hit primary. Secondary writes are fail-open.
type hedgingHeartbeatRepository struct {
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	primary        HeartbeatRepository
	secondary      HeartbeatRepository
	failureCounter otelMetric.Int64Counter
}

// NewHedgingHeartbeatRepository creates a hedging HeartbeatRepository
func NewHedgingHeartbeatRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	primary HeartbeatRepository,
	secondary HeartbeatRepository,
	failureCounter otelMetric.Int64Counter,
) HeartbeatRepository {
	return &hedgingHeartbeatRepository{
		logger:         logger.WithService(fmt.Sprintf("%T", &hedgingHeartbeatRepository{})),
		tracer:         tracer,
		primary:        primary,
		secondary:      secondary,
		failureCounter: failureCounter,
	}
}

func (repository *hedgingHeartbeatRepository) Store(ctx context.Context, heartbeat *entities.Heartbeat) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.primary.Store(ctx, heartbeat); err != nil {
		return err
	}

	if err := repository.secondary.Store(ctx, heartbeat); err != nil {
		repository.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("hedging: secondary write failed for heartbeat [%s]", heartbeat.ID)))
		repository.failureCounter.Add(ctx, 1)
	}

	return nil
}

func (repository *hedgingHeartbeatRepository) Index(ctx context.Context, userID entities.UserID, owner string, params IndexParams) (*[]entities.Heartbeat, error) {
	return repository.primary.Index(ctx, userID, owner, params)
}

func (repository *hedgingHeartbeatRepository) Last(ctx context.Context, userID entities.UserID, owner string) (*entities.Heartbeat, error) {
	return repository.primary.Last(ctx, userID, owner)
}

func (repository *hedgingHeartbeatRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.primary.DeleteAllForUser(ctx, userID); err != nil {
		return err
	}

	if err := repository.secondary.DeleteAllForUser(ctx, userID); err != nil {
		repository.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("hedging: secondary delete failed for user [%s]", userID)))
		repository.failureCounter.Add(ctx, 1)
	}

	return nil
}
