package repositories

import (
	"context"
	"strings"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMessageStatusUpdatesPreserveConcurrentSendAttemptCounts(t *testing.T) {
	pool := &messageThreadTestConnPool{}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool, WithoutReturning: true}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	logger := &messageThreadTestLogger{}
	repository := NewGormMessageRepository(logger, telemetry.NewOtelLogger("test", logger), db)

	err = repository.Update(context.Background(), &entities.Message{
		ID:               uuid.New(),
		UserID:           entities.UserID("user-id"),
		Status:           entities.MessageStatusScheduled,
		SendAttemptCount: 0,
	})

	require.NoError(t, err)
	require.NotEmpty(t, pool.statements)
	update := pool.statements[len(pool.statements)-1]
	assert.True(t, strings.HasPrefix(update.query, `UPDATE "messages"`))
	assert.Contains(t, update.query, `"status"=`)
	assert.NotContains(t, update.query, `"send_attempt_count"=`, "a stale status snapshot must not reset an atomic attempt increment")
}

func TestMessageSendAttemptIncrementOnlyUpdatesItsOwnedColumn(t *testing.T) {
	pool := &messageThreadTestConnPool{}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool, WithoutReturning: true}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	logger := &messageThreadTestLogger{}
	repository := NewGormMessageRepository(logger, telemetry.NewOtelLogger("test", logger), db)

	err = repository.IncrementSendAttemptCount(context.Background(), entities.UserID("user-id"), uuid.New())

	require.NoError(t, err)
	require.Len(t, pool.statements, 1)
	update := pool.statements[0]
	assert.Contains(t, update.query, `"send_attempt_count"=send_attempt_count +`)
	assert.NotContains(t, update.query, `"status"=`)
	assert.Contains(t, update.query, "WHERE user_id =")
	assert.Contains(t, update.query, "AND id =")
}
