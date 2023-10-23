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
	// select id, a.timestamp, a.owner,  a.timestamp - (SELECT timestamp from heartbeats b where  b.timestamp < a.timestamp and a.owner = b.owner and a.user_id = b.user_id order by b.timestamp desc  limit 1) as diff  from heartbeats a;
	heartbeatCheckInterval = 16 * time.Minute
)

// HeartbeatService is handles heartbeat requests
type HeartbeatService struct {
	service
	logger            telemetry.Logger
	tracer            telemetry.Tracer
	repository        repositories.HeartbeatRepository
	monitorRepository repositories.HeartbeatMonitorRepository
	dispatcher        *EventDispatcher
}

// NewHeartbeatService creates a new HeartbeatService
func NewHeartbeatService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.HeartbeatRepository,
	monitorRepository repositories.HeartbeatMonitorRepository,
	dispatcher *EventDispatcher,
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
	Version   string
	Charging  bool
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
		Charging:  params.Charging,
		Timestamp: params.Timestamp,
		Version:   params.Version,
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
	Source  string
	UserID  entities.UserID
}

// StoreMonitor a new entities.HeartbeatMonitor
func (service *HeartbeatService) StoreMonitor(ctx context.Context, params *HeartbeatMonitorStoreParams) (*entities.HeartbeatMonitor, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	monitor, scheduleCheck, err := service.phoneMonitor(ctx, params)
	if err != nil {
		msg := fmt.Sprintf("cannot create monitor for with userID [%s] and owner [%s]", params.UserID, params.Owner)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !scheduleCheck {
		ctxLogger.Info(fmt.Sprintf("heartbeat monitor [%s] for owner [%s] does not need scheduling because it was updated at [%s]", monitor.ID, monitor.Owner, monitor.UpdatedAt))
		return monitor, nil
	}

	ctxLogger.Info(fmt.Sprintf("scheduling heartbeat monitor [%s] for owner [%s] and user [%s]", monitor.ID, monitor.Owner, monitor.UserID))

	monitorParams := &HeartbeatMonitorParams{
		Owner:     monitor.Owner,
		PhoneID:   monitor.PhoneID,
		UserID:    monitor.UserID,
		MonitorID: monitor.ID,
		Source:    params.Source,
	}
	if err = service.scheduleHeartbeatCheck(ctx, time.Now().UTC(), monitorParams); err != nil {
		msg := fmt.Sprintf("cannot schedule healthcheck for monitor [%s] with owner [%s] and userID [%s]", monitor.ID, params.Owner, params.UserID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return monitor, nil
}

func (service *HeartbeatService) phoneMonitor(ctx context.Context, params *HeartbeatMonitorStoreParams) (*entities.HeartbeatMonitor, bool, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	monitor, err := service.monitorRepository.Load(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		monitor = &entities.HeartbeatMonitor{
			ID:        uuid.New(),
			PhoneID:   params.PhoneID,
			UserID:    params.UserID,
			Owner:     params.Owner,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		if err = service.monitorRepository.Store(ctx, monitor); err != nil {
			msg := fmt.Sprintf("cannot save heartbeat monitor for owner [%s] and user [%s]", monitor.Owner, monitor.UserID)
			return nil, false, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}

		ctxLogger.Info(fmt.Sprintf("heartbeat monitor saved with id [%s] for owner [%s] and user [%s]", monitor.ID, monitor.Owner, monitor.UserID))
		return monitor, true, nil
	}

	if err != nil {
		msg := fmt.Sprintf("cannot check if monitor exists with userID [%s] and owner [%s]", params.UserID, params.Owner)
		return nil, false, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return monitor, monitor.RequiresCheck(), nil
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
	Owner     string
	MonitorID uuid.UUID
	PhoneID   uuid.UUID
	UserID    entities.UserID
	Source    string
}

// Monitor the heartbeats of an owner and phone number
func (service *HeartbeatService) Monitor(ctx context.Context, params *HeartbeatMonitorParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	exists, err := service.monitorRepository.Exists(ctx, params.UserID, params.MonitorID)
	if err != nil {
		msg := fmt.Sprintf("cannot check if monitor exists with userID [%s] and owner [%s]", params.UserID, params.Owner)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return service.scheduleHeartbeatCheck(ctx, time.Now().UTC(), params)
	}

	if !exists {
		ctxLogger.Info(fmt.Sprintf("heartbeat monitor does not exist for owner [%s] and user [%s]", params.Owner, params.UserID))
		return nil
	}

	heartbeat, err := service.repository.Last(ctx, params.UserID, params.Owner)
	if err != nil {
		msg := fmt.Sprintf("cannot fetch last heartbeat for userID [%s] and owner [%s] and ID [%s] removing check", params.UserID, params.Owner, params.MonitorID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return nil
	}

	// send urgent FCM message if the last heartbeat is late
	if time.Now().UTC().Sub(heartbeat.Timestamp) > heartbeatCheckInterval && time.Now().UTC().Sub(heartbeat.Timestamp) < (heartbeatCheckInterval*5) {
		ctxLogger.Info(fmt.Sprintf("sending missed heartbeat notification for userID [%s] and owner [%s] and monitor ID [%s]", params.UserID, params.Owner, params.MonitorID))
		service.handleMissedMonitor(ctx, heartbeat.Timestamp, params)
	}

	if time.Now().UTC().Sub(heartbeat.Timestamp) > (heartbeatCheckInterval*4) &&
		time.Now().UTC().Sub(heartbeat.Timestamp) < (heartbeatCheckInterval*5) {
		return service.handleFailedMonitor(ctx, heartbeat.Timestamp, params)
	}

	return service.scheduleHeartbeatCheck(ctx, heartbeat.Timestamp, params)
}

func (service *HeartbeatService) handleMissedMonitor(ctx context.Context, lastTimestamp time.Time, params *HeartbeatMonitorParams) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	event, err := service.createPhoneHeartbeatMissedEvent(params.Source, &events.PhoneHeartbeatMissedPayload{
		PhoneID:                params.PhoneID,
		UserID:                 params.UserID,
		MonitorID:              params.MonitorID,
		LastHeartbeatTimestamp: lastTimestamp,
		Timestamp:              time.Now().UTC(),
		Owner:                  params.Owner,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event when phone monitor [%s] missed heartbeat", params.MonitorID.String())
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return
	}

	if _, err = service.dispatcher.DispatchWithTimeout(ctx, event, heartbeatCheckInterval); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), params.PhoneID)
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
	}
}

func (service *HeartbeatService) handleFailedMonitor(ctx context.Context, lastTimestamp time.Time, params *HeartbeatMonitorParams) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	err := service.scheduleHeartbeatCheck(ctx, time.Now().UTC(), params)
	if err != nil {
		msg := fmt.Sprintf("cannot schedule healthcheck for monitor with owner [%s] and userID [%s]", params.Owner, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	event, err := service.createPhoneHeartbeatDeadEvent(params.Source, &events.PhoneHeartbeatDeadPayload{
		PhoneID:                params.PhoneID,
		UserID:                 params.UserID,
		MonitorID:              params.MonitorID,
		LastHeartbeatTimestamp: lastTimestamp,
		Timestamp:              time.Now().UTC(),
		Owner:                  params.Owner,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event when phone monitor failed")
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), params.PhoneID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat monitor with id [%s] and phone id [%s] failed for user [%s]", params.MonitorID, params.PhoneID, params.UserID))
	return nil
}

func (service *HeartbeatService) scheduleHeartbeatCheck(ctx context.Context, lastTimestamp time.Time, params *HeartbeatMonitorParams) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	event, err := service.createPhoneHeartbeatCheckEvent(params.Source, &events.PhoneHeartbeatCheckPayload{
		PhoneID:     params.PhoneID,
		UserID:      params.UserID,
		MonitorID:   params.MonitorID,
		ScheduledAt: lastTimestamp.Add(heartbeatCheckInterval),
		Owner:       params.Owner,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event when phone monitor failed")
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	queueID, err := service.dispatcher.DispatchWithTimeout(ctx, event, heartbeatCheckInterval)
	if err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), params.PhoneID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.monitorRepository.UpdateQueueID(ctx, params.MonitorID, queueID); err != nil {
		msg := fmt.Sprintf("cannot update monitor with id [%s] with queue with ID [%s]", params.MonitorID, queueID)
		service.logger.Error(stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat check scheduled for monitor with id [%s] and phone id [%s] and queue id [%s] for user [%s]", params.MonitorID, params.PhoneID, queueID, params.UserID))

	return nil
}

func (service *HeartbeatService) createPhoneHeartbeatMissedEvent(source string, payload *events.PhoneHeartbeatMissedPayload) (cloudevents.Event, error) {
	return service.createEvent(events.PhoneHeartbeatMissed, source, payload)
}

func (service *HeartbeatService) createPhoneHeartbeatDeadEvent(source string, payload *events.PhoneHeartbeatDeadPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypePhoneHeartbeatDead, source, payload)
}

func (service *HeartbeatService) createPhoneHeartbeatCheckEvent(source string, payload *events.PhoneHeartbeatCheckPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypePhoneHeartbeatCheck, source, payload)
}
