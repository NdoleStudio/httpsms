package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
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
	router.Put("/message-threads/:messageThreadID", h.Update)
}

// Index returns message threads for a phone number
// @Summary      Get message threads for a phone number
// @Description  Get list of contacts which a phone number has communicated with (threads). It will be sorted by timestamp in descending order.
// @Security	 ApiKeyAuth
// @Tags         Message Threads
// @Accept       json
// @Produce      json
// @Param        owner	query  string  	true 	"owner phone number" 						default(+18005550199)
// @Param        skip	query  int  	false	"number of messages to skip"				minimum(0)
// @Param        query	query  string  	false 	"filter message threads containing query"
// @Param        limit	query  int  	false	"number of messages to return"				minimum(1)	maximum(20)
// @Success      200 	{object}	responses.MessageThreadsResponse
// @Failure      400	{object}	responses.BadRequest
// @Failure 	 401    {object}	responses.Unauthorized
// @Failure      422	{object}	responses.UnprocessableEntity
// @Failure      500	{object}	responses.InternalServerError
// @Router       /message-threads [get]
func (h *MessageThreadHandler) Index(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	ctxLogger.Info(c.OriginalURL())

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

	threads, err := h.service.GetThreads(ctx, request.ToGetParams(h.userIDFomContext(c)))
	if err != nil {
		msg := fmt.Sprintf("cannot get message threads with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d message %s", len(*threads), h.pluralize("thread", len(*threads))), threads)
}

// Update an entities.MessageThread
// @Summary      Update a message thread
// @Description  Updates the details of a message thread
// @Security	 ApiKeyAuth
// @Tags         Message Threads
// @Accept       json
// @Produce      json
// @Param 		 messageThreadID	path		string 							true 	"ID of the message thread" 						default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Param        payload   			body 		requests.MessageThreadUpdate 	true 	"Payload of message thread details to update"
// @Success      200 				{object}	responses.PhoneResponse
// @Failure      400				{object}	responses.BadRequest
// @Failure 	 401    			{object}	responses.Unauthorized
// @Failure      422				{object}	responses.UnprocessableEntity
// @Failure      500				{object}	responses.InternalServerError
// @Router       /message-threads/{messageThreadID} [put]
func (h *MessageThreadHandler) Update(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.MessageThreadUpdate
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	request.MessageThreadID = c.Params("messageThreadID")
	if errors := h.validator.ValidateUpdate(ctx, request); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while updating message thread [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating message thread")
	}

	thread, err := h.service.UpdateStatus(ctx, request.ToUpdateParams(h.userIDFomContext(c)))
	if err != nil {
		msg := fmt.Sprintf("cannot update message thread with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "message thread updated successfully", thread)
}
