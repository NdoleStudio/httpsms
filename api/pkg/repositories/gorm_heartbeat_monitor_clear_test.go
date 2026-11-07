package repositories

import (
	"context"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestHeartbeatMonitorClearQueueID_MatchesObservedTask(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: &messageThreadTestConnPool{}}), &gorm.Config{
		DisableAutomaticPing: true, SkipDefaultTransaction: true, DryRun: true,
	})
	require.NoError(t, err)
	var query string
	var args []any
	require.NoError(t, db.Callback().Update().After("gorm:update").Register("capture", func(db *gorm.DB) {
		query = db.Statement.SQL.String()
		args = db.Statement.Vars
	}))
	logger := &messageThreadTestLogger{}
	repository := NewGormHeartbeatMonitorRepository(logger, telemetry.NewOtelLogger("test", logger), db)
	monitorID := uuid.New()

	require.NoError(t, repository.ClearQueueID(context.Background(), monitorID, "old-check"))

	assert.Contains(t, query, "WHERE id = $3 AND queue_id = $4")
	require.Len(t, args, 4)
	assert.Equal(t, "", args[0])
	assert.Equal(t, monitorID, args[2])
	assert.Equal(t, "old-check", args[3])
}
