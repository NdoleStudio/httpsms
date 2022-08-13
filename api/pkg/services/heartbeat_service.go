package services

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"
	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

const (
	heartbeatCheckInterval = 16 * time.Minute
)

// HeartbeatService is handles heartbeat requests
type HeartbeatService struct {
	service
	logger            telemetry.Logger
	tracer            telemetry.Tracer
	repository        repositories.HeartbeatRepository
	monitorRepository repositories.HeartbeatMonitorRepository
	dispatcher        EventDispatcher
}

// NewHeartbeatService creates a new HeartbeatService
func NewHeartbeatService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.HeartbeatRepository,
	monitorRepository repositories.HeartbeatMonitorRepository,
	dispatcher EventDispatcher,
) (s *HeartbeatService) {
	return &HeartbeatService{
		logger:            logger.WithService(fmt.Sprintf("%T", s)),
		tracer:            tracer,
		repository:        repository,
		monitorRepository: monitorRepository,
		dispatcher:        dispatcher,
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
		UserID:    params.UserID,
	}

	if err := service.repository.Store(ctx, heartbeat); err != nil {
		msg := fmt.Sprintf("cannot save heartbeat with id [%s]", heartbeat.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat saved with id [%s] in the userRepository", heartbeat.ID))
	return heartbeat, nil
}

// HeartbeatMonitorStoreParams are parameters for creating a new entities.Heartbeat
type HeartbeatMonitorStoreParams struct {
	Owner   string
	PhoneID uuid.UUID
	UserID  entities.UserID
}

// StoreMonitor a new entities.HeartbeatMonitor
func (service *HeartbeatService) StoreMonitor(ctx context.Context, params HeartbeatMonitorStoreParams) (*entities.HeartbeatMonitor, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	exists, err := service.monitorRepository.Exists(ctx, params.UserID, params.Owner)
	if err != nil {
		msg := fmt.Sprintf("cannot check if monitor exists with userID [%s] and owner [%s]", params.UserID, params.Owner)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if exists {
		ctxLogger.Info(fmt.Sprintf("heartbeat monitor exists for owner [%s] and user [%s]", params.Owner, params.UserID))
		return nil, nil
	}

	heartbeatMonitor := &entities.HeartbeatMonitor{
		ID:      uuid.New(),
		PhoneID: params.PhoneID,
		UserID:  params.UserID,
		Owner:   params.Owner,
	}

	if err = service.monitorRepository.Store(ctx, heartbeatMonitor); err != nil {
		msg := fmt.Sprintf("cannot save heartbeat monitor for owner [%s] and user [%s]", heartbeatMonitor.Owner, heartbeatMonitor.UserID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat monitor saved with id [%s] for wner [%s] and user [%s]", heartbeatMonitor.ID, heartbeatMonitor.Owner, heartbeatMonitor.UserID))
	return heartbeatMonitor, nil
}

// DeleteMonitor an entities.HeartbeatMonitor
func (service *HeartbeatService) DeleteMonitor(ctx context.Context, userID entities.UserID, owner string) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if err := service.monitorRepository.Delete(ctx, userID, owner); err != nil {
		msg := fmt.Sprintf("cannot delete heartbeat monitor with userID [%s] and owner [%s]", userID, owner)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat monitor deleted for userID [%s] and owner [%s]", userID, owner))
	return nil
}

// HeartbeatMonitorParams are parameters for monitoring the heartbeat
type HeartbeatMonitorParams struct {
	Owner   string
	PhoneID uuid.UUID
	UserID  entities.UserID
	Source  string
}

// Monitor the heartbeats of an owner and phone number
func (service *HeartbeatService) Monitor(ctx context.Context, params HeartbeatMonitorParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	exists, err := service.monitorRepository.Exists(ctx, params.UserID, params.Owner)
	if err != nil {
		msg := fmt.Sprintf("cannot check if monitor exists with userID [%s] and owner [%s]", params.UserID, params.Owner)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !exists {
		ctxLogger.Info(fmt.Sprintf("heartbeat monitor does not exist for owner [%s] and user [%s]", params.Owner, params.UserID))
		return nil
	}

	heartbeat, err := service.repository.Last(ctx, params.UserID, params.Owner)
	if err != nil {
		msg := fmt.Sprintf("cannot fetch last heartbeat for userID [%s] and owner [%s]", params.UserID, params.Owner)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return service.handleFailedMonitor(ctx, params, false)
	}

	if time.Now().UTC().Sub(heartbeat.Timestamp) > heartbeatCheckInterval &&
		time.Now().UTC().Sub(heartbeat.Timestamp) < (heartbeatCheckInterval*2) {
		ctxLogger.Error(stacktrace.NewError(fmt.Sprintf("last heartbeat was at [%s] which is more than [%s]", heartbeat.Timestamp, heartbeatCheckInterval)))
		return service.handleFailedMonitor(ctx, params, true)
	}

	return service.handlePassingMonitor(ctx, heartbeat, params)
}

func (service *HeartbeatService) handleFailedMonitor(ctx context.Context, params HeartbeatMonitorParams, raiseEvent bool) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	event, err := service.createPhoneHeartbeatCheckEvent(params.Source, &events.PhoneHeartbeatCheckPayload{
		PhoneID:     params.PhoneID,
		UserID:      params.UserID,
		ScheduledAt: time.Now().UTC().Add(heartbeatCheckInterval),
		Owner:       params.Owner,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event when phone monitor failed")
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.DispatchWithTimeout(ctx, event, heartbeatCheckInterval); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), params.PhoneID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !raiseEvent {
		return nil
	}

	event, err = service.createPhoneHeartbeatDeadEvent(params.Source, &events.PhoneHeartbeatDeadPayload{
		PhoneID:   params.PhoneID,
		UserID:    params.UserID,
		Timestamp: time.Now().UTC(),
		Owner:     params.Owner,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event when phone monitor failed")
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.DispatchWithTimeout(ctx, event, heartbeatCheckInterval); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), params.PhoneID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (service *HeartbeatService) handlePassingMonitor(ctx context.Context, heartbeat *entities.Heartbeat, params HeartbeatMonitorParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	event, err := service.createPhoneHeartbeatCheckEvent(params.Source, &events.PhoneHeartbeatCheckPayload{
		PhoneID:     params.PhoneID,
		UserID:      params.UserID,
		ScheduledAt: heartbeat.Timestamp.Add(heartbeatCheckInterval),
		Owner:       params.Owner,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event when phone monitor failed")
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.DispatchWithTimeout(ctx, event, heartbeatCheckInterval); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), params.PhoneID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	return nil
}

func (service *HeartbeatService) createPhoneHeartbeatDeadEvent(source string, payload *events.PhoneHeartbeatDeadPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypePhoneHeartbeatDead, source, payload)
}

func (service *HeartbeatService) createPhoneHeartbeatCheckEvent(source string, payload *events.PhoneHeartbeatCheckPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypePhoneHeartbeatCheck, source, payload)
}
