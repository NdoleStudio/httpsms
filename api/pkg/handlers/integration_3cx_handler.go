package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/davecgh/go-spew/spew"

	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// Integration3CXHandler handles 3CX events
type Integration3CXHandler struct {
	handler
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	messageService *services.MessageService
	billingService *services.BillingService
}

// NewIntegration3CxHandler creates a new Integration3CXHandler
func NewIntegration3CxHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	messageService *services.MessageService,
	billingService *services.BillingService,
) (h *Integration3CXHandler) {
	return &Integration3CXHandler{
		logger:         logger.WithService(fmt.Sprintf("%T", h)),
		tracer:         tracer,
		messageService: messageService,
		billingService: billingService,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *Integration3CXHandler) RegisterRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	router := app.Group("integration/3cx/")
	router.Post("/messages", h.computeRoute(middlewares, h.Messages)...)
}

// Messages consumes a 3cx event
// @Summary      Sends a 3CX SMS message
// @Description  Sends an SMS message from the 3CX platform
// @Tags         3CXIntegration
// @Accept       json
// @Produce      json
// @Success      204 		{object}	responses.NoContent
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /integration/3cx/messages [post]
func (h *Integration3CXHandler) Messages(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	spew.Dump(string(c.Body()))

	var request requests.Integration3CXMessage
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall [%s] into %T", c.Body(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if msg := h.billingService.IsEntitled(ctx, h.userIDFomContext(c)); msg != nil {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] can't send a [3cx] message", h.userIDFomContext(c))))
		return h.responsePaymentRequired(c, *msg)
	}

	message, err := h.messageService.SendMessage(ctx, request.ToMessageSendParams(h.userIDFomContext(c), c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot send [3cx] message with paylod [%s]", c.Body())
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "message added to queue", message)
}
