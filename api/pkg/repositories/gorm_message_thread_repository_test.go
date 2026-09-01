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

func TestMessageThreadStoreIncrementsUnreadCountOnConflict(t *testing.T) {
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
		ID:          uuid.New(),
		UnreadCount: 1,
	}

	require.NoError(t, repository.Store(context.Background(), thread))
	assert.Equal(t, uint(1), thread.UnreadCount)

	require.NotEmpty(t, pool.statements)
	insert := pool.statements[len(pool.statements)-1]
	assert.True(t, strings.HasPrefix(insert.query, `INSERT INTO "message_threads"`))
	assert.Contains(t, insert.query, `ON CONFLICT ("user_id","owner","contact") DO UPDATE SET "unread_count"=message_threads.unread_count + $`)
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
	assert.NotContains(t, updates, "unread_count")
	assert.NotContains(t, updates, "is_archived")
}

func TestUpdateActivityMarksUnreadWithOneQuery(t *testing.T) {
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

	err = repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: uuid.New(),
		UserID:          entities.UserID("user-id"),
		Timestamp:       time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC),
		MessageID:       uuid.New(),
		Content:         "hello",
		Status:          entities.MessageStatusReceived,
		MarkAsUnread:    true,
		EventTimestamp:  time.Date(2026, 7, 19, 10, 0, 1, 0, time.UTC),
	})

	require.NoError(t, err)
	var updates []messageThreadTestStatement
	for _, statement := range pool.statements {
		if strings.HasPrefix(statement.query, `UPDATE "message_threads"`) {
			updates = append(updates, statement)
		}
	}
	require.Len(t, updates, 1)
	assert.Contains(t, updates[0].query, `"unread_count"=unread_count + $`)
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

func TestMessageThreadStatusUpdatesUnreadCountOnly(t *testing.T) {
	unreadCount := uint(0)

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		UnreadCount: &unreadCount,
	})

	assert.Equal(t, map[string]any{"unread_count": uint(0)}, updates)
	assert.NotContains(t, updates, "is_archived")
}

func TestMessageThreadStatusUpdatesArchiveOnly(t *testing.T) {
	isArchived := true

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		IsArchived: &isArchived,
	})
	assert.Equal(t, map[string]any{"is_archived": true}, updates)
	assert.NotContains(t, updates, "unread_count")
}
