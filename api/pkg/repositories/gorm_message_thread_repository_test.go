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
	statements       []messageThreadTestStatement
	thread           *entities.MessageThread
	deletedMessageID *uuid.UUID
	rowsAffected     func(query string) int64
	execError        func(query string) error
	queryDB          *sql.DB
	begins           int
	commits          int
	rollbacks        int
}

func (messageThreadTestConnPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("unexpected PrepareContext")
}

func (pool *messageThreadTestConnPool) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	pool.statements = append(pool.statements, messageThreadTestStatement{
		query: query,
		args:  append([]any(nil), args...),
	})
	if pool.execError != nil {
		if err := pool.execError(query); err != nil {
			return nil, err
		}
	}
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

func (conn *messageThreadRowsConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, `"message_thread_deleted_items"`) {
		rows := &messageThreadDriverRows{
			columns: []string{"message_id"},
		}
		if conn.pool.deletedMessageID != nil {
			rows.values = []driver.Value{conn.pool.deletedMessageID.String()}
		}
		return rows, nil
	}

	rows := &messageThreadDriverRows{
		columns: []string{
			"id",
			"user_id",
			"owner",
			"contact",
			"is_archived",
			"last_read_at",
			"unread_count",
			"last_message_id",
			"last_message_content",
			"status",
			"order_timestamp",
		},
	}
	if conn.pool.thread != nil {
		var lastMessageID driver.Value
		if conn.pool.thread.LastMessageID != nil {
			lastMessageID = conn.pool.thread.LastMessageID.String()
		}
		var lastMessageContent driver.Value
		if conn.pool.thread.LastMessageContent != nil {
			lastMessageContent = *conn.pool.thread.LastMessageContent
		}
		rows.values = []driver.Value{
			conn.pool.thread.ID.String(),
			string(conn.pool.thread.UserID),
			conn.pool.thread.Owner,
			conn.pool.thread.Contact,
			conn.pool.thread.IsArchived,
			conn.pool.thread.LastReadAt,
			int64(conn.pool.thread.UnreadCount),
			lastMessageID,
			lastMessageContent,
			string(conn.pool.thread.Status),
			conn.pool.thread.OrderTimestamp,
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

type messageThreadRetryableError struct{}

func (messageThreadRetryableError) Error() string {
	return "restart transaction"
}

func (messageThreadRetryableError) SQLState() string {
	return "40001"
}

func newMessageThreadTestRepository(t *testing.T, pool *messageThreadTestConnPool) *gormMessageThreadRepository {
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
	repository, ok := NewGormMessageThreadRepository(logger, telemetry.NewOtelLogger("test", logger), db).(*gormMessageThreadRepository)
	require.True(t, ok)
	return repository
}

func messageThreadStatementIndex(pool *messageThreadTestConnPool, fragment string) int {
	for index, statement := range pool.statements {
		if strings.Contains(statement.query, fragment) {
			return index
		}
	}
	return -1
}

func messageThreadStatementCount(pool *messageThreadTestConnPool, fragment string) int {
	count := 0
	for _, statement := range pool.statements {
		if strings.Contains(statement.query, fragment) {
			count++
		}
	}
	return count
}

func messageThreadStatementIndexAfter(pool *messageThreadTestConnPool, fragment string, after int) int {
	for index := after + 1; index < len(pool.statements); index++ {
		if strings.Contains(pool.statements[index].query, fragment) {
			return index
		}
	}
	return -1
}

func TestMessageThreadMutationsRetrySerializationFailures(t *testing.T) {
	t.Run("store", func(t *testing.T) {
		attempts := 0
		pool := &messageThreadTestConnPool{
			execError: func(query string) error {
				if strings.Contains(query, `INSERT INTO "message_threads"`) {
					attempts++
					if attempts == 1 {
						return messageThreadRetryableError{}
					}
				}
				return nil
			},
		}
		repository := newMessageThreadTestRepository(t, pool)

		err := repository.Store(context.Background(), MessageThreadStoreParams{
			Thread: &entities.MessageThread{
				ID:      uuid.New(),
				UserID:  entities.UserID("user-id"),
				Owner:   "+18005550199",
				Contact: "+18005550100",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("activity", func(t *testing.T) {
		attempts := 0
		threadID := uuid.New()
		pool := &messageThreadTestConnPool{
			thread: &entities.MessageThread{ID: threadID, UserID: entities.UserID("user-id")},
			execError: func(query string) error {
				if strings.Contains(query, `UPDATE "message_threads"`) && strings.Contains(query, `"order_timestamp"`) {
					attempts++
					if attempts == 1 {
						return messageThreadRetryableError{}
					}
				}
				return nil
			},
		}
		repository := newMessageThreadTestRepository(t, pool)

		err := repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
			MessageThreadID: threadID,
			UserID:          entities.UserID("user-id"),
			Timestamp:       time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC),
			MessageID:       uuid.New(),
			Content:         "hello",
			Status:          entities.MessageStatusReceived,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("status", func(t *testing.T) {
		attempts := 0
		clockCalls := 0
		threadID := uuid.New()
		pool := &messageThreadTestConnPool{
			thread: &entities.MessageThread{ID: threadID, UserID: entities.UserID("user-id"), UnreadCount: 1},
			execError: func(query string) error {
				if strings.Contains(query, `UPDATE "message_threads"`) && strings.Contains(query, `"unread_count"`) {
					attempts++
					if attempts == 1 {
						return messageThreadRetryableError{}
					}
				}
				return nil
			},
		}
		repository := newMessageThreadTestRepository(t, pool)
		repository.now = func() time.Time {
			clockCalls++
			return time.Date(2026, 8, 21, 10, 0, clockCalls, 0, time.UTC)
		}
		zero := uint(0)

		thread, err := repository.UpdateStatus(
			context.Background(),
			entities.UserID("user-id"),
			threadID,
			MessageThreadStatusUpdate{UnreadCount: &zero},
		)

		require.NoError(t, err)
		require.NotNil(t, thread)
		assert.Equal(t, 2, attempts)
		assert.Equal(t, 2, clockCalls)
		assert.Equal(t, time.Date(2026, 8, 21, 10, 0, 2, 0, time.UTC), thread.LastReadAt)
	})

	t.Run("deleted message", func(t *testing.T) {
		attempts := 0
		threadID := uuid.New()
		pool := &messageThreadTestConnPool{
			thread: &entities.MessageThread{ID: threadID, UserID: entities.UserID("user-id"), UnreadCount: 1},
			execError: func(query string) error {
				if strings.Contains(query, `UPDATE "message_thread_unread_items"`) {
					attempts++
					if attempts == 1 {
						return messageThreadRetryableError{}
					}
				}
				return nil
			},
		}
		repository := newMessageThreadTestRepository(t, pool)

		err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
			UserID:           entities.UserID("user-id"),
			Owner:            "+18005550199",
			Contact:          "+18005550100",
			DeletedMessageID: uuid.New(),
		})

		require.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})
}

func TestMessageThreadUnreadStoreCreatesInitialLedgerItem(t *testing.T) {
	threadID := uuid.New()
	messageID := uuid.New()
	pool := &messageThreadTestConnPool{}
	repository := newMessageThreadTestRepository(t, pool)
	thread := &entities.MessageThread{
		ID:            threadID,
		UserID:        entities.UserID("user-id"),
		UnreadCount:   1,
		LastMessageID: &messageID,
	}

	require.NoError(t, repository.Store(context.Background(), MessageThreadStoreParams{
		Thread:        thread,
		CountAsUnread: true,
	}))
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

func TestMessageThreadStoreConflictAppliesLosingActivity(t *testing.T) {
	winnerThreadID := uuid.New()
	losingThreadID := uuid.New()
	messageID := uuid.New()
	userID := entities.UserID("user-id")
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:          winnerThreadID,
			UserID:      userID,
			UnreadCount: 1,
			LastReadAt:  time.Unix(0, 0).UTC(),
		},
		rowsAffected: func(query string) int64 {
			if strings.Contains(query, `INSERT INTO "message_threads"`) {
				return 0
			}
			return 1
		},
	}
	repository := newMessageThreadTestRepository(t, pool)
	content := "losing message"
	eventTimestamp := time.Date(2026, 8, 21, 10, 0, 1, 0, time.UTC)
	thread := &entities.MessageThread{
		ID:                 losingThreadID,
		UserID:             userID,
		Owner:              "+18005550199",
		Contact:            "+18005550100",
		UnreadCount:        1,
		LastReadAt:         time.Unix(0, 0).UTC(),
		LastMessageID:      &messageID,
		LastMessageContent: &content,
		Status:             entities.MessageStatusReceived,
		OrderTimestamp:     eventTimestamp,
	}

	err := repository.Store(context.Background(), MessageThreadStoreParams{
		Thread:         thread,
		CountAsUnread:  true,
		EventTimestamp: eventTimestamp,
	})

	require.NoError(t, err)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	activity := messageThreadStatementIndexAfter(pool, `"order_timestamp"`, lock)
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
	assert.Contains(t, pool.statements[lock].query, "owner =")
	assert.Contains(t, pool.statements[lock].query, "contact =")
	assert.Contains(t, pool.statements[ledger].args, winnerThreadID)
}

func TestMessageThreadStoreConflictStaleActivityPreservesWinnerMetadataAndCountsUnread(t *testing.T) {
	winnerThreadID := uuid.New()
	losingThreadID := uuid.New()
	winnerMessageID := uuid.New()
	losingMessageID := uuid.New()
	userID := entities.UserID("user-id")
	winnerContent := "winner"
	winnerTimestamp := time.Date(2026, 8, 21, 10, 0, 2, 0, time.UTC)
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:                 winnerThreadID,
			UserID:             userID,
			UnreadCount:        1,
			LastReadAt:         time.Unix(0, 0).UTC(),
			LastMessageID:      &winnerMessageID,
			LastMessageContent: &winnerContent,
			Status:             entities.MessageStatusDelivered,
			OrderTimestamp:     winnerTimestamp,
		},
		rowsAffected: func(query string) int64 {
			if strings.Contains(query, `INSERT INTO "message_threads"`) {
				return 0
			}
			return 1
		},
	}
	repository := newMessageThreadTestRepository(t, pool)
	losingContent := "loser"
	losingTimestamp := winnerTimestamp.Add(-time.Second)

	err := repository.Store(context.Background(), MessageThreadStoreParams{
		Thread: &entities.MessageThread{
			ID:                 losingThreadID,
			UserID:             userID,
			Owner:              "+18005550199",
			Contact:            "+18005550100",
			UnreadCount:        1,
			LastReadAt:         time.Unix(0, 0).UTC(),
			LastMessageID:      &losingMessageID,
			LastMessageContent: &losingContent,
			Status:             entities.MessageStatusReceived,
			OrderTimestamp:     losingTimestamp,
		},
		CountAsUnread:  true,
		EventTimestamp: losingTimestamp,
	})

	require.NoError(t, err)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	metadata := messageThreadStatementIndexAfter(pool, `"order_timestamp"`, lock)
	ledger := messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`)
	increment := messageThreadStatementIndex(pool, "unread_count +")
	require.NotEqual(t, -1, lock)
	assert.Equal(t, -1, metadata)
	require.NotEqual(t, -1, ledger)
	require.NotEqual(t, -1, increment)
	assert.Less(t, lock, ledger)
	assert.Less(t, ledger, increment)
	assert.Contains(t, pool.statements[ledger].args, losingMessageID)
	assert.Contains(t, pool.statements[ledger].args, winnerThreadID)
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

func TestMessageThreadStaleActivityPreservesPreviewAndCountsUnread(t *testing.T) {
	threadID := uuid.New()
	currentMessageID := uuid.New()
	incomingMessageID := uuid.New()
	userID := entities.UserID("user-id")
	content := "current"
	currentTimestamp := time.Date(2026, 8, 21, 10, 0, 2, 0, time.UTC)
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:                 threadID,
			UserID:             userID,
			LastReadAt:         time.Unix(0, 0).UTC(),
			LastMessageID:      &currentMessageID,
			LastMessageContent: &content,
			Status:             entities.MessageStatusDelivered,
			OrderTimestamp:     currentTimestamp,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: threadID,
		UserID:          userID,
		Timestamp:       currentTimestamp.Add(-time.Second),
		MessageID:       incomingMessageID,
		Content:         "stale",
		Status:          entities.MessageStatusReceived,
		CountAsUnread:   true,
		EventTimestamp:  currentTimestamp,
	})

	require.NoError(t, err)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	metadata := messageThreadStatementIndexAfter(pool, `"order_timestamp"`, lock)
	ledger := messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`)
	increment := messageThreadStatementIndex(pool, "unread_count +")
	require.NotEqual(t, -1, lock)
	assert.Equal(t, -1, metadata)
	require.NotEqual(t, -1, ledger)
	require.NotEqual(t, -1, increment)
	assert.Less(t, lock, ledger)
	assert.Less(t, ledger, increment)
}

func TestMessageThreadStaleActivityStillUnarchives(t *testing.T) {
	threadID := uuid.New()
	currentMessageID := uuid.New()
	content := "current"
	currentTimestamp := time.Date(2026, 8, 21, 10, 0, 2, 0, time.UTC)
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:                 threadID,
			UserID:             entities.UserID("user-id"),
			IsArchived:         true,
			LastMessageID:      &currentMessageID,
			LastMessageContent: &content,
			Status:             entities.MessageStatusDelivered,
			OrderTimestamp:     currentTimestamp,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: threadID,
		UserID:          entities.UserID("user-id"),
		Timestamp:       currentTimestamp.Add(-time.Second),
		MessageID:       uuid.New(),
		Content:         "stale",
		Status:          entities.MessageStatusReceived,
		Unarchive:       true,
	})

	require.NoError(t, err)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	unarchive := messageThreadStatementIndexAfter(pool, `"is_archived"`, lock)
	require.NotEqual(t, -1, lock)
	require.NotEqual(t, -1, unarchive)
	assert.NotContains(t, pool.statements[unarchive].query, `"order_timestamp"`)
	assert.NotContains(t, pool.statements[unarchive].query, `"last_message_id"`)
	assert.NotContains(t, pool.statements[unarchive].query, `"last_message_content"`)
	assert.NotContains(t, pool.statements[unarchive].query, `"status"`)
}

func TestMessageThreadDeliveredActivityDoesNotRegressSameMessage(t *testing.T) {
	threadID := uuid.New()
	messageID := uuid.New()
	content := "delivered"
	timestamp := time.Date(2026, 8, 21, 10, 0, 2, 0, time.UTC)
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:                 threadID,
			UserID:             entities.UserID("user-id"),
			LastMessageID:      &messageID,
			LastMessageContent: &content,
			Status:             entities.MessageStatusDelivered,
			OrderTimestamp:     timestamp,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: threadID,
		UserID:          entities.UserID("user-id"),
		Timestamp:       timestamp.Add(time.Second),
		MessageID:       messageID,
		Content:         "regressed",
		Status:          entities.MessageStatusSent,
	})

	require.NoError(t, err)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	require.NotEqual(t, -1, lock)
	assert.Equal(t, -1, messageThreadStatementIndexAfter(pool, `"order_timestamp"`, lock))
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
	status := entities.MessageStatus(entities.MessageStatusDelivered)
	updates, err := messageThreadDeletedUpdates(MessageThreadDeletedUpdate{
		LastMessageID:      &messageID,
		LastMessageContent: &content,
		LastMessageStatus:  &status,
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"last_message_id":      &messageID,
		"last_message_content": &content,
		"status":               entities.MessageStatus(entities.MessageStatusDelivered),
	}, updates)
}

func TestMessageThreadDeletedUpdatesRequirePreviousContent(t *testing.T) {
	status := entities.MessageStatus(entities.MessageStatusDelivered)
	updates, err := messageThreadDeletedUpdates(MessageThreadDeletedUpdate{
		DeletedMessageID:  uuid.New(),
		LastMessageStatus: &status,
	})

	require.Error(t, err)
	assert.Nil(t, updates)
	assert.Contains(t, err.Error(), "content")
}

func TestMessageThreadDeletedMessageDecrementsWhenLedgerDeleted(t *testing.T) {
	threadID := uuid.New()
	messageID := uuid.New()
	previousMessageID := uuid.New()
	previousContent := "previous"
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:            threadID,
			UserID:        entities.UserID("user-id"),
			UnreadCount:   1,
			LastMessageID: &messageID,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)
	previousStatus := entities.MessageStatus(entities.MessageStatusDelivered)

	err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:             entities.UserID("user-id"),
		Owner:              "+18005550199",
		Contact:            "+18005550100",
		DeletedMessageID:   messageID,
		LastMessageID:      &previousMessageID,
		LastMessageContent: &previousContent,
		LastMessageStatus:  &previousStatus,
	})

	require.NoError(t, err)
	require.Equal(t, 1, pool.begins)
	require.Equal(t, 1, pool.commits)
	require.Zero(t, pool.rollbacks)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	ledgerDelete := messageThreadStatementIndex(pool, `UPDATE "message_thread_unread_items"`)
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
	assert.Contains(t, pool.statements[ledgerDelete].query, "counted =")
}

func TestMessageThreadDeletedStaleReplacementPreservesNewerLastActivity(t *testing.T) {
	threadID := uuid.New()
	deletedMessageID := uuid.New()
	newerMessageID := uuid.New()
	previousMessageID := uuid.New()
	previousContent := "previous"
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:            threadID,
			UserID:        entities.UserID("user-id"),
			UnreadCount:   1,
			LastMessageID: &newerMessageID,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)
	previousStatus := entities.MessageStatus(entities.MessageStatusDelivered)

	err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:             entities.UserID("user-id"),
		Owner:              "+18005550199",
		Contact:            "+18005550100",
		DeletedMessageID:   deletedMessageID,
		LastMessageID:      &previousMessageID,
		LastMessageContent: &previousContent,
		LastMessageStatus:  &previousStatus,
	})

	require.NoError(t, err)
	tombstone := messageThreadStatementIndex(pool, `UPDATE "message_thread_unread_items"`)
	decrement := messageThreadStatementIndex(pool, "GREATEST(unread_count - 1, 0)")
	require.NotEqual(t, -1, tombstone)
	require.NotEqual(t, -1, decrement)
	assert.Less(t, tombstone, decrement)
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `"last_message_id"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `DELETE FROM "message_threads"`))
}

func TestMessageThreadDeletedStaleFinalMessagePreservesNewerLastActivity(t *testing.T) {
	threadID := uuid.New()
	deletedMessageID := uuid.New()
	newerMessageID := uuid.New()
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:            threadID,
			UserID:        entities.UserID("user-id"),
			UnreadCount:   1,
			LastMessageID: &newerMessageID,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:           entities.UserID("user-id"),
		Owner:            "+18005550199",
		Contact:          "+18005550100",
		DeletedMessageID: deletedMessageID,
	})

	require.NoError(t, err)
	tombstone := messageThreadStatementIndex(pool, `UPDATE "message_thread_unread_items"`)
	decrement := messageThreadStatementIndex(pool, "GREATEST(unread_count - 1, 0)")
	require.NotEqual(t, -1, tombstone)
	require.NotEqual(t, -1, decrement)
	assert.Less(t, tombstone, decrement)
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `"last_message_id"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `DELETE FROM "message_threads"`))
}

func TestMessageThreadDeletedCurrentFinalMessageDeletesThread(t *testing.T) {
	threadID := uuid.New()
	deletedMessageID := uuid.New()
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:            threadID,
			UserID:        entities.UserID("user-id"),
			UnreadCount:   1,
			LastMessageID: &deletedMessageID,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:           entities.UserID("user-id"),
		Owner:            "+18005550199",
		Contact:          "+18005550100",
		DeletedMessageID: deletedMessageID,
	})

	require.NoError(t, err)
	require.Equal(t, 1, pool.begins)
	require.Equal(t, 1, pool.commits)
	require.Zero(t, pool.rollbacks)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	tombstone := messageThreadStatementIndex(pool, `UPDATE "message_thread_unread_items"`)
	decrement := messageThreadStatementIndex(pool, "GREATEST(unread_count - 1, 0)")
	threadDelete := messageThreadStatementIndex(pool, `DELETE FROM "message_threads"`)
	require.NotEqual(t, -1, lock)
	require.NotEqual(t, -1, tombstone)
	require.NotEqual(t, -1, decrement)
	require.NotEqual(t, -1, threadDelete)
	assert.Less(t, lock, tombstone)
	assert.Less(t, tombstone, decrement)
	assert.Less(t, decrement, threadDelete)
}

func TestMessageThreadDeletedCurrentMessageRequiresPreviousStatus(t *testing.T) {
	threadID := uuid.New()
	deletedMessageID := uuid.New()
	previousMessageID := uuid.New()
	previousContent := "previous"
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:            threadID,
			UserID:        entities.UserID("user-id"),
			LastMessageID: &deletedMessageID,
		},
		rowsAffected: func(query string) int64 {
			if strings.Contains(query, `UPDATE "message_thread_unread_items"`) {
				return 0
			}
			return 1
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:             entities.UserID("user-id"),
		Owner:              "+18005550199",
		Contact:            "+18005550100",
		DeletedMessageID:   deletedMessageID,
		LastMessageID:      &previousMessageID,
		LastMessageContent: &previousContent,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status")
	require.Equal(t, 1, pool.rollbacks)
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `"last_message_id"`))
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
			if strings.Contains(query, `UPDATE "message_thread_unread_items"`) {
				return 0
			}
			return 1
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:           entities.UserID("user-id"),
		Owner:            "+18005550199",
		Contact:          "+18005550100",
		DeletedMessageID: uuid.New(),
	})

	require.NoError(t, err)
	assert.NotEqual(t, -1, messageThreadStatementIndex(pool, `UPDATE "message_thread_unread_items"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, "GREATEST"))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `"last_message_id"`))
}

func TestMessageThreadDeletedItemReplayDoesNotIncrement(t *testing.T) {
	threadID := uuid.New()
	messageID := uuid.New()
	userID := entities.UserID("user-id")
	tombstoned := false
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:          threadID,
			UserID:      userID,
			UnreadCount: 1,
			LastReadAt:  time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC),
		},
		rowsAffected: func(query string) int64 {
			switch {
			case strings.Contains(query, `UPDATE "message_thread_unread_items"`) &&
				strings.Contains(query, `"counted"`):
				if tombstoned {
					return 0
				}
				tombstoned = true
				return 1
			case strings.Contains(query, `INSERT INTO "message_thread_unread_items"`):
				if tombstoned {
					return 0
				}
				return 1
			default:
				return 1
			}
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	require.NoError(t, repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:           userID,
		Owner:            "+18005550199",
		Contact:          "+18005550100",
		DeletedMessageID: messageID,
	}))
	require.NoError(t, repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: threadID,
		UserID:          userID,
		Timestamp:       time.Date(2026, 8, 21, 10, 0, 1, 0, time.UTC),
		MessageID:       messageID,
		Content:         "replayed",
		Status:          entities.MessageStatusReceived,
		CountAsUnread:   true,
		EventTimestamp:  time.Date(2026, 8, 21, 10, 0, 1, 0, time.UTC),
	}))

	tombstone := messageThreadStatementIndex(pool, `UPDATE "message_thread_unread_items"`)
	replay := messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`)
	require.NotEqual(t, -1, tombstone)
	require.NotEqual(t, -1, replay)
	assert.Less(t, tombstone, replay)
	assert.Contains(t, pool.statements[tombstone].query, `"counted"`)
	assert.Zero(t, messageThreadStatementCount(pool, "unread_count +"))
	assert.Equal(t, 1, messageThreadStatementCount(pool, "GREATEST(unread_count - 1, 0)"))
}

func TestMessageThreadDeletionBeforeThreadBlocksLaterStore(t *testing.T) {
	messageID := uuid.New()
	threadID := uuid.New()
	userID := entities.UserID("user-id")
	pool := &messageThreadTestConnPool{}
	repository := newMessageThreadTestRepository(t, pool)

	err := repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:           userID,
		Owner:            "+18005550199",
		Contact:          "+18005550100",
		DeletedMessageID: messageID,
	})

	require.NoError(t, err)
	marker := messageThreadStatementIndex(pool, `INSERT INTO "message_thread_deleted_items"`)
	lock := messageThreadStatementIndex(pool, "FOR UPDATE")
	require.NotEqual(t, -1, marker)
	require.NotEqual(t, -1, lock)
	assert.Less(t, marker, lock)

	pool.deletedMessageID = &messageID
	pool.statements = nil
	content := "deleted inbound"
	err = repository.Store(context.Background(), MessageThreadStoreParams{
		Thread: &entities.MessageThread{
			ID:                 threadID,
			UserID:             userID,
			Owner:              "+18005550199",
			Contact:            "+18005550100",
			UnreadCount:        1,
			LastMessageID:      &messageID,
			LastMessageContent: &content,
			Status:             entities.MessageStatusReceived,
		},
		CountAsUnread: true,
	})

	require.NoError(t, err)
	assert.NotEqual(t, -1, messageThreadStatementIndex(pool, `FROM "message_thread_deleted_items"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `INSERT INTO "message_threads"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`))
}

func TestMessageThreadDeletedInboundReplayPreservesExistingThread(t *testing.T) {
	messageID := uuid.New()
	newerMessageID := uuid.New()
	threadID := uuid.New()
	userID := entities.UserID("user-id")
	currentContent := "newer preview"
	currentTimestamp := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:                 threadID,
			UserID:             userID,
			Owner:              "+18005550199",
			Contact:            "+18005550100",
			IsArchived:         true,
			LastMessageID:      &newerMessageID,
			LastMessageContent: &currentContent,
			OrderTimestamp:     currentTimestamp,
			LastReadAt:         time.Unix(0, 0).UTC(),
		},
		rowsAffected: func(query string) int64 {
			if strings.Contains(query, `UPDATE "message_thread_unread_items"`) {
				return 0
			}
			return 1
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	require.NoError(t, repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:           userID,
		Owner:            "+18005550199",
		Contact:          "+18005550100",
		DeletedMessageID: messageID,
	}))

	pool.deletedMessageID = &messageID
	pool.statements = nil
	require.NoError(t, repository.UpdateActivity(context.Background(), MessageThreadActivityUpdate{
		MessageThreadID: threadID,
		UserID:          userID,
		Timestamp:       currentTimestamp.Add(time.Second),
		MessageID:       messageID,
		Content:         "replayed deleted inbound",
		Status:          entities.MessageStatusReceived,
		CountAsUnread:   true,
		EventTimestamp:  currentTimestamp.Add(time.Second),
		Unarchive:       true,
	}))

	assert.NotEqual(t, -1, messageThreadStatementIndex(pool, `FROM "message_thread_deleted_items"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, "FOR UPDATE"))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `"order_timestamp"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `"is_archived"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, "unread_count +"))
}

func TestMessageThreadFinalDeletionBlocksStoreReplay(t *testing.T) {
	messageID := uuid.New()
	threadID := uuid.New()
	userID := entities.UserID("user-id")
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:            threadID,
			UserID:        userID,
			Owner:         "+18005550199",
			Contact:       "+18005550100",
			LastMessageID: &messageID,
		},
		rowsAffected: func(query string) int64 {
			if strings.Contains(query, `UPDATE "message_thread_unread_items"`) {
				return 0
			}
			return 1
		},
	}
	repository := newMessageThreadTestRepository(t, pool)

	require.NoError(t, repository.UpdateAfterDeletedMessage(context.Background(), MessageThreadDeletedUpdate{
		UserID:           userID,
		Owner:            "+18005550199",
		Contact:          "+18005550100",
		DeletedMessageID: messageID,
	}))
	assert.NotEqual(t, -1, messageThreadStatementIndex(pool, `DELETE FROM "message_threads"`))

	pool.thread = nil
	pool.deletedMessageID = &messageID
	pool.statements = nil
	content := "replayed final message"
	require.NoError(t, repository.Store(context.Background(), MessageThreadStoreParams{
		Thread: &entities.MessageThread{
			ID:                 uuid.New(),
			UserID:             userID,
			Owner:              "+18005550199",
			Contact:            "+18005550100",
			UnreadCount:        1,
			LastMessageID:      &messageID,
			LastMessageContent: &content,
			Status:             entities.MessageStatusReceived,
		},
		CountAsUnread: true,
	}))

	assert.NotEqual(t, -1, messageThreadStatementIndex(pool, `FROM "message_thread_deleted_items"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `INSERT INTO "message_threads"`))
	assert.Equal(t, -1, messageThreadStatementIndex(pool, `INSERT INTO "message_thread_unread_items"`))
}

func TestMessageThreadStatusUpdatesResetUnreadCount(t *testing.T) {
	zero := uint(0)
	readAt := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		UnreadCount: &zero,
	}, readAt)

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
	}, time.Time{})

	assert.Equal(t, map[string]any{"is_archived": true}, updates)
	assert.NotContains(t, updates, "unread_count")
	assert.NotContains(t, updates, "last_read_at")
}

func TestMessageThreadStatusResetDeletesLedgerRows(t *testing.T) {
	threadID := uuid.New()
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
		},
	)

	require.NoError(t, err)
	require.NotNil(t, thread)
	assert.Zero(t, thread.UnreadCount)
	assert.False(t, thread.LastReadAt.IsZero())
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

func TestMessageThreadStatusResetCreatesUTCWatermarkAfterLock(t *testing.T) {
	threadID := uuid.New()
	zero := uint(0)
	pool := &messageThreadTestConnPool{
		thread: &entities.MessageThread{
			ID:          threadID,
			UserID:      entities.UserID("user-id"),
			UnreadCount: 2,
		},
	}
	repository := newMessageThreadTestRepository(t, pool)
	sourceWatermark := time.Date(2026, 8, 21, 12, 30, 0, 123, time.FixedZone("UTC+2", 2*60*60))
	repository.now = func() time.Time {
		assert.NotEqual(t, -1, messageThreadStatementIndex(pool, "FOR UPDATE"))
		return sourceWatermark
	}

	thread, err := repository.UpdateStatus(
		context.Background(),
		entities.UserID("user-id"),
		threadID,
		MessageThreadStatusUpdate{UnreadCount: &zero},
	)

	require.NoError(t, err)
	require.NotNil(t, thread)
	assert.Equal(t, sourceWatermark.UTC(), thread.LastReadAt)
	statusUpdate := messageThreadStatementIndex(pool, `"last_read_at"`)
	require.NotEqual(t, -1, statusUpdate)
	assert.Contains(t, pool.statements[statusUpdate].args, sourceWatermark.UTC())
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
