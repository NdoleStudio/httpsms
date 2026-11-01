package repositories

import (
	"context"
	"database/sql"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestHeartbeatMonitorUpdatesCarryContext(t *testing.T) {
	updates := map[string]func(HeartbeatMonitorRepository, context.Context) error{
		"phone online": func(repository HeartbeatMonitorRepository, ctx context.Context) error {
			return repository.UpdatePhoneOnline(ctx, "user-id", uuid.New(), true)
		},
		"queue ID": func(repository HeartbeatMonitorRepository, ctx context.Context) error {
			return repository.UpdateQueueID(ctx, uuid.New(), "queue-id")
		},
		"conditional queue ID": func(repository HeartbeatMonitorRepository, ctx context.Context) error {
			_, err := repository.CompareAndSwapQueueID(ctx, uuid.New(), "old-check", "new-check")
			return err
		},
	}

	for name, update := range updates {
		t.Run(name, func(t *testing.T) {
			db, err := gorm.Open(postgres.New(postgres.Config{Conn: &messageThreadTestConnPool{}}), &gorm.Config{
				DisableAutomaticPing:   true,
				SkipDefaultTransaction: true,
				DryRun:                 true,
			})
			require.NoError(t, err)
			type contextKey struct{}
			ctx := context.WithValue(context.Background(), contextKey{}, "request")
			called := false
			require.NoError(t, db.Callback().Update().Before("gorm:update").Register("check_context", func(db *gorm.DB) {
				called = true
				require.Equal(t, "request", db.Statement.Context.Value(contextKey{}))
				_, hasDeadline := db.Statement.Context.Deadline()
				require.True(t, hasDeadline)
			}))
			logger := &messageThreadTestLogger{}
			repository := NewGormHeartbeatMonitorRepository(logger, telemetry.NewOtelLogger("test", logger), db)

			require.NoError(t, update(repository, ctx))
			require.True(t, called)
		})
	}
}

func TestHeartbeatMonitorCompareAndSwapQueueID_RequiresMatchingOwnership(t *testing.T) {
	for _, matched := range []bool{false, true} {
		t.Run(map[bool]string{false: "lost claim", true: "claimed"}[matched], func(t *testing.T) {
			db, err := gorm.Open(postgres.New(postgres.Config{Conn: &messageThreadTestConnPool{}}), &gorm.Config{
				DisableAutomaticPing: true, SkipDefaultTransaction: true, DryRun: true,
			})
			require.NoError(t, err)
			monitorID := uuid.New()
			require.NoError(t, db.Callback().Update().After("gorm:update").Register("check_ownership", func(db *gorm.DB) {
				require.Contains(t, db.Statement.SQL.String(), `WHERE id = $3 AND queue_id = $4`)
				require.Equal(t, monitorID, db.Statement.Vars[2])
				require.Equal(t, "old-check", db.Statement.Vars[3])
				if matched {
					db.RowsAffected = 1
				}
			}))
			logger := &messageThreadTestLogger{}
			repository := NewGormHeartbeatMonitorRepository(logger, telemetry.NewOtelLogger("test", logger), db)
			claimed, err := repository.CompareAndSwapQueueID(context.Background(), monitorID, "old-check", "new-check")
			require.NoError(t, err)
			require.Equal(t, matched, claimed)
		})
	}
}

type heartbeatVersionTestPool struct{ messageThreadTestConnPool }

func (pool *heartbeatVersionTestPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	return pool, nil
}

func (*heartbeatVersionTestPool) Commit() error   { return nil }
func (*heartbeatVersionTestPool) Rollback() error { return nil }

func TestHeartbeatMonitorRecordHeartbeat_PreservesPreviousStatusAndLease(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: &heartbeatVersionTestPool{}}), &gorm.Config{
		DisableAutomaticPing: true, DryRun: true,
	})
	require.NoError(t, err)
	monitorID, previousID, nextID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("existing_monitor", func(db *gorm.DB) {
		require.Contains(t, db.Statement.SQL.String(), "FOR UPDATE")
		require.Contains(t, db.Statement.SQL.String(), "last_heartbeat_id =")
		require.Contains(t, db.Statement.Vars, previousID)
		require.Contains(t, db.Statement.Vars, entities.UserID("user-id"))
		_, deadline := db.Statement.Context.Deadline()
		require.True(t, deadline)
		monitor := db.Statement.Dest.(*entities.HeartbeatMonitor)
		*monitor = entities.HeartbeatMonitor{ID: monitorID, LastHeartbeatID: previousID, PhoneOnline: false}
		db.RowsAffected = 1
	}))
	require.NoError(t, db.Callback().Update().After("gorm:update").Register("record_online", func(db *gorm.DB) {
		require.NotContains(t, db.Statement.SQL.String(), "updated_at")
		require.Contains(t, db.Statement.Vars, nextID)
		require.Contains(t, db.Statement.Vars, true)
		db.RowsAffected = 1
	}))
	logger := &messageThreadTestLogger{}
	repository := NewGormHeartbeatMonitorRepository(logger, telemetry.NewOtelLogger("test", logger), db)
	previous, err := repository.RecordHeartbeat(context.Background(), "user-id", monitorID, previousID, nextID)
	require.NoError(t, err)
	require.NotNil(t, previous)
	require.False(t, previous.PhoneOnline, "returning-phone notifications need the status before the atomic update")
	require.Equal(t, previousID, previous.LastHeartbeatID)
}

func TestHeartbeatMonitorOfflineVersion_RequiresMatchingHeartbeatWithoutRefreshingLease(t *testing.T) {
	for _, matched := range []bool{false, true} {
		t.Run(map[bool]string{false: "new heartbeat wins", true: "offline transition wins"}[matched], func(t *testing.T) {
			db, err := gorm.Open(postgres.New(postgres.Config{Conn: &messageThreadTestConnPool{}}), &gorm.Config{
				DisableAutomaticPing: true, SkipDefaultTransaction: true, DryRun: true,
			})
			require.NoError(t, err)
			monitorID, previousID, nextID := uuid.New(), uuid.New(), uuid.New()
			require.NoError(t, db.Callback().Update().After("gorm:update").Register("offline_version", func(db *gorm.DB) {
				require.Contains(t, db.Statement.SQL.String(), "last_heartbeat_id =")
				require.NotContains(t, db.Statement.SQL.String(), "updated_at")
				require.Equal(t, []any{nextID, false, monitorID, entities.UserID("user-id"), previousID}, db.Statement.Vars)
				if matched {
					db.RowsAffected = 1
				}
			}))
			logger := &messageThreadTestLogger{}
			repository := NewGormHeartbeatMonitorRepository(logger, telemetry.NewOtelLogger("test", logger), db)
			applied, err := repository.UpdatePhoneOfflineForHeartbeat(context.Background(), "user-id", monitorID, previousID, nextID)
			require.NoError(t, err)
			require.Equal(t, matched, applied)
		})
	}
}
