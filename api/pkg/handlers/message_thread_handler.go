package handlers

import (
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/requests"
	"github.com/NdoleStudio/http-sms-manager/pkg/services"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/NdoleStudio/http-sms-manager/pkg/validators"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// MessageThreadHandler handles message-thead http requests.
type MessageThreadHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.MessageThreadHandlerValidator
	service   *services.MessageThreadService
}

// NewMessageThreadHandler creates a new MessageThreadHandler
func NewMessageThreadHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.MessageThreadHandlerValidator,
	service *services.MessageThreadService,
) (h *MessageThreadHandler) {
	return &MessageThreadHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *MessageThreadHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/message-threads", h.Index)
}

// Index returns message threads for a phone number
// @Summary      Get message threads for a phone number
// @Description  Get list of contacts which a phone number has communicated with (threads). It will be sorted by timestamp in descending order.
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        owner	query  string  	true 	"owner phone number" 						default(+18005550199)
// @Param        skip	query  int  	false	"number of messages to skip"				minimum(0)
// @Param        query	query  string  	false 	"filter message threads containing query"
// @Param        limit	query  int  	false	"number of messages to return"				minimum(1)	maximum(20)
// @Success      200 	{object}	responses.MessageThreadsResponse
// @Success      400	{object}	responses.BadRequest
// @Success      422	{object}	responses.UnprocessableEntity
// @Success      500	{object}	responses.InternalServerError
// @Router       /message-threads [get]
func (h *MessageThreadHandler) Index(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.MessageThreadIndex
	if err := c.QueryParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateMessageThreadIndex(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while fetching message threads [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching message threads")
	}

	threads, err := h.service.GetThreads(ctx, request.ToGetParams())
	if err != nil {
		msg := fmt.Sprintf("cannot get message threads with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d message %s", len(*threads), h.pluralize("thread", len(*threads))), threads)
}
