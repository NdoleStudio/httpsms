package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"
	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

const (
	// select id, a.timestamp, a.owner,  a.timestamp - (SELECT timestamp from heartbeats b where  b.timestamp < a.timestamp and a.owner = b.owner and a.user_id = b.user_id order by b.timestamp desc  limit 1) as diff  from heartbeats a;
	heartbeatCheckInterval = 16 * time.Minute
	heartbeatClaimLease    = entities.HeartbeatCheckClaimLease
	legacyHeartbeatCheckID = "legacy"
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

// DeleteAllForUser deletes all entities.Heartbeat for an entities.UserID.
func (service *HeartbeatService) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.repository.DeleteAllForUser(ctx, userID); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "could not delete all [entities.Heartbeat] for user with ID [%s]", userID))
	}

	if err := service.monitorRepository.DeleteAllForUser(ctx, userID); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "could not delete all [entities.HeartbeatMonitor] for user with ID [%s]", userID))
	}

	ctxLogger.Info(fmt.Sprintf("deleted all [entities.Heartbeat] and [entities.HeartbeatMonitor] for user with ID [%s]", userID))
	return nil
}

// Index fetches the heartbeats for a phone number
func (service *HeartbeatService) Index(ctx context.Context, userID entities.UserID, owner string, params repositories.IndexParams) (*[]entities.Heartbeat, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	heartbeats, err := service.repository.Index(ctx, userID, owner, params)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "could not fetch heartbeats with parms [%+#v]", params))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] entities.Heartbeat with prams [%+#v]", len(*heartbeats), params))
	return heartbeats, nil
}

// HeartbeatStoreParams are parameters for creating a new entities.Heartbeat
type HeartbeatStoreParams struct {
	Owner     string
	Version   string
	Charging  bool
	Source    string
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
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot save heartbeat with id [%s]", heartbeat.ID))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat saved with id [%s] for user [%s]", heartbeat.ID, heartbeat.UserID))

	monitor, err := service.monitorRepository.Load(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		ctxLogger.Info(fmt.Sprintf("heartbeat monitor does not exist for owner [%s] and user [%s]", params.Owner, params.UserID))
		return heartbeat, nil
	}
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot load heartbeat monitor for owner [%s] and user [%s]", params.Owner, params.UserID))
		return heartbeat, nil
	}

	service.restartMonitorOnHeartbeat(ctx, heartbeat, monitor, params.Source)

	if monitor.PhoneIsOffline() {
		ctxLogger.Info(fmt.Sprintf("phone with monitor ID [%s] was offline for user [%s]", monitor.ID, monitor.UserID))
		service.handleHeartbeatWhenPhoneWasOffline(ctx, params.Source, heartbeat, monitor)
	}

	return heartbeat, nil
}

// HeartbeatMonitorStoreParams are parameters for creating a new entities.Heartbeat
type HeartbeatMonitorStoreParams struct {
	Owner   string
	PhoneID uuid.UUID
	Source  string
	UserID  entities.UserID
}

func (service *HeartbeatService) handleHeartbeatWhenPhoneWasOffline(ctx context.Context, source string, heartbeat *entities.Heartbeat, monitor *entities.HeartbeatMonitor) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.UpdatePhoneOnline(ctx, monitor.UserID, monitor.ID, true); err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update phone online status for heartbeat monitor [%s]", monitor.ID)))
	}

	event, err := service.createEvent(events.EventTypePhoneHeartbeatOnline, source, &events.PhoneHeartbeatOnlinePayload{
		PhoneID:                monitor.PhoneID,
		UserID:                 monitor.UserID,
		LastHeartbeatTimestamp: heartbeat.Timestamp,
		Timestamp:              time.Now().UTC(),
		MonitorID:              monitor.ID,
		Owner:                  heartbeat.Owner,
	})
	if err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot create [%s] event for monitor with ID [%s]", events.EventTypePhoneHeartbeatOnline, monitor.ID)))
		return
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), monitor.PhoneID)))
		return
	}

	ctxLogger.Info(fmt.Sprintf("[%s] event created with ID [%s] for monitor ID [%s] and user [%s]", event.Type(), event.ID(), monitor.ID, monitor.UserID))
	return
}

// StoreMonitor a new entities.HeartbeatMonitor
func (service *HeartbeatService) StoreMonitor(ctx context.Context, params *HeartbeatMonitorStoreParams) (*entities.HeartbeatMonitor, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	monitor, scheduleCheck, err := service.phoneMonitor(ctx, params)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot create monitor for with userID [%s] and owner [%s]", params.UserID, params.Owner))
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
	if err = service.scheduleHeartbeatCheck(ctx, time.Now().UTC(), monitorParams, monitor, true); err != nil {
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot schedule healthcheck for monitor [%s] with owner [%s] and userID [%s]", monitor.ID, params.Owner, params.UserID))
	}

	return monitor, nil
}

func (service *HeartbeatService) phoneMonitor(ctx context.Context, params *HeartbeatMonitorStoreParams) (*entities.HeartbeatMonitor, bool, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	monitor, err := service.monitorRepository.Load(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		monitor = &entities.HeartbeatMonitor{
			ID:          uuid.New(),
			PhoneID:     params.PhoneID,
			UserID:      params.UserID,
			Owner:       params.Owner,
			PhoneOnline: true,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		if err = service.monitorRepository.Store(ctx, monitor); err != nil {
			return nil, false, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot save heartbeat monitor for owner [%s] and user [%s]", monitor.Owner, monitor.UserID))
		}

		ctxLogger.Info(fmt.Sprintf("heartbeat monitor saved with id [%s] for owner [%s] and user [%s]", monitor.ID, monitor.Owner, monitor.UserID))
		return monitor, true, nil
	}

	if err != nil {
		return nil, false, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot check if monitor exists with userID [%s] and owner [%s]", params.UserID, params.Owner))
	}

	return monitor, monitor.RequiresCheck(), nil
}

// DeleteMonitor an entities.HeartbeatMonitor
func (service *HeartbeatService) DeleteMonitor(ctx context.Context, userID entities.UserID, owner string) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if err := service.monitorRepository.Delete(ctx, userID, owner); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete heartbeat monitor with userID [%s] and owner [%s]", userID, owner))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat monitor deleted for userID [%s] and owner [%s]", userID, owner))
	return nil
}

// UpdatePhoneOnline updates the phone_online field in an entities.HeartbeatMonitor
func (service *HeartbeatService) UpdatePhoneOnline(ctx context.Context, userID entities.UserID, monitorID uuid.UUID, phoneOnline bool) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	if err := service.monitorRepository.UpdatePhoneOnline(ctx, userID, monitorID, phoneOnline); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update heartbeat monitor [%s] with userID [%s] and status [%t]", monitorID, userID, phoneOnline))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat monitor [%s] updated for userID [%s] and status [%t]", monitorID, userID, phoneOnline))
	return nil
}

// HeartbeatMonitorParams are parameters for monitoring the heartbeat
type HeartbeatMonitorParams struct {
	Owner     string
	MonitorID uuid.UUID
	PhoneID   uuid.UUID
	UserID    entities.UserID
	Source    string
	CheckID   string
}

// Monitor the heartbeats of an owner and phone number
func (service *HeartbeatService) Monitor(ctx context.Context, params *HeartbeatMonitorParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)
	monitor, err := service.monitorRepository.Load(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		ctxLogger.Info(fmt.Sprintf("heartbeat monitor does not exist for owner [%s] and user [%s]", params.Owner, params.UserID))
		return nil
	}
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot check if monitor exists with userID [%s] and owner [%s]", params.UserID, params.Owner))
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load heartbeat monitor for check"))
	}

	retryID, canRun, err := service.heartbeatCheckOwnership(ctx, monitor, params)
	if err != nil || !canRun {
		return err
	}
	params.PhoneID = monitor.PhoneID
	params.MonitorID = monitor.ID

	// An empty marker can follow an interrupted stop; history still gates any restart.
	heartbeat, err := service.repository.Last(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return service.stopHeartbeatCheck(ctx, params, monitor)
	}
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch heartbeat history for check"))
	}
	return service.runHeartbeatCheck(ctx, params, monitor, heartbeat, retryID)
}

func (service *HeartbeatService) restartMonitorOnHeartbeat(ctx context.Context, heartbeat *entities.Heartbeat, monitor *entities.HeartbeatMonitor, source string) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if !monitor.RequiresCheck() {
		return
	}
	latest, err := service.repository.Last(ctx, heartbeat.UserID, heartbeat.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return
	}
	if err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot validate heartbeat history before restarting monitor [%s]", monitor.ID)))
		return
	}
	params := &HeartbeatMonitorParams{
		Owner:     monitor.Owner,
		PhoneID:   monitor.PhoneID,
		UserID:    monitor.UserID,
		MonitorID: monitor.ID,
		Source:    source,
	}
	if err := service.scheduleHeartbeatCheck(ctx, latest.Timestamp, params, monitor, true); err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot restart heartbeat check for monitor [%s]", monitor.ID)))
	}
}

func (service *HeartbeatService) heartbeatCheckOwnership(ctx context.Context, monitor *entities.HeartbeatMonitor, params *HeartbeatMonitorParams) (string, bool, error) {
	_, span := service.tracer.Start(ctx)
	defer span.End()

	if params.CheckID != "" && params.MonitorID != monitor.ID {
		return "", false, nil
	}
	incomingID := params.CheckID
	if incomingID == "" {
		incomingID = legacyHeartbeatCheckID
	}
	// Claims retain the incoming check ID so redelivery can recover an interrupted worker.
	claim := strings.SplitN(monitor.QueueID, ":", 3)
	if len(claim) == 3 && (claim[0] == "running" || claim[0] == "pending") {
		deliveredSuccessor := claim[0] == "pending" && params.CheckID != "" && claim[2] == params.CheckID
		restartOnly := claim[0] == "pending" && claim[1] == ""
		if !deliveredSuccessor && !restartOnly && claim[1] != incomingID {
			return "", false, nil
		}
		if !deliveredSuccessor && monitor.UpdatedAt.Add(heartbeatClaimLease).After(time.Now()) {
			return "", false, service.tracer.WrapErrorSpan(span, stacktrace.NewErrorf("heartbeat check for monitor [%s] is still claimed; retry after its lease", monitor.ID))
		}
		retryID := ""
		if params.CheckID != "" {
			retryID = "check:" + params.CheckID
		}
		return retryID, true, nil
	}
	if params.CheckID == "" && strings.HasPrefix(monitor.QueueID, "check:") {
		return "", false, nil
	}
	if params.CheckID != "" && monitor.QueueID != "" && params.CheckID != strings.TrimPrefix(monitor.QueueID, "check:") {
		return "", false, nil
	}
	return monitor.QueueID, true, nil
}

func (service *HeartbeatService) stopHeartbeatCheck(ctx context.Context, params *HeartbeatMonitorParams, monitor *entities.HeartbeatMonitor) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	ctxLogger.Info(fmt.Sprintf("heartbeat history does not exist for userID [%s] and owner [%s], stopping check", params.UserID, params.Owner))
	stopped, err := service.monitorRepository.CompareAndSwapQueueID(ctx, monitor.ID, monitor.QueueID, "")
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot clear heartbeat check queue for monitor [%s]", monitor.ID))
	}
	if !stopped {
		return service.tracer.WrapErrorSpan(span, stacktrace.NewErrorf("heartbeat stop ownership changed for monitor [%s]; retry to verify its restart", monitor.ID))
	}
	return service.restartAfterHeartbeatStop(ctx, params)
}

func (service *HeartbeatService) restartAfterHeartbeatStop(ctx context.Context, params *HeartbeatMonitorParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	// A heartbeat may have been saved while its Store call still saw the old active marker.
	heartbeat, err := service.repository.Last(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return nil
	}
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot recheck heartbeat history after stopping monitor [%s]", params.MonitorID))
	}
	monitor, err := service.monitorRepository.Load(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return nil
	}
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot reload stopped heartbeat monitor [%s]", params.MonitorID))
	}
	if !monitor.RequiresCheck() {
		return service.requireHeartbeatCheckCommit(ctx, monitor, params.CheckID)
	}
	params.PhoneID = monitor.PhoneID
	params.MonitorID = monitor.ID
	return service.scheduleHeartbeatCheck(ctx, heartbeat.Timestamp, params, monitor, true)
}

func (service *HeartbeatService) runHeartbeatCheck(ctx context.Context, params *HeartbeatMonitorParams, monitor *entities.HeartbeatMonitor, heartbeat *entities.Heartbeat, retryID string) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	incomingID := params.CheckID
	if incomingID == "" {
		incomingID = legacyHeartbeatCheckID
	}
	processingID := "running:" + incomingID + ":" + uuid.NewString()
	claimed, err := service.monitorRepository.CompareAndSwapQueueID(ctx, monitor.ID, monitor.QueueID, processingID)
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot claim running heartbeat check for monitor [%s]", monitor.ID))
	}
	if !claimed {
		return service.tracer.WrapErrorSpan(span, stacktrace.NewErrorf("heartbeat check ownership changed for monitor [%s]; retry to verify its successor", monitor.ID))
	}
	monitor.QueueID = processingID
	// Restore the incoming check on failure so the task queue can retry it, without stealing a successor.
	defer service.releaseHeartbeatCheck(ctx, monitor.ID, processingID, retryID)
	return service.checkHeartbeatAvailability(ctx, params, monitor, heartbeat)
}

func (service *HeartbeatService) checkHeartbeatAvailability(ctx context.Context, params *HeartbeatMonitorParams, monitor *entities.HeartbeatMonitor, heartbeat *entities.Heartbeat) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	// send urgent FCM message if the last heartbeat is late
	if time.Now().UTC().Sub(heartbeat.Timestamp) > heartbeatCheckInterval && time.Now().UTC().Sub(heartbeat.Timestamp) < (heartbeatCheckInterval*5) {
		ctxLogger.Info(fmt.Sprintf("sending missed heartbeat notification for userID [%s] and owner [%s] and monitor ID [%s]", params.UserID, params.Owner, params.MonitorID))
		service.handleMissedMonitor(ctx, heartbeat.Timestamp, params)
	}

	heartbeat, owned, err := service.refreshHeartbeatCheck(ctx, params, monitor)
	if err != nil || !owned {
		return err
	}
	age := time.Since(heartbeat.Timestamp)
	failed := age > heartbeatCheckInterval*4 && age < heartbeatCheckInterval*5 && monitor.PhoneOnline
	nextTimestamp := heartbeat.Timestamp
	if failed {
		nextTimestamp = time.Now().UTC()
		if err := service.handleFailedMonitor(ctx, heartbeat, params); err != nil {
			return err
		}
	}
	return service.scheduleHeartbeatCheck(ctx, nextTimestamp, params, monitor, false)
}

func (service *HeartbeatService) refreshHeartbeatCheck(ctx context.Context, params *HeartbeatMonitorParams, monitor *entities.HeartbeatMonitor) (*entities.Heartbeat, bool, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	current, err := service.monitorRepository.Load(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot revalidate running heartbeat monitor [%s]", monitor.ID))
	}
	if current.ID != monitor.ID {
		return nil, false, nil
	}
	if current.QueueID != monitor.QueueID {
		return nil, false, service.requireHeartbeatCheckCommit(ctx, current, params.CheckID)
	}
	heartbeat, err := service.repository.Last(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return nil, false, service.stopHeartbeatCheck(ctx, params, monitor)
	}
	if err != nil {
		return nil, false, service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot revalidate heartbeat history for monitor [%s]", monitor.ID))
	}
	monitor.PhoneOnline = current.PhoneOnline
	return heartbeat, true, nil
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
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot create event when phone monitor [%s] missed heartbeat", params.MonitorID.String())))
		return
	}

	if _, err = service.dispatcher.DispatchWithTimeout(ctx, event, heartbeatCheckInterval); err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), params.PhoneID)))
	}
}

func (service *HeartbeatService) handleFailedMonitor(ctx context.Context, heartbeat *entities.Heartbeat, params *HeartbeatMonitorParams) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	event, err := service.createPhoneHeartbeatOfflineEvent(params.Source, params.CheckID, &events.PhoneHeartbeatOfflinePayload{
		PhoneID:                params.PhoneID,
		HeartbeatID:            heartbeat.ID,
		UserID:                 params.UserID,
		MonitorID:              params.MonitorID,
		LastHeartbeatTimestamp: heartbeat.Timestamp,
		Timestamp:              time.Now().UTC(),
		Owner:                  params.Owner,
	})
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot create event when phone monitor failed"))
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), params.PhoneID))
	}

	ctxLogger.Info(fmt.Sprintf("heartbeat monitor with id [%s] and phone id [%s] failed for user [%s]", params.MonitorID, params.PhoneID, params.UserID))
	return nil
}

func (service *HeartbeatService) scheduleHeartbeatCheck(ctx context.Context, lastTimestamp time.Time, params *HeartbeatMonitorParams, monitor *entities.HeartbeatMonitor, restart bool) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	checkID := uuid.NewString()
	event, err := service.createPhoneHeartbeatCheckEvent(params, lastTimestamp, checkID)
	if err != nil {
		if restart {
			service.releaseHeartbeatCheck(ctx, monitor.ID, monitor.QueueID, "")
		}
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot create event when phone monitor failed"))
	}

	incomingID := params.CheckID
	if !restart && incomingID == "" {
		incomingID = legacyHeartbeatCheckID
	}
	pendingID := "pending:" + incomingID + ":" + checkID
	claimed, err := service.monitorRepository.CompareAndSwapQueueID(ctx, monitor.ID, monitor.QueueID, pendingID)
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot claim heartbeat check for monitor [%s]", monitor.ID))
	}
	if !claimed {
		return service.verifyHeartbeatCheckCommit(ctx, params)
	}
	return service.enqueueHeartbeatCheck(ctx, event, monitor, pendingID, checkID, restart)
}

func (service *HeartbeatService) enqueueHeartbeatCheck(ctx context.Context, event cloudevents.Event, monitor *entities.HeartbeatMonitor, pendingID, checkID string, restart bool) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	queueID, err := service.dispatcher.DispatchWithTimeout(ctx, event, heartbeatCheckInterval)
	if err != nil {
		releaseID := monitor.QueueID
		if restart {
			releaseID = ""
		}
		service.releaseHeartbeatCheck(ctx, monitor.ID, pendingID, releaseID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot dispatch event [%s] for heartbeat monitor with phone id [%s]", event.Type(), monitor.PhoneID))
	}

	if err := service.finalizeHeartbeatCheck(ctx, monitor.ID, pendingID, checkID); err != nil {
		return err
	}
	ctxLogger.Info(fmt.Sprintf("heartbeat check scheduled for monitor with id [%s] and phone id [%s] and queue id [%s] for user [%s]", monitor.ID, monitor.PhoneID, queueID, monitor.UserID))

	return nil
}

func (service *HeartbeatService) finalizeHeartbeatCheck(ctx context.Context, monitorID uuid.UUID, pendingID, checkID string) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	committed, err := service.monitorRepository.CompareAndSwapQueueID(ctx, monitorID, pendingID, "check:"+checkID)
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot finalize queued heartbeat check for monitor [%s]", monitorID))
	}
	if !committed {
		return service.tracer.WrapErrorSpan(span, stacktrace.NewErrorf("heartbeat finalization ownership changed for monitor [%s]; retry delivery", monitorID))
	}
	return nil
}

func (service *HeartbeatService) releaseHeartbeatCheck(ctx context.Context, monitorID uuid.UUID, ownedID, retryID string) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if _, err := service.monitorRepository.CompareAndSwapQueueID(ctx, monitorID, ownedID, retryID); err != nil {
		ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot release heartbeat check claim for monitor [%s]", monitorID)))
	}
}

func (service *HeartbeatService) verifyHeartbeatCheckCommit(ctx context.Context, params *HeartbeatMonitorParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	monitor, err := service.monitorRepository.Load(ctx, params.UserID, params.Owner)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return nil
	}
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot verify successor for heartbeat monitor [%s]", params.MonitorID))
	}
	return service.requireHeartbeatCheckCommit(ctx, monitor, params.CheckID)
}

func (service *HeartbeatService) requireHeartbeatCheckCommit(ctx context.Context, monitor *entities.HeartbeatMonitor, incomingCheckID string) error {
	_, span := service.tracer.Start(ctx)
	defer span.End()

	if !strings.HasPrefix(monitor.QueueID, "check:") || strings.TrimPrefix(monitor.QueueID, "check:") == incomingCheckID {
		return service.tracer.WrapErrorSpan(span, stacktrace.NewErrorf("heartbeat successor for monitor [%s] is not committed; retry delivery", monitor.ID))
	}
	return nil
}

func (service *HeartbeatService) createPhoneHeartbeatMissedEvent(source string, payload *events.PhoneHeartbeatMissedPayload) (cloudevents.Event, error) {
	return service.createEvent(events.PhoneHeartbeatMissed, source, payload)
}

func (service *HeartbeatService) createPhoneHeartbeatOfflineEvent(source, checkID string, payload *events.PhoneHeartbeatOfflinePayload) (cloudevents.Event, error) {
	event, err := service.createEvent(events.EventTypePhoneHeartbeatOffline, source, payload)
	if err != nil {
		return event, err
	}
	// Queue-level deduplication survives retries before the offline listener updates phone status.
	event.SetID(uuid.NewSHA1(payload.MonitorID, []byte(checkID+":"+payload.HeartbeatID.String()+":"+payload.LastHeartbeatTimestamp.UTC().Format(time.RFC3339Nano))).String())
	return event, nil
}

func (service *HeartbeatService) createPhoneHeartbeatCheckEvent(params *HeartbeatMonitorParams, lastTimestamp time.Time, checkID string) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypePhoneHeartbeatCheck, params.Source, &events.PhoneHeartbeatCheckPayload{
		PhoneID:     params.PhoneID,
		UserID:      params.UserID,
		MonitorID:   params.MonitorID,
		ScheduledAt: lastTimestamp.Add(heartbeatCheckInterval),
		Owner:       params.Owner,
		CheckID:     checkID,
	})
}
