package repositories

import (
	"context"
	"testing"

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
