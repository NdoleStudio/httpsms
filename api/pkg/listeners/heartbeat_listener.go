package listeners

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// HeartbeatListener handles cloud events which need to register entities.Heartbeat
type HeartbeatListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.HeartbeatService
}

// NewHeartbeatListener creates a new instance of HeartbeatListener
func NewHeartbeatListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.HeartbeatService,
) (l *HeartbeatListener, routes map[string]events.EventListener) {
	l = &HeartbeatListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{}
}
