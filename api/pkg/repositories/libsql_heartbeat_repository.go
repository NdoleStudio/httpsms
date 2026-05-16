package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// libsqlHeartbeatRepository is responsible for persisting entities.Heartbeat in Turso/libSQL
type libsqlHeartbeatRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *sql.DB
}

// NewLibsqlHeartbeatRepository creates the libSQL version of the HeartbeatRepository
func NewLibsqlHeartbeatRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *sql.DB,
) HeartbeatRepository {
	return &libsqlHeartbeatRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &libsqlHeartbeatRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *libsqlHeartbeatRepository) Store(ctx context.Context, heartbeat *entities.Heartbeat) error {
	ctx, span, ctxLogger := repository.tracer.StartWithLogger(ctx, repository.logger)
	defer span.End()

	ctxLogger.Trace("saving new heartbeat")

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.db.ExecContext(ctx,
		"INSERT INTO "+tableHeartbeats+" (id, owner, version, charging, user_id, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
		heartbeat.ID.String(),
		heartbeat.Owner,
		heartbeat.Version,
		boolToInt(heartbeat.Charging),
		string(heartbeat.UserID),
		heartbeat.Timestamp.UTC(),
	)
	if err != nil {
		msg := fmt.Sprintf("cannot save heartbeat with ID [%s]", heartbeat.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *libsqlHeartbeatRepository) Index(ctx context.Context, userID entities.UserID, owner string, params IndexParams) (*[]entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	var rows *sql.Rows
	var err error

	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		rows, err = repository.db.QueryContext(ctx,
			"SELECT id, owner, version, charging, user_id, timestamp FROM "+tableHeartbeats+" WHERE user_id = ? AND owner = ? AND version LIKE ? ORDER BY timestamp DESC LIMIT ? OFFSET ?",
			string(userID), owner, queryPattern, params.Limit, params.Skip,
		)
	} else {
		rows, err = repository.db.QueryContext(ctx,
			"SELECT id, owner, version, charging, user_id, timestamp FROM "+tableHeartbeats+" WHERE user_id = ? AND owner = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?",
			string(userID), owner, params.Limit, params.Skip,
		)
	}
	if err != nil {
		msg := fmt.Sprintf("cannot fetch heartbeats with owner [%s] and params [%+#v]", owner, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	defer rows.Close()

	heartbeats := make([]entities.Heartbeat, 0)
	for rows.Next() {
		heartbeat, scanErr := scanHeartbeat(rows)
		if scanErr != nil {
			msg := fmt.Sprintf("cannot scan heartbeat row for owner [%s]", owner)
			return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(scanErr, msg))
		}
		heartbeats = append(heartbeats, *heartbeat)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		msg := fmt.Sprintf("error iterating heartbeat rows for owner [%s]", owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(rowsErr, msg))
	}

	return &heartbeats, nil
}

func (repository *libsqlHeartbeatRepository) Last(ctx context.Context, userID entities.UserID, owner string) (*entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	row := repository.db.QueryRowContext(ctx,
		"SELECT id, owner, version, charging, user_id, timestamp FROM "+tableHeartbeats+" WHERE user_id = ? AND owner = ? ORDER BY timestamp DESC LIMIT 1",
		string(userID), owner,
	)

	heartbeat, err := scanHeartbeatRow(row)
	if err == sql.ErrNoRows {
		msg := fmt.Sprintf("heartbeat with userID [%s] and owner [%s] does not exist", userID, owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}
	if err != nil {
		msg := fmt.Sprintf("cannot load heartbeat with userID [%s] and owner [%s]", userID, owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return heartbeat, nil
}

func (repository *libsqlHeartbeatRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.db.ExecContext(ctx, "DELETE FROM "+tableHeartbeats+" WHERE user_id = ?", string(userID))
	if err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.Heartbeat{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func scanHeartbeat(rows *sql.Rows) (*entities.Heartbeat, error) {
	heartbeat := new(entities.Heartbeat)
	var id string
	var charging int
	var userID string
	err := rows.Scan(&id, &heartbeat.Owner, &heartbeat.Version, &charging, &userID, &heartbeat.Timestamp)
	if err != nil {
		return nil, err
	}
	heartbeat.ID, err = uuid.Parse(id)
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot parse heartbeat ID [%s]", id))
	}
	heartbeat.Charging = charging != 0
	heartbeat.UserID = entities.UserID(userID)
	return heartbeat, nil
}

func scanHeartbeatRow(row *sql.Row) (*entities.Heartbeat, error) {
	heartbeat := new(entities.Heartbeat)
	var id string
	var charging int
	var userID string
	err := row.Scan(&id, &heartbeat.Owner, &heartbeat.Version, &charging, &userID, &heartbeat.Timestamp)
	if err != nil {
		return nil, err
	}
	heartbeat.ID, err = uuid.Parse(id)
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot parse heartbeat ID [%s]", id))
	}
	heartbeat.Charging = charging != 0
	heartbeat.UserID = entities.UserID(userID)
	return heartbeat, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
