package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/hirosassa/zerodriver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestHedgingHeartbeatRepository_StoreAndRead(t *testing.T) {
	postgresURL := os.Getenv("TEST_DATABASE_URL")
	tursoURL := os.Getenv("TEST_TURSO_DATABASE_URL")
	if postgresURL == "" || tursoURL == "" {
		t.Skip("TEST_DATABASE_URL and TEST_TURSO_DATABASE_URL must be set for integration tests")
	}

	// Setup PostgreSQL (primary)
	gormDB, err := gorm.Open(postgres.Open(postgresURL), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gormDB.AutoMigrate(&entities.Heartbeat{}))

	// Setup Turso/libSQL (secondary)
	tursoDB, err := NewTursoDB(tursoURL, os.Getenv("TEST_TURSO_AUTH_TOKEN"))
	require.NoError(t, err)
	defer tursoDB.Close()

	// Telemetry
	driver := zerodriver.NewProductionLogger()
	logger := telemetry.NewZerologLogger("test", map[string]string{}, driver, nil)
	tracer := telemetry.NewOtelLogger("test", logger)

	failureCounter, err := otel.Meter("test").Int64Counter("hedging.test.failures")
	require.NoError(t, err)

	// Build repositories
	primaryRepo := NewGormHeartbeatRepository(logger, tracer, gormDB)
	secondaryRepo := NewLibsqlHeartbeatRepository(logger, tracer, tursoDB)
	hedgingRepo := NewHedgingHeartbeatRepository(logger, tracer, primaryRepo, secondaryRepo, failureCounter)

	// Create test heartbeat
	now := time.Now().UTC().Truncate(time.Second)
	heartbeat := &entities.Heartbeat{
		ID:        uuid.New(),
		Owner:     "+18005551234",
		Version:   "test-v1",
		Charging:  true,
		UserID:    entities.UserID("test-user-" + uuid.New().String()),
		Timestamp: now,
	}

	// Store via hedging repo (writes to both stores)
	ctx := context.Background()
	err = hedgingRepo.Store(ctx, heartbeat)
	require.NoError(t, err)

	// Read from primary (PostgreSQL)
	primaryResult, err := primaryRepo.Last(ctx, heartbeat.UserID, heartbeat.Owner)
	require.NoError(t, err)
	assert.Equal(t, heartbeat.ID, primaryResult.ID)
	assert.Equal(t, heartbeat.Owner, primaryResult.Owner)
	assert.Equal(t, heartbeat.Version, primaryResult.Version)
	assert.Equal(t, heartbeat.Charging, primaryResult.Charging)
	assert.Equal(t, heartbeat.UserID, primaryResult.UserID)
	assert.WithinDuration(t, heartbeat.Timestamp, primaryResult.Timestamp, time.Second)

	// Read from secondary (Turso/libSQL)
	secondaryResult, err := secondaryRepo.Last(ctx, heartbeat.UserID, heartbeat.Owner)
	require.NoError(t, err)
	assert.Equal(t, heartbeat.ID, secondaryResult.ID)
	assert.Equal(t, heartbeat.Owner, secondaryResult.Owner)
	assert.Equal(t, heartbeat.Version, secondaryResult.Version)
	assert.Equal(t, heartbeat.Charging, secondaryResult.Charging)
	assert.Equal(t, heartbeat.UserID, secondaryResult.UserID)
	assert.WithinDuration(t, heartbeat.Timestamp, secondaryResult.Timestamp, time.Second)

	// Verify both stores have the same data
	assert.Equal(t, primaryResult.ID, secondaryResult.ID)

	// Cleanup
	require.NoError(t, hedgingRepo.DeleteAllForUser(ctx, heartbeat.UserID))
}
