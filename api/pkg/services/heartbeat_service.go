package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// HeartbeatService is handles heartbeat requests
type HeartbeatService struct {
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.HeartbeatRepository
}

// NewHeartbeatService creates a new HeartbeatService
func NewHeartbeatService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.HeartbeatRepository,
) (s *HeartbeatService) {
	return &HeartbeatService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		repository: repository,
	}
}

// Index fetches the heartbeats for a phone number
func (service *HeartbeatService) Index(ctx context.Context, userID entities.UserID, owner string, params repositories.IndexParams) (*[]entities.Heartbeat, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	heartbeats, err := service.repository.Index(ctx, userID, owner, params)
	if err != nil {
		msg := fmt.Sprintf("could not fetch heartbeats with parms [%+#v]", params)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] messages with prams [%+#v]", len(*heartbeats), params))
	return heartbeats, nil
}

// HeartbeatStoreParams are parameters for creating a new entities.Heartbeat
type HeartbeatStoreParams struct {
	Owner     string
	Timestamp time.Time
	UserID    entities.UserID
	MessageID uuid.UUID
}

// Store a new entities.Heartbeat
func (service *HeartbeatService) Store(ctx context.Context, params HeartbeatStoreParams) (*entities.Heartbeat, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	heartbeat := &entities.Heartbeat{
		ID:        uuid.New(),
		Owner:     params.Owner,
		Timestamp: params.Timestamp,
		MessageID: params.MessageID,
		UserID:    params.UserID,
	}

	if err := service.repository.Store(ctx, heartbeat); err != nil {
		msg := fmt.Sprintf("cannot save heartbeat with id [%s]", heartbeat.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat saved with id [%s] in the userRepository", heartbeat.ID))
	return heartbeat, nil
}
