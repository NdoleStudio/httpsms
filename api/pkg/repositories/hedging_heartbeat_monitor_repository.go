package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	otelMetric "go.opentelemetry.io/otel/metric"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// hedgingHeartbeatMonitorRepository writes to both primary and secondary repositories.
// Reads only hit primary. Secondary writes are fail-open.
type hedgingHeartbeatMonitorRepository struct {
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	primary        HeartbeatMonitorRepository
	secondary      HeartbeatMonitorRepository
	failureCounter otelMetric.Int64Counter
}

// NewHedgingHeartbeatMonitorRepository creates a hedging HeartbeatMonitorRepository
func NewHedgingHeartbeatMonitorRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	primary HeartbeatMonitorRepository,
	secondary HeartbeatMonitorRepository,
	failureCounter otelMetric.Int64Counter,
) HeartbeatMonitorRepository {
	return &hedgingHeartbeatMonitorRepository{
		logger:         logger.WithService(fmt.Sprintf("%T", &hedgingHeartbeatMonitorRepository{})),
		tracer:         tracer,
		primary:        primary,
		secondary:      secondary,
		failureCounter: failureCounter,
	}
}

func (repository *hedgingHeartbeatMonitorRepository) Store(ctx context.Context, monitor *entities.HeartbeatMonitor) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.primary.Store(ctx, monitor); err != nil {
		return err
	}

	if err := repository.secondary.Store(ctx, monitor); err != nil {
		repository.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("hedging: secondary write failed for monitor [%s]", monitor.ID)))
		repository.failureCounter.Add(ctx, 1)
	}

	return nil
}

func (repository *hedgingHeartbeatMonitorRepository) Load(ctx context.Context, userID entities.UserID, phoneNumber string) (*entities.HeartbeatMonitor, error) {
	return repository.primary.Load(ctx, userID, phoneNumber)
}

func (repository *hedgingHeartbeatMonitorRepository) Exists(ctx context.Context, userID entities.UserID, monitorID uuid.UUID) (bool, error) {
	return repository.primary.Exists(ctx, userID, monitorID)
}

func (repository *hedgingHeartbeatMonitorRepository) UpdateQueueID(ctx context.Context, monitorID uuid.UUID, queueID string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.primary.UpdateQueueID(ctx, monitorID, queueID); err != nil {
		return err
	}

	if err := repository.secondary.UpdateQueueID(ctx, monitorID, queueID); err != nil {
		repository.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("hedging: secondary UpdateQueueID failed for monitor [%s]", monitorID)))
		repository.failureCounter.Add(ctx, 1)
	}

	return nil
}

func (repository *hedgingHeartbeatMonitorRepository) Delete(ctx context.Context, userID entities.UserID, phoneNumber string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.primary.Delete(ctx, userID, phoneNumber); err != nil {
		return err
	}

	if err := repository.secondary.Delete(ctx, userID, phoneNumber); err != nil {
		repository.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("hedging: secondary delete failed for monitor with owner [%s]", phoneNumber)))
		repository.failureCounter.Add(ctx, 1)
	}

	return nil
}

func (repository *hedgingHeartbeatMonitorRepository) UpdatePhoneOnline(ctx context.Context, userID entities.UserID, monitorID uuid.UUID, online bool) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.primary.UpdatePhoneOnline(ctx, userID, monitorID, online); err != nil {
		return err
	}

	if err := repository.secondary.UpdatePhoneOnline(ctx, userID, monitorID, online); err != nil {
		repository.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("hedging: secondary UpdatePhoneOnline failed for monitor [%s]", monitorID)))
		repository.failureCounter.Add(ctx, 1)
	}

	return nil
}

func (repository *hedgingHeartbeatMonitorRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.primary.DeleteAllForUser(ctx, userID); err != nil {
		return err
	}

	if err := repository.secondary.DeleteAllForUser(ctx, userID); err != nil {
		repository.logger.Error(stacktrace.Propagate(err, fmt.Sprintf("hedging: secondary delete all failed for user [%s]", userID)))
		repository.failureCounter.Add(ctx, 1)
	}

	return nil
}
