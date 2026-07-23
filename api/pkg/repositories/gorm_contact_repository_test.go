package repositories

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func normalizeContactSQL(query string) string {
	return strings.Join(strings.Fields(query), " ")
}

type contactTestConnPool struct {
	messageThreadTestConnPool
}

func (pool *contactTestConnPool) QueryContext(_ context.Context, query string, args ...any) (*sql.Rows, error) {
	pool.statements = append(pool.statements, messageThreadTestStatement{
		query: query,
		args:  append([]any(nil), args...),
	})

	db := sql.OpenDB(contactTestRowsConnector{})
	return db.QueryContext(context.Background(), "SELECT 1")
}

func (pool *contactTestConnPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	return pool, nil
}

type contactTestRowsConnector struct{}

func (contactTestRowsConnector) Connect(context.Context) (driver.Conn, error) {
	return contactTestRowsConn{}, nil
}

func (contactTestRowsConnector) Driver() driver.Driver {
	return contactTestRowsDriver{}
}

type contactTestRowsDriver struct{}

func (contactTestRowsDriver) Open(string) (driver.Conn, error) {
	return contactTestRowsConn{}, nil
}

type contactTestRowsConn struct{}

func (contactTestRowsConn) Prepare(string) (driver.Stmt, error) {
	return nil, assert.AnError
}

func (contactTestRowsConn) Close() error {
	return nil
}

func (contactTestRowsConn) Begin() (driver.Tx, error) {
	return nil, assert.AnError
}

func (contactTestRowsConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return contactTestRows{}, nil
}

type contactTestRows struct{}

func (contactTestRows) Columns() []string {
	return []string{"id", "user_id", "name", "emails", "phone_numbers", "properties", "created_at", "updated_at"}
}

func (contactTestRows) Close() error {
	return nil
}

func (contactTestRows) Next([]driver.Value) error {
	return io.EOF
}

func lastContactStatement(t *testing.T, pool *contactTestConnPool) messageThreadTestStatement {
	t.Helper()
	require.NotEmpty(t, pool.statements)

	statement := pool.statements[len(pool.statements)-1]
	statement.query = normalizeContactSQL(statement.query)
	return statement
}

func newContactTestRepo(t *testing.T) (ContactRepository, *contactTestConnPool) {
	t.Helper()

	pool := &contactTestConnPool{}
	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn:             pool,
			WithoutReturning: true,
		}),
		&gorm.Config{
			DisableAutomaticPing: true,
			NowFunc: func() time.Time {
				return time.Date(2026, 7, 19, 18, 0, 0, 0, time.UTC)
			},
		},
	)
	require.NoError(t, err)

	logger := &messageThreadTestLogger{}
	return NewGormContactRepository(logger, telemetry.NewOtelLogger("test", logger), db), pool
}

func TestGormContactRepository_Store_BatchInsertsContacts(t *testing.T) {
	repository, recorder := newContactTestRepo(t)
	firstID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	secondID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	err := repository.Store(context.Background(), []*entities.Contact{
		{
			ID:           firstID,
			UserID:       entities.UserID("user-1"),
			Name:         "Alice",
			Emails:       pq.StringArray{"alice@example.com"},
			PhoneNumbers: pq.StringArray{"+18005550199"},
			Properties:   entities.ContactProperties{"source": "phone"},
		},
		{
			ID:           secondID,
			UserID:       entities.UserID("user-1"),
			Name:         "Bob",
			Emails:       pq.StringArray{"bob@example.com"},
			PhoneNumbers: pq.StringArray{"+18005550200"},
			Properties:   entities.ContactProperties{"source": "import"},
		},
	})

	require.NoError(t, err)
	require.Len(t, recorder.statements, 1)
	statement := lastContactStatement(t, recorder)
	assert.True(t, strings.HasPrefix(statement.query, `INSERT INTO "contacts"`))
	assert.Contains(t, statement.query, `("id","user_id","name","emails","phone_numbers","properties","created_at","updated_at")`)
	assert.Regexp(t, regexp.MustCompile(`VALUES \(\$1,\$2,\$3,\$4,\$5,\$6,\$7,\$8\),\(\$9,\$10,\$11,\$12,\$13,\$14,\$15,\$16\)`), statement.query)
	require.Len(t, statement.args, 16)
	assert.Equal(t, firstID, statement.args[0])
	assert.Equal(t, entities.UserID("user-1"), statement.args[1])
	assert.Equal(t, "Alice", statement.args[2])
	assert.Equal(t, pq.StringArray{"alice@example.com"}, statement.args[3])
	assert.Equal(t, pq.StringArray{"+18005550199"}, statement.args[4])
	assert.Equal(t, entities.ContactProperties{"source": "phone"}, statement.args[5])
	assert.Equal(t, secondID, statement.args[8])
	assert.Equal(t, entities.UserID("user-1"), statement.args[9])
	assert.Equal(t, "Bob", statement.args[10])
	assert.Equal(t, pq.StringArray{"bob@example.com"}, statement.args[11])
	assert.Equal(t, pq.StringArray{"+18005550200"}, statement.args[12])
	assert.Equal(t, entities.ContactProperties{"source": "import"}, statement.args[13])
}

func TestGormContactRepository_Update_SavesContact(t *testing.T) {
	repository, recorder := newContactTestRepo(t)
	contactID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	err := repository.Update(context.Background(), &entities.Contact{
		ID:           contactID,
		UserID:       entities.UserID("user-1"),
		Name:         "Updated Alice",
		Emails:       pq.StringArray{"updated@example.com"},
		PhoneNumbers: pq.StringArray{"+18005550199"},
		Properties:   entities.ContactProperties{"group": "friends"},
	})

	require.NoError(t, err)
	statement := lastContactStatement(t, recorder)
	assert.True(t, strings.HasPrefix(statement.query, `UPDATE "contacts"`))
	assert.Contains(t, statement.query, `"user_id"=$1`)
	assert.Contains(t, statement.query, `"name"=$2`)
	assert.Contains(t, statement.query, `"emails"=$3`)
	assert.Contains(t, statement.query, `"phone_numbers"=$4`)
	assert.Contains(t, statement.query, `"properties"=$5`)
	assert.Contains(t, statement.query, `"created_at"=$6`)
	assert.Contains(t, statement.query, `"updated_at"=$7`)
	assert.Contains(t, statement.query, `WHERE "id" = $8`)
	require.Len(t, statement.args, 8)
	assert.Equal(t, entities.UserID("user-1"), statement.args[0])
	assert.Equal(t, "Updated Alice", statement.args[1])
	assert.Equal(t, pq.StringArray{"updated@example.com"}, statement.args[2])
	assert.Equal(t, pq.StringArray{"+18005550199"}, statement.args[3])
	assert.Equal(t, entities.ContactProperties{"group": "friends"}, statement.args[4])
	assert.Equal(t, contactID, statement.args[7])
}

func TestGormContactRepository_Load_ScopesByUserAndContactID(t *testing.T) {
	repository, recorder := newContactTestRepo(t)
	contactID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	_, err := repository.Load(context.Background(), entities.UserID("user-1"), contactID)

	require.Error(t, err)
	assert.Equal(t, ErrCodeNotFound, stacktrace.GetCode(err))
	statement := lastContactStatement(t, recorder)
	assert.True(t, strings.HasPrefix(statement.query, `SELECT * FROM "contacts"`))
	assert.Contains(t, statement.query, `WHERE user_id = $1 AND id = $2`)
	assert.Contains(t, statement.query, `ORDER BY "contacts"."id" LIMIT $3`)
	require.Len(t, statement.args, 3)
	assert.Equal(t, entities.UserID("user-1"), statement.args[0])
	assert.Equal(t, contactID, statement.args[1])
	assert.Equal(t, 1, statement.args[2])
}

func TestGormContactRepository_Count_ReusesIndexFilterWithoutPagination(t *testing.T) {
	repository, recorder := newContactTestRepo(t)

	total, err := repository.Count(context.Background(), entities.UserID("user-1"), IndexParams{
		Query: "alice",
		// Limit/Skip must be ignored by Count.
		Limit: 20,
		Skip:  40,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(0), total)

	statement := lastContactStatement(t, recorder)
	assert.True(t, strings.HasPrefix(statement.query, `SELECT count(*) FROM "contacts"`))
	// Count must apply the exact same user + name/emails/phone_numbers filter as Index.
	assert.Contains(t, statement.query, `WHERE user_id = $1 AND (name ILIKE $2 OR array_to_string(emails, ',') ILIKE $3 OR array_to_string(phone_numbers, ',') ILIKE $4)`)
	// Count must ignore pagination.
	assert.NotContains(t, statement.query, "LIMIT")
	assert.NotContains(t, statement.query, "OFFSET")
	require.Len(t, statement.args, 4)
	assert.Equal(t, entities.UserID("user-1"), statement.args[0])
	assert.Equal(t, "%alice%", statement.args[1])
	assert.Equal(t, "%alice%", statement.args[2])
	assert.Equal(t, "%alice%", statement.args[3])
}

func TestGormContactRepository_Count_ScopesByUserWithoutQuery(t *testing.T) {
	repository, recorder := newContactTestRepo(t)

	_, err := repository.Count(context.Background(), entities.UserID("user-1"), IndexParams{})

	require.NoError(t, err)
	statement := lastContactStatement(t, recorder)
	assert.Equal(t, `SELECT count(*) FROM "contacts" WHERE user_id = $1`, statement.query)
	require.Len(t, statement.args, 1)
	assert.Equal(t, entities.UserID("user-1"), statement.args[0])
}

func TestGormContactRepository_Index_FiltersByUserAndQueryAcrossContactFields(t *testing.T) {
	repository, recorder := newContactTestRepo(t)

	_, err := repository.Index(context.Background(), entities.UserID("user-1"), IndexParams{
		Query: "alice",
		Limit: 20,
		Skip:  40,
	})

	require.NoError(t, err)
	statement := lastContactStatement(t, recorder)
	assert.True(t, strings.HasPrefix(statement.query, `SELECT * FROM "contacts"`))
	assert.Contains(t, statement.query, `WHERE user_id = $1 AND (name ILIKE $2 OR array_to_string(emails, ',') ILIKE $3 OR array_to_string(phone_numbers, ',') ILIKE $4)`)
	assert.Contains(t, statement.query, `ORDER BY updated_at DESC LIMIT $5 OFFSET $6`)
	require.Len(t, statement.args, 6)
	assert.Equal(t, entities.UserID("user-1"), statement.args[0])
	assert.Equal(t, "%alice%", statement.args[1])
	assert.Equal(t, "%alice%", statement.args[2])
	assert.Equal(t, "%alice%", statement.args[3])
	assert.Equal(t, 20, statement.args[4])
	assert.Equal(t, 40, statement.args[5])
}

func TestGormContactRepository_FetchAll_ScopesByUserAndOrdersUpdatedAtAsc(t *testing.T) {
	repository, recorder := newContactTestRepo(t)

	_, err := repository.FetchAll(context.Background(), entities.UserID("user-1"))

	require.NoError(t, err)
	statement := lastContactStatement(t, recorder)
	assert.Equal(t, `SELECT * FROM "contacts" WHERE user_id = $1 ORDER BY updated_at ASC`, statement.query)
	require.Len(t, statement.args, 1)
	assert.Equal(t, entities.UserID("user-1"), statement.args[0])
}

func TestGormContactRepository_Delete_ScopesByUserAndContactID(t *testing.T) {
	repository, recorder := newContactTestRepo(t)
	contactID := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	err := repository.Delete(context.Background(), entities.UserID("user-1"), contactID)

	require.NoError(t, err)
	statement := lastContactStatement(t, recorder)
	assert.Equal(t, `DELETE FROM "contacts" WHERE user_id = $1 AND id = $2`, statement.query)
	require.Len(t, statement.args, 2)
	assert.Equal(t, entities.UserID("user-1"), statement.args[0])
	assert.Equal(t, contactID, statement.args[1])
}

func TestGormContactRepository_DeleteAllForUser_ScopesByUserOnly(t *testing.T) {
	repository, recorder := newContactTestRepo(t)

	err := repository.DeleteAllForUser(context.Background(), entities.UserID("user-1"))

	require.NoError(t, err)
	statement := lastContactStatement(t, recorder)
	assert.Equal(t, `DELETE FROM "contacts" WHERE user_id = $1`, statement.query)
	require.Len(t, statement.args, 1)
	assert.Equal(t, entities.UserID("user-1"), statement.args[0])
}
