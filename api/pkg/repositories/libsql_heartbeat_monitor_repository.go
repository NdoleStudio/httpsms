package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// libsqlHeartbeatMonitorRepository is responsible for persisting entities.HeartbeatMonitor in Turso/libSQL
type libsqlHeartbeatMonitorRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *sql.DB
}

// NewLibsqlHeartbeatMonitorRepository creates the libSQL version of the HeartbeatMonitorRepository
func NewLibsqlHeartbeatMonitorRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *sql.DB,
) HeartbeatMonitorRepository {
	return &libsqlHeartbeatMonitorRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &libsqlHeartbeatMonitorRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *libsqlHeartbeatMonitorRepository) Store(ctx context.Context, monitor *entities.HeartbeatMonitor) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.db.ExecContext(ctx,
		"INSERT INTO "+tableHeartbeatMonitors+" (id, phone_id, user_id, queue_id, owner, phone_online, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		monitor.ID.String(),
		monitor.PhoneID.String(),
		string(monitor.UserID),
		monitor.QueueID,
		monitor.Owner,
		boolToInt(monitor.PhoneOnline),
		monitor.CreatedAt.UTC(),
		monitor.UpdatedAt.UTC(),
	)
	if err != nil {
		msg := fmt.Sprintf("cannot save heartbeat monitor with ID [%s]", monitor.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *libsqlHeartbeatMonitorRepository) Load(ctx context.Context, userID entities.UserID, phoneNumber string) (*entities.HeartbeatMonitor, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	row := repository.db.QueryRowContext(ctx,
		"SELECT id, phone_id, user_id, queue_id, owner, phone_online, created_at, updated_at FROM "+tableHeartbeatMonitors+" WHERE user_id = ? AND owner = ? LIMIT 1",
		string(userID), phoneNumber,
	)

	monitor, err := repository.scanHeartbeatMonitorRow(row)
	if err == sql.ErrNoRows {
		msg := fmt.Sprintf("heartbeat monitor with userID [%s] and owner [%s] does not exist", userID, phoneNumber)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}
	if err != nil {
		msg := fmt.Sprintf("cannot load heartbeat monitor with userID [%s] and owner [%s]", userID, phoneNumber)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return monitor, nil
}

func (repository *libsqlHeartbeatMonitorRepository) Exists(ctx context.Context, userID entities.UserID, monitorID uuid.UUID) (bool, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	var count int
	err := repository.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM "+tableHeartbeatMonitors+" WHERE user_id = ? AND id = ?",
		string(userID), monitorID.String(),
	).Scan(&count)
	if err != nil {
		msg := fmt.Sprintf("cannot check if heartbeat monitor exists with userID [%s] and monitor ID [%s]", userID, monitorID)
		return false, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return count > 0, nil
}

func (repository *libsqlHeartbeatMonitorRepository) UpdateQueueID(ctx context.Context, monitorID uuid.UUID, queueID string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.db.ExecContext(ctx,
		"UPDATE "+tableHeartbeatMonitors+" SET queue_id = ?, updated_at = ? WHERE id = ?",
		queueID, time.Now().UTC(), monitorID.String(),
	)
	if err != nil {
		msg := fmt.Sprintf("cannot update heartbeat monitor ID [%s]", monitorID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *libsqlHeartbeatMonitorRepository) Delete(ctx context.Context, userID entities.UserID, phoneNumber string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.db.ExecContext(ctx,
		"DELETE FROM "+tableHeartbeatMonitors+" WHERE user_id = ? AND owner = ?",
		string(userID), phoneNumber,
	)
	if err != nil {
		msg := fmt.Sprintf("cannot delete heartbeat monitor with owner [%s] and userID [%s]", phoneNumber, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *libsqlHeartbeatMonitorRepository) UpdatePhoneOnline(ctx context.Context, userID entities.UserID, monitorID uuid.UUID, online bool) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.db.ExecContext(ctx,
		"UPDATE "+tableHeartbeatMonitors+" SET phone_online = ?, updated_at = ? WHERE id = ? AND user_id = ?",
		boolToInt(online), time.Now().UTC(), monitorID.String(), string(userID),
	)
	if err != nil {
		msg := fmt.Sprintf("cannot update heartbeat monitor ID [%s] for user [%s]", monitorID, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *libsqlHeartbeatMonitorRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.db.ExecContext(ctx, "DELETE FROM "+tableHeartbeatMonitors+" WHERE user_id = ?", string(userID))
	if err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.HeartbeatMonitor{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *libsqlHeartbeatMonitorRepository) scanHeartbeatMonitorRow(row *sql.Row) (*entities.HeartbeatMonitor, error) {
	monitor := new(entities.HeartbeatMonitor)
	var id, phoneID, userID string
	var phoneOnline int
	err := row.Scan(&id, &phoneID, &userID, &monitor.QueueID, &monitor.Owner, &phoneOnline, &monitor.CreatedAt, &monitor.UpdatedAt)
	if err != nil {
		return nil, err
	}
	monitor.ID, err = uuid.Parse(id)
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot parse heartbeat monitor ID [%s]", id))
	}
	monitor.PhoneID, err = uuid.Parse(phoneID)
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot parse heartbeat monitor phone ID [%s]", phoneID))
	}
	monitor.UserID = entities.UserID(userID)
	monitor.PhoneOnline = phoneOnline != 0
	return monitor, nil
}
