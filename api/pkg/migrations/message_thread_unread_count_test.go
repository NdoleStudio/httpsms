package migrations

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMigrateMessageThreadUnreadCountSkipsLegacyBackfillWhenIsReadColumnMissing(t *testing.T) {
	db, recorder := newMigrationTestDB(t, migrationTestDBOptions{})

	err := MigrateMessageThreadUnreadCount(db)

	require.NoError(t, err)
	assert.NotEmpty(t, recorder.execs)
	assert.NotContains(t, strings.Join(recorder.execs, "\n"), `UPDATE "message_threads" SET "unread_count"=$1 WHERE is_read = $2 AND unread_count = $3`)
	assert.NotContains(t, strings.Join(recorder.execs, "\n"), `ALTER TABLE "message_threads" DROP COLUMN "is_read"`)
}

func TestMigrateMessageThreadUnreadCountPropagatesSchemaErrors(t *testing.T) {
	db, _ := newMigrationTestDB(t, migrationTestDBOptions{
		failExecContains: `CREATE TABLE "message_threads"`,
	})

	err := MigrateMessageThreadUnreadCount(db)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot migrate message thread unread count schema")
	assert.Contains(t, err.Error(), `create table failed: CREATE TABLE "message_threads"`)
}

type migrationTestDBOptions struct {
	failExecContains string
}

type migrationTestRecorder struct {
	execs            []string
	failExecContains string
}

type migrationTestDriver struct {
	recorder *migrationTestRecorder
}

func (driver *migrationTestDriver) Open(string) (driver.Conn, error) {
	return &migrationTestConn{recorder: driver.recorder}, nil
}

type migrationTestConn struct {
	recorder *migrationTestRecorder
}

func (*migrationTestConn) Close() error {
	return nil
}

func (*migrationTestConn) Begin() (driver.Tx, error) {
	return migrationTestTx{}, nil
}

func (*migrationTestConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return migrationTestTx{}, nil
}

func (conn *migrationTestConn) Prepare(query string) (driver.Stmt, error) {
	return &migrationTestStmt{conn: conn, query: query}, nil
}

func (conn *migrationTestConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	conn.recorder.execs = append(conn.recorder.execs, query)
	if conn.recorder.failExecContains != "" && strings.Contains(query, conn.recorder.failExecContains) {
		return nil, errors.New("create table failed: " + query)
	}
	return driver.RowsAffected(1), nil
}

func (conn *migrationTestConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	upperQuery := strings.ToUpper(query)
	switch {
	case strings.Contains(upperQuery, "SELECT CURRENT_DATABASE()"):
		return &migrationTestRows{
			columns: []string{"current_database"},
			values:  [][]driver.Value{{"httpsms_test"}},
		}, nil
	case strings.Contains(upperQuery, "FROM INFORMATION_SCHEMA.TABLES"):
		return &migrationTestRows{
			columns: []string{"count"},
			values:  [][]driver.Value{{int64(0)}},
		}, nil
	case strings.Contains(upperQuery, "FROM INFORMATION_SCHEMA.COLUMNS"):
		return &migrationTestRows{
			columns: []string{"count"},
			values:  [][]driver.Value{{int64(0)}},
		}, nil
	default:
		return nil, errors.New("unexpected query: " + query)
	}
}

type migrationTestStmt struct {
	conn  *migrationTestConn
	query string
}

func (*migrationTestStmt) Close() error {
	return nil
}

func (*migrationTestStmt) NumInput() int {
	return -1
}

func (stmt *migrationTestStmt) Exec(args []driver.Value) (driver.Result, error) {
	return stmt.conn.ExecContext(context.Background(), stmt.query, migrationNamedValues(args))
}

func (stmt *migrationTestStmt) Query(args []driver.Value) (driver.Rows, error) {
	return stmt.conn.QueryContext(context.Background(), stmt.query, migrationNamedValues(args))
}

type migrationTestTx struct{}

func (migrationTestTx) Commit() error {
	return nil
}

func (migrationTestTx) Rollback() error {
	return nil
}

type migrationTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (rows *migrationTestRows) Columns() []string {
	return rows.columns
}

func (*migrationTestRows) Close() error {
	return nil
}

func (rows *migrationTestRows) Next(dest []driver.Value) error {
	if rows.index >= len(rows.values) {
		return io.EOF
	}

	copy(dest, rows.values[rows.index])
	rows.index++
	return nil
}

func migrationNamedValues(values []driver.Value) []driver.NamedValue {
	namedValues := make([]driver.NamedValue, len(values))
	for index, value := range values {
		namedValues[index] = driver.NamedValue{
			Ordinal: index + 1,
			Value:   value,
		}
	}
	return namedValues
}

func newMigrationTestDB(t *testing.T, options migrationTestDBOptions) (*gorm.DB, *migrationTestRecorder) {
	t.Helper()

	recorder := &migrationTestRecorder{
		failExecContains: options.failExecContains,
	}
	driverName := "migration-test-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	sql.Register(driverName, &migrationTestDriver{recorder: recorder})

	sqlDB, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})

	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn:             sqlDB,
			WithoutReturning: true,
		}),
		&gorm.Config{DisableAutomaticPing: true},
	)
	require.NoError(t, err)

	return db, recorder
}
