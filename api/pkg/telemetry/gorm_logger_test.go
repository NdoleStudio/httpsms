package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGormLoggerUsesParameterizedQueries(t *testing.T) {
	logger := &gormLogger{}
	secret := "https://adapter.example.com/secret?token=customer-secret"

	query, params := logger.ParamsFilter(
		context.Background(),
		`UPDATE "phones" SET "fcm_token"=$1 WHERE "id"=$2`,
		secret,
		"phone-id",
	)

	assert.Equal(t, `UPDATE "phones" SET "fcm_token"=$1 WHERE "id"=$2`, query)
	assert.Empty(t, params)
	assert.NotContains(t, query, secret)
}
