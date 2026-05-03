package services

import (
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

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
