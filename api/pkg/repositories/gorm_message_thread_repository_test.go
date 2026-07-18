package repositories

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type messageThreadTestStatement struct {
	query string
	args  []any
}

type messageThreadTestConnPool struct {
	statements []messageThreadTestStatement
}

func (messageThreadTestConnPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("unexpected PrepareContext")
}

func (pool *messageThreadTestConnPool) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	pool.statements = append(pool.statements, messageThreadTestStatement{
		query: query,
		args:  append([]any(nil), args...),
	})
	return driver.RowsAffected(1), nil
}

func (messageThreadTestConnPool) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unexpected QueryContext")
}

func (messageThreadTestConnPool) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func (pool *messageThreadTestConnPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	return pool, nil
}

func (*messageThreadTestConnPool) Commit() error {
	return nil
}

func (*messageThreadTestConnPool) Rollback() error {
	return nil
}

type messageThreadTestLogger struct{}

func (logger *messageThreadTestLogger) Error(error)                         {}
func (logger *messageThreadTestLogger) WithService(string) telemetry.Logger { return logger }

func (logger *messageThreadTestLogger) WithString(string, string) telemetry.Logger { return logger }

func (logger *messageThreadTestLogger) WithSpan(trace.SpanContext) telemetry.Logger { return logger }
func (logger *messageThreadTestLogger) Trace(string)                                {}
func (logger *messageThreadTestLogger) Info(string)                                 {}
func (logger *messageThreadTestLogger) Warn(error)                                  {}
func (logger *messageThreadTestLogger) Debug(string)                                {}
func (logger *messageThreadTestLogger) Fatal(error)                                 {}
func (logger *messageThreadTestLogger) Printf(string, ...interface{})               {}

func TestMessageThreadStorePreservesExplicitUnreadState(t *testing.T) {
	pool := &messageThreadTestConnPool{}
	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn:             pool,
			WithoutReturning: true,
		}),
		&gorm.Config{DisableAutomaticPing: true},
	)
	require.NoError(t, err)

	logger := &messageThreadTestLogger{}
	repository := NewGormMessageThreadRepository(logger, telemetry.NewOtelLogger("test", logger), db)
	thread := &entities.MessageThread{
		ID:     uuid.New(),
		IsRead: false,
	}

	require.NoError(t, repository.Store(context.Background(), thread))
	assert.False(t, thread.IsRead)

	require.NotEmpty(t, pool.statements)
	update := pool.statements[len(pool.statements)-1]
	assert.True(t, strings.HasPrefix(update.query, `UPDATE "message_threads"`))
	assert.Contains(t, update.query, `"is_read"=$1`)
	assert.Contains(t, update.args, false)
}

func TestMessageThreadActivityUpdatesOwnOnlyMessageColumns(t *testing.T) {
	messageID := uuid.New()
	updates := messageThreadActivityUpdates(MessageThreadActivityUpdate{
		Timestamp: time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC),
		MessageID: messageID,
		Content:   "hello",
		Status:    entities.MessageStatusReceived,
	})

	assert.Equal(t, map[string]any{
		"order_timestamp":      time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC),
		"last_message_id":      messageID,
		"last_message_content": "hello",
		"status":               entities.MessageStatus(entities.MessageStatusReceived),
	}, updates)
	assert.NotContains(t, updates, "is_read")
	assert.NotContains(t, updates, "is_archived")
	assert.NotContains(t, updates, "last_read_at")
}

func TestMessageThreadDeletedUpdatesPreserveStatusType(t *testing.T) {
	messageID := uuid.New()
	content := "previous message"
	updates := messageThreadDeletedUpdates(MessageThreadDeletedUpdate{
		LastMessageID:      &messageID,
		LastMessageContent: &content,
		LastMessageStatus:  entities.MessageStatusDelivered,
	})

	assert.Equal(t, map[string]any{
		"last_message_id":      &messageID,
		"last_message_content": &content,
		"status":               entities.MessageStatus(entities.MessageStatusDelivered),
	}, updates)
}

func TestMessageThreadStatusUpdatesReadOnly(t *testing.T) {
	isRead := true
	readAt := time.Date(2026, 7, 18, 7, 1, 0, 0, time.UTC)

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		IsRead: &isRead,
		ReadAt: readAt,
	})

	assert.Equal(t, map[string]any{
		"is_read":      true,
		"last_read_at": readAt,
	}, updates)
	assert.NotContains(t, updates, "is_archived")
}

func TestMessageThreadStatusUpdatesArchiveOnly(t *testing.T) {
	isArchived := true

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		IsArchived: &isArchived,
	})

	assert.Equal(t, map[string]any{"is_archived": true}, updates)
	assert.NotContains(t, updates, "is_read")
	assert.NotContains(t, updates, "last_read_at")
}
