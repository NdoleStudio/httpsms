package repositories

import (
	"database/sql"
	"fmt"

	_ "github.com/tursodatabase/libsql-client-go/libsql" // libSQL database driver

	"github.com/palantir/stacktrace"
)

const (
	tableHeartbeats        = "heartbeats"
	tableHeartbeatMonitors = "heartbeat_monitors"
)

// NewTursoDB creates a new *sql.DB connection to a Turso database and auto-creates tables
func NewTursoDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("libsql", dsn)
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot open turso database with DSN [%s]", dsn))
	}

	if err = db.Ping(); err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot ping turso database with DSN [%s]", dsn))
	}

	if err = createTursoTables(db); err != nil {
		return nil, stacktrace.Propagate(err, "cannot create turso tables")
	}

	return db, nil
}

func createTursoTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS ` + tableHeartbeats + ` (
			id TEXT PRIMARY KEY,
			owner TEXT NOT NULL,
			version TEXT NOT NULL,
			charging INTEGER NOT NULL DEFAULT 0,
			user_id TEXT NOT NULL,
			timestamp DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_heartbeats_owner_timestamp ON ` + tableHeartbeats + `(owner, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_heartbeats_user_id ON ` + tableHeartbeats + `(user_id)`,
		`CREATE TABLE IF NOT EXISTS ` + tableHeartbeatMonitors + ` (
			id TEXT PRIMARY KEY,
			phone_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			queue_id TEXT NOT NULL DEFAULT '',
			owner TEXT NOT NULL,
			phone_online INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_heartbeat_monitors_user_owner ON ` + tableHeartbeatMonitors + `(user_id, owner)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return stacktrace.Propagate(err, fmt.Sprintf("cannot execute statement: %s", stmt))
		}
	}

	return nil
}
