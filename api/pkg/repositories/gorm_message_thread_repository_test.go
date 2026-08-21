package repositories

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
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
	statements   []messageThreadTestStatement
	thread       *entities.MessageThread
	rowsAffected func(query string) int64
	queryDB      *sql.DB
	begins       int
	commits      int
	rollbacks    int
}

func (messageThreadTestConnPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("unexpected PrepareContext")
}

func (pool *messageThreadTestConnPool) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	pool.statements = append(pool.statements, messageThreadTestStatement{
		query: query,
		args:  append([]any(nil), args...),
	})
	if pool.rowsAffected != nil {
		return driver.RowsAffected(pool.rowsAffected(query)), nil
	}
	return driver.RowsAffected(1), nil
}

func (pool *messageThreadTestConnPool) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	pool.statements = append(pool.statements, messageThreadTestStatement{
		query: query,
		args:  append([]any(nil), args...),
	})
	return pool.queryDB.QueryContext(ctx, query, args...)
}

func (messageThreadTestConnPool) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func (pool *messageThreadTestConnPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	pool.begins++
	return &messageThreadTestTxPool{pool: pool}, nil
}

type messageThreadTestTxPool struct {
	pool *messageThreadTestConnPool
}

func (tx *messageThreadTestTxPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return tx.pool.PrepareContext(ctx, query)
}

func (tx *messageThreadTestTxPool) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.pool.ExecContext(ctx, query, args...)
}

func (tx *messageThreadTestTxPool) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.pool.QueryContext(ctx, query, args...)
}

func (tx *messageThreadTestTxPool) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.pool.QueryRowContext(ctx, query, args...)
}

func (tx *messageThreadTestTxPool) Commit() error {
	tx.pool.commits++
	return nil
}

func (tx *messageThreadTestTxPool) Rollback() error {
	tx.pool.rollbacks++
	return nil
}

type messageThreadRowsConnector struct {
	pool *messageThreadTestConnPool
}

func (connector *messageThreadRowsConnector) Connect(context.Context) (driver.Conn, error) {
	return &messageThreadRowsConn{pool: connector.pool}, nil
}

func (*messageThreadRowsConnector) Driver() driver.Driver {
	return messageThreadRowsDriver{}
}

type messageThreadRowsDriver struct{}

func (messageThreadRowsDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("message thread test driver requires a connector")
}

type messageThreadRowsConn struct {
	pool *messageThreadTestConnPool
}

func (*messageThreadRowsConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected Prepare")
}

func (*messageThreadRowsConn) Close() error {
	return nil
}

func (*messageThreadRowsConn) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected Begin")
}

func (conn *messageThreadRowsConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	rows := &messageThreadDriverRows{
		columns: []string{"id", "user_id", "last_read_at", "unread_count"},
	}
	if conn.pool.thread != nil {
		rows.values = []driver.Value{
			conn.pool.thread.ID.String(),
			string(conn.pool.thread.UserID),
			conn.pool.thread.LastReadAt,
			int64(conn.pool.thread.UnreadCount),
		}
	}
	return rows, nil
}

type messageThreadDriverRows struct {
	columns []string
	values  []driver.Value
	read    bool
}

func (rows *messageThreadDriverRows) Columns() []string {
	return rows.columns
}

func (*messageThreadDriverRows) Close() error {
	return nil
}

func (rows *messageThreadDriverRows) Next(dest []driver.Value) error {
	if rows.read || rows.values == nil {
		return io.EOF
	}
	copy(dest, rows.values)
	rows.read = true
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

func newMessageThreadTestRepository(t *testing.T, pool *messageThreadTestConnPool) MessageThreadRepository {
	t.Helper()

	pool.queryDB = sql.OpenDB(&messageThreadRowsConnector{pool: pool})
	t.Cleanup(func() {
		require.NoError(t, pool.queryDB.Close())
	})

	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn:             pool,
			WithoutReturning: true,
		}),
		&gorm.Config{DisableAutomaticPing: true},
	)
	require.NoError(t, err)

	logger := &messageThreadTestLogger{}
	return NewGormMessageThreadRepository(logger, telemetry.NewOtelLogger("test", logger), db)
}

func messageThreadStatementIndex(pool *messageThreadTestConnPool, fragment string) int {
	for index, statement := range pool.statements {
		if strings.Contains(statement.query, fragment) {
			return index
		}
	}
	return -1
}

func TestMessageThreadUnreadStoreCreatesInitialLedgerItem(t *testing.T) {
	threadID := uuid.New()
	messageID := uuid.New()
	pool := &messageThreadTestConnPool{}
	repository := newMessageThreadTestRepository(t, pool)
	thread := &entities.MessageThread{
		ID:          threadID,
		UserID:      entities.UserID("user-id"),
		UnreadCount: 1,
	}

	require.NoError(t, repository.Store(context.Background(), thread, &messageID))
	require.Equal(t, 1, pool.begins)
	require.Equal(t, 1, pool.commits)
	require.Zero(t, pool.rollbacks)

	threadInsert := messageThreadStatementIndex(pool, `INSERT INTO "message_threads"`)
	ledgerInsert := messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`)
	require.NotEqual(t, -1, threadInsert)
	require.NotEqual(t, -1, ledgerInsert)
	assert.Less(t, threadInsert, ledgerInsert)
	assert.Contains(t, pool.statements[ledgerInsert].query, "ON CONFLICT DO NOTHING")
	assert.Contains(t, pool.statements[ledgerInsert].query, `"message_id"`)
	assert.Contains(t, pool.statements[ledgerInsert].query, `"message_thread_id"`)
}

func TestMessageThreadActivityUpdatesDoNotOwnUnreadColumns(t *testing.T) {
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
	assert.NotContains(t, updates, "last_read_at")
}

func TestMessageThreadActivityCountableItemInsertsLedgerAndIncrements(t *testing.T) {
	threadID := uuid.New()
	messageID := uuid.New()
	userID := entities.UserID("user-id")
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:         threadID,
			UserID:     userID,
			LastReadAt: time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC),
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: threadID,
		UserID:          userID,
		Timestamp:       time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC),
		MessageID:       messageID,
		Content:         "hello",
		Status:          entities.MessageStatusReceived,
		CountAsUnread:   true,
		EventTimestamp:  time.Date(2026, 7, 19, 10, 0, 1, 0, time.UTC),
	})

	require.NoError(t, err)
	require.Equal(t, 1, pool.begins)
	require.Equal(t, 1, pool.commits)
	require.Zero(t, pool.rollbacks)

	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	activity := messageThreadStatementIndex(pool, `"order_timestamp"`)
	ledger := messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`)
	increment := messageThreadStatementIndex(pool, "unread_count +")
	require.NotEqual(t, -1, lock)
	require.NotEqual(t, -1, activity)
	require.NotEqual(t, -1, ledger)
	require.NotEqual(t, -1, increment)
	assert.Less(t, lock, activity)
	assert.Less(t, activity, ledger)
	assert.Less(t, ledger, increment)
	assert.Contains(t, pool.statements[lock].query, "user_id =")
	assert.Contains(t, pool.statements[lock].query, "id =")
	assert.Contains(t, pool.statements[ledger].query, "ON CONFLICT DO NOTHING")
}

func TestMessageThreadActivityDuplicateLedgerItemDoesNotIncrement(t *testing.T) {
	threadID := uuid.New()
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:         threadID,
			UserID:     entities.UserID("user-id"),
			LastReadAt: time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC),
		},
		rowsAffected: func(query string) int64 {
			if strings.Contains(query, `INSERT INTO "message_thread_unread_items"`) {
				return 0
			}
			return 1
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: threadID,
		UserID:          entities.UserID("user-id"),
		Timestamp:       time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC),
		MessageID:       uuid.New(),
		Content:         "hello",
		Status:          entities.MessageStatusReceived,
		CountAsUnread:   true,
		EventTimestamp:  time.Date(2026, 7, 19, 10, 0, 1, 0, time.UTC),
	})

	require.NoError(t, err)
	assert.NotEqual(t, -1, messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, "unread_count +"))
}

func TestMessageThreadActivityAtWatermarkDoesNotCount(t *testing.T) {
	watermark := time.Date(2026, 7, 19, 10, 0, 1, 0, time.UTC)
	threadID := uuid.New()
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:         threadID,
			UserID:     entities.UserID("user-id"),
			LastReadAt: watermark,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: threadID,
		UserID:          entities.UserID("user-id"),
		Timestamp:       watermark,
		MessageID:       uuid.New(),
		Content:         "hello",
		Status:          entities.MessageStatusReceived,
		CountAsUnread:   true,
		EventTimestamp:  watermark,
	})

	require.NoError(t, err)
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, "unread_count +"))
}

func TestMessageThreadActivityMissingThreadReturnsScopedNotFound(t *testing.T) {
	threadID := uuid.New()
	userID := entities.UserID("user-id")
	pool := &messageThreadTestConnPool{}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: threadID,
		UserID:          userID,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCodeNotFound, stacktrace.GetCode(err))
	assert.Contains(t, err.Error(), threadID.String())
	assert.Contains(t, err.Error(), string(userID))
	require.Equal(t, 1, pool.begins)
	require.Zero(t, pool.commits)
	require.Equal(t, 1, pool.rollbacks)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	require.NotEqual(t, -1, lock)
	assert.Contains(t, pool.statements[lock].query, "user_id =")
	assert.Contains(t, pool.statements[lock].query, "id =")
}

func TestMessageThreadDeletedUpdatesPreserveStatusType(t *testing.T) {
	messageID := uuid.New()
	content := "previous message"
	updates := messageThreadDeletedUpdates(MessageThreadDeletedUpdate{
		LastMessageID:      &messageID,
		LastMessageContent: &content,
		LastMessageStatus:  entities.MessageStatusDelivered,
		UpdateLastMessage:  true,
	})

	assert.Equal(t, map[string]any{
		"last_message_id":      &messageID,
		"last_message_content": &content,
		"status":               entities.MessageStatus(entities.MessageStatusDelivered),
	}, updates)
}

func TestMessageThreadDeletedUpdatesSkipLastMessageWhenNotRequested(t *testing.T) {
	assert.Empty(t, messageThreadDeletedUpdates(MessageThreadDeletedUpdate{}))
}

func TestMessageThreadDeletedMessageDecrementsWhenLedgerDeleted(t *testing.T) {
	threadID := uuid.New()
	messageID := uuid.New()
	previousMessageID := uuid.New()
	previousContent := "previous"
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:          threadID,
			UserID:      entities.UserID("user-id"),
			UnreadCount: 1,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		MessageThreadID:    threadID,
		UserID:             entities.UserID("user-id"),
		DeletedMessageID:   messageID,
		UpdateLastMessage:  true,
		LastMessageID:      &previousMessageID,
		LastMessageContent: &previousContent,
		LastMessageStatus:  entities.MessageStatusDelivered,
	})

	require.NoError(t, err)
	require.Equal(t, 1, pool.begins)
	require.Equal(t, 1, pool.commits)
	require.Zero(t, pool.rollbacks)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	ledgerDelete := messageThreadStatementIndex(pool, `DELETE FROM "message_thread_unread_items"`)
	decrement := messageThreadStatementIndex(pool, "GREATEST(unread_count - 1, 0)")
	metadata := messageThreadStatementIndex(pool, `"last_message_id"`)
	require.NotEqual(t, -1, lock)
	require.NotEqual(t, -1, ledgerDelete)
	require.NotEqual(t, -1, decrement)
	require.NotEqual(t, -1, metadata)
	assert.Less(t, lock, ledgerDelete)
	assert.Less(t, ledgerDelete, decrement)
	assert.Less(t, decrement, metadata)
	assert.Contains(t, pool.statements[ledgerDelete].query, "message_id =")
	assert.Contains(t, pool.statements[ledgerDelete].query, "message_thread_id =")
}

func TestMessageThreadDeletedMessageWithoutLedgerDoesNotDecrement(t *testing.T) {
	threadID := uuid.New()
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:          threadID,
			UserID:      entities.UserID("user-id"),
			UnreadCount: 0,
		},
		rowsAffected: func(query string) int64 {
			if strings.Contains(query, `DELETE FROM "message_thread_unread_items"`) {
				return 0
			}
			return 1
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		MessageThreadID:  threadID,
		UserID:           entities.UserID("user-id"),
		DeletedMessageID: uuid.New(),
	})

	require.NoError(t, err)
	assert.NotEqual(t, -1, messageThreadStatementIndex(pool, `DELETE FROM "message_thread_unread_items"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, "GREATEST"))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `"last_message_id"`))
}

func TestMessageThreadStatusUpdatesResetUnreadCount(t *testing.T) {
	zero := uint(0)
	readAt := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		UnreadCount: &zero,
		ReadAt:      readAt,
	})

	assert.Equal(t, map[string]any{
		"unread_count": 0,
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
	assert.NotContains(t, updates, "unread_count")
	assert.NotContains(t, updates, "last_read_at")
}

func TestMessageThreadStatusResetDeletesLedgerRows(t *testing.T) {
	threadID := uuid.New()
	readAt := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	zero := uint(0)
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:          threadID,
			UserID:      entities.UserID("user-id"),
			UnreadCount: 2,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	thread, err := repository.UpdateStatus(
		context.Background(),
		entities.UserID("user-id"),
		threadID,
		MessageThreadStatusUpdate{
			UnreadCount: &zero,
			ReadAt:      readAt,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, thread)
	assert.Zero(t, thread.UnreadCount)
	assert.Equal(t, readAt, thread.LastReadAt)
	require.Equal(t, 1, pool.begins)
	require.Equal(t, 1, pool.commits)
	require.Zero(t, pool.rollbacks)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	statusUpdate := messageThreadStatementIndex(pool, `"unread_count"`)
	ledgerDelete := messageThreadStatementIndex(pool, `DELETE FROM "message_thread_unread_items"`)
	require.NotEqual(t, -1, lock)
	require.NotEqual(t, -1, statusUpdate)
	require.NotEqual(t, -1, ledgerDelete)
	assert.Less(t, lock, statusUpdate)
	assert.Less(t, statusUpdate, ledgerDelete)
	assert.Contains(t, pool.statements[statusUpdate].query, `"last_read_at"`)
	assert.Contains(t, pool.statements[ledgerDelete].query, "message_thread_id =")
}

func TestMessageThreadStatusRejectsNonzeroUnreadCount(t *testing.T) {
	threadID := uuid.New()
	one := uint(1)
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:     threadID,
			UserID: entities.UserID("user-id"),
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	thread, err := repository.UpdateStatus(
		context.Background(),
		entities.UserID("user-id"),
		threadID,
		MessageThreadStatusUpdate{UnreadCount: &one},
	)

	require.Error(t, err)
	assert.Nil(t, thread)
	assert.Contains(t, err.Error(), "unread count")
}
