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
