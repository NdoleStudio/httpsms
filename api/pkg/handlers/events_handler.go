package handlers

import (
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v3"
	"github.com/palantir/stacktrace"
)

// EventsHandler handles heartbeat http requests.
type EventsHandler struct {
	handler
	logger      telemetry.Logger
	tracer      telemetry.Tracer
	queueConfig services.PushQueueConfig
	service     *services.EventDispatcher
}

// NewEventsHandler creates a new EventsHandler
func NewEventsHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	queueConfig services.PushQueueConfig,
	service *services.EventDispatcher,
) (h *EventsHandler) {
	return &EventsHandler{
		logger:      logger.WithService(fmt.Sprintf("%T", h)),
		tracer:      tracer,
		queueConfig: queueConfig,
		service:     service,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *EventsHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	h.register(router, fiber.MethodPost, "/v1/events", middlewares, h.Dispatch)
}

// Dispatch a cloud event
// This is an internal API so no documentation provided
func (h *EventsHandler) Dispatch(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	var request cloudevents.Event
	if err := c.Bind().Body(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagate(err, "cannot marshall params [%s] into %T", c.OriginalURL(), request))
		return h.responseBadRequest(c, err)
	}

	if err := request.Validate(); err != nil {
		ctxLogger.Warn(stacktrace.NewError("validation errors [%s], while dispatching event [%+#v]", spew.Sdump(err.Error()), request))
		return h.responseUnprocessableEntity(c, map[string][]string{"event": {err.Error()}}, "validation errors while dispatching event")
	}

	if h.userIDFomContext(c) != h.queueConfig.UserID {
		ctxLogger.Error(stacktrace.NewError("user with ID [%s], cannot dispatch event [%+#v]", h.userIDFomContext(c), request))
		return h.responseForbidden(c)
	}

	ctxLogger.Info(fmt.Sprintf("handling [%s] event with ID [%s]", request.Type(), request.ID()))
	err := h.service.DispatchSync(ctx, request)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot dispatch [%s] event with ID [%s]", request.Type(), request.ID()))
		return h.responseInternalServerError(c)
	}

	return h.responseNoContent(c, "event dispatched successfully")
}
