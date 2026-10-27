package services

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

type notificationMessageRepository struct {
	repositories.MessageRepository
	stored entities.Message
}

func (repository *notificationMessageRepository) Load(context.Context, entities.UserID, uuid.UUID) (*entities.Message, error) {
	stale := repository.stored
	stale.Status = entities.MessageStatusPending
	return &stale, nil
}

func (repository *notificationMessageRepository) Update(_ context.Context, message *entities.Message) error {
	repository.stored = *message
	return nil
}

func (repository *notificationMessageRepository) IncrementSendAttemptCount(context.Context, entities.UserID, uuid.UUID) error {
	repository.stored.SendAttemptCount++
	return nil
}

func TestNotificationSentDoesNotRestoreStalePendingStatus(t *testing.T) {
	repository := &notificationMessageRepository{stored: entities.Message{
		ID:     uuid.New(),
		UserID: entities.UserID("user-id"),
		Status: entities.MessageStatusScheduled,
	}}
	logger := &noopLogger{}
	service := &MessageService{
		logger:     logger,
		tracer:     telemetry.NewOtelLogger("test", logger),
		repository: repository,
	}

	err := service.HandleMessageNotificationSent(context.Background(), HandleMessageParams{
		ID:        repository.stored.ID,
		UserID:    repository.stored.UserID,
		Timestamp: time.Now().UTC(),
	})

	require.NoError(t, err)
	assert.Equal(t, entities.MessageStatus(entities.MessageStatusScheduled), repository.stored.Status)
	assert.EqualValues(t, 1, repository.stored.SendAttemptCount)
}

func TestGetSendDelay_WithSendAt_ReturnsTimeUntil(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	sendAt := time.Now().UTC().Add(5 * time.Minute)
	params := MessageSendParams{SendAt: &sendAt}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 10)

	// Should be approximately 5 minutes (within 2 seconds tolerance)
	assert.InDelta(t, float64(5*time.Minute), float64(delay), float64(2*time.Second))
}

func TestGetSendDelay_WithSendAtInPast_ReturnsZero(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	sendAt := time.Now().UTC().Add(-5 * time.Minute)
	params := MessageSendParams{SendAt: &sendAt}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 10)

	assert.Equal(t, time.Duration(0), delay)
}

func TestGetSendDelay_BulkIndex_RateBasedDelay(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	params := MessageSendParams{Index: 3}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	// 10 messages per minute = 6 seconds interval
	delay := service.getSendDelay(logger, payload, params, 10)

	expected := time.Duration(3) * (time.Minute / time.Duration(10))
	assert.Equal(t, expected, delay)
}

func TestGetSendDelay_BulkIndex_ZeroRate_ReturnsZero(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	params := MessageSendParams{Index: 5}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 0)

	assert.Equal(t, time.Duration(0), delay)
}

func TestGetSendDelay_IndexZero_ReturnsZero(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	params := MessageSendParams{Index: 0}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 10)

	assert.Equal(t, time.Duration(0), delay)
}

func TestGetSendDelay_NoSendAtNoIndex_ReturnsZero(t *testing.T) {
	service := &MessageService{}
	logger := &noopLogger{}

	params := MessageSendParams{}
	payload := events.MessageAPISentPayload{MessageID: uuid.New()}

	delay := service.getSendDelay(logger, payload, params, 10)

	assert.Equal(t, time.Duration(0), delay)
}

// noopLogger implements telemetry.Logger for testing
type noopLogger struct{}

var _ telemetry.Logger = (*noopLogger)(nil)

func (l *noopLogger) Error(_ error)                                 {}
func (l *noopLogger) WithService(_ string) telemetry.Logger         { return l }
func (l *noopLogger) WithString(_, _ string) telemetry.Logger       { return l }
func (l *noopLogger) WithSpan(_ trace.SpanContext) telemetry.Logger { return l }
func (l *noopLogger) Trace(_ string)                                {}
func (l *noopLogger) Info(_ string)                                 {}
func (l *noopLogger) Warn(_ error)                                  {}
func (l *noopLogger) Debug(_ string)                                {}
func (l *noopLogger) Fatal(_ error)                                 {}
func (l *noopLogger) Printf(_ string, _ ...interface{})             {}
