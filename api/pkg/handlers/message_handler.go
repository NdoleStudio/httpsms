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

// MessageHandler handles message http requests.
type MessageHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.MessageHandlerValidator
	service   *services.MessageService
}

// NewMessageHandler creates a new MessageHandler
func NewMessageHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.MessageHandlerValidator,
	service *services.MessageService,
) (h *MessageHandler) {
	return &MessageHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *MessageHandler) RegisterRoutes(router fiber.Router) {
	router.Post("/messages/send", h.PostSend)
	router.Get("/messages/outstanding", h.GetOutstanding)
	router.Get("/messages", h.Index)
}

// PostSend a new entities.Message
// @Summary      Send a new SMS message
// @Description  Add a new SMS message to be sent by the android phone
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        payload   body requests.MessageSend  true  "PostSend message request payload"
// @Success      200  {object}  responses.MessageResponse
// @Success      400  {object}  responses.BadRequest
// @Success      422  {object}  responses.UnprocessableEntity
// @Success      500  {object}  responses.InternalServerError
// @Router       /messages/send [post]
func (h *MessageHandler) PostSend(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.MessageSend
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall [%s] into %T", c.Body(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateMessageSend(ctx, request); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while sending payload [%s]", spew.Sdump(errors), c.Body())
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while sending message")
	}

	message, err := h.service.SendMessage(ctx, request.ToMessageSendParams(c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot send message with paylod [%s]", c.Body())
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "message added to queue", message)
}

// GetOutstanding returns entities.Message which are still to be sent by the mobile phone
// @Summary      Get outstanding messages
// @Description  Get list of messages which are outstanding to be sent by the phone
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        limit	query  int  false  "Number of outstanding messages to return"	minimum(1)	maximum(10)
// @Success      200 	{object}	responses.MessagesResponse
// @Success      400	{object}	responses.BadRequest
// @Success      422	{object}	responses.UnprocessableEntity
// @Success      500	{object}	responses.InternalServerError
// @Router       /messages/outstanding [get]
func (h *MessageHandler) GetOutstanding(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.MessageOutstanding
	if err := c.QueryParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateMessageOutstanding(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while fetching outstanding messages [%s]", spew.Sdump(errors), c.OriginalURL())
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching outstanding messages")
	}

	messages, err := h.service.GetOutstanding(ctx, request.ToGetOutstandingParams(c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot get messgaes with URL [%s]", c.OriginalURL())
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d %s", len(*messages), h.pluralize("message", len(*messages))), messages)
}

// Index returns messages sent between 2 phone numbers
// @Summary      Get messages which are sent between 2 phone numbers
// @Description  Get list of messages which are sent between 2 phone numbers. It will be sorted by timestamp in descending order.
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        from	query  string  	true 	"from phone number" 				default(+18005550199)
// @Param        to		query  string  	true 	"to phone number" 					default(+18005550100)
// @Param        skip	query  int  	false	"number of messages to skip"		minimum(0)
// @Param        query	query  string  	false 	"filter messages containing query"
// @Param        limit	query  int  	false	"number of messages to return"		minimum(1)	maximum(20)
// @Success      200 	{object}	responses.MessagesResponse
// @Success      400	{object}	responses.BadRequest
// @Success      422	{object}	responses.UnprocessableEntity
// @Success      500	{object}	responses.InternalServerError
// @Router       /messages [get]
func (h *MessageHandler) Index(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.MessageIndex
	if err := c.QueryParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateMessageIndex(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while fetching messages [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching messages")
	}

	messages, err := h.service.GetMessages(ctx, request.ToGetParams())
	if err != nil {
		msg := fmt.Sprintf("cannot get messgaes with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d %s", len(*messages), h.pluralize("message", len(*messages))), messages)
}
