package listeners

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/pusher/pusher-http-go/v5"
	"github.com/stretchr/testify/assert"
)

func TestWebsocketListenerRegistersMissedCalls(t *testing.T) {
	logger := &noopListenerLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	_, routes := NewWebsocketListener(logger, tracer, &pusher.Client{})

	assert.Contains(t, routes, events.MessageCallMissed)
}
