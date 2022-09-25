package handlers

import (
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
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
	router.Post("/messages/receive", h.PostReceive)
	router.Get("/messages/outstanding", h.GetOutstanding)
	router.Get("/messages", h.Index)
	router.Post("/messages/:messageID/events", h.PostEvent)
}

// PostSend a new entities.Message
// @Summary      Send a new SMS message
// @Description  Add a new SMS message to be sent by the android phone
// @Security	 ApiKeyAuth
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        payload   body requests.MessageSend  true  "PostSend message request payload"
// @Success      200  {object}  responses.MessageResponse
// @Failure      400  {object}  responses.BadRequest
// @Failure 	 401  {object}	responses.Unauthorized
// @Failure      422  {object}  responses.UnprocessableEntity
// @Failure      500  {object}  responses.InternalServerError
// @Router       /messages/send [post]
// @Security	 ApiKeyAuth
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

	if errors := h.validator.ValidateMessageSend(ctx, h.userIDFomContext(c), request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while sending payload [%s]", spew.Sdump(errors), c.Body())
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while sending message")
	}

	message, err := h.service.SendMessage(ctx, request.ToMessageSendParams(h.userIDFomContext(c), c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot send message with paylod [%s]", c.Body())
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "message added to queue", message)
}

// GetOutstanding returns an entities.Message which is still to be sent by the mobile phone
// @Summary      Get an outstanding message
// @Description  Get an outstanding message to be sent by an android phone
// @Security	 ApiKeyAuth
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        message_id	query  		string  						true "The ID of the message" default(32343a19-da5e-4b1b-a767-3298a73703cb)
// @Success      200 		{object}	responses.MessageResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /messages/outstanding [get]
func (h *MessageHandler) GetOutstanding(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	timestamp := time.Now().UTC()
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

	message, err := h.service.GetOutstanding(ctx, request.ToGetOutstandingParams(c.OriginalURL(), h.userIDFomContext(c), timestamp))
	if err != nil {
		msg := fmt.Sprintf("cannot get outstnading messgage with ID [%s]", request.MessageID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "outstanding message fetched successfully", message)
}

// Index returns messages sent between 2 phone numbers
// @Summary      Get messages which are sent between 2 phone numbers
// @Description  Get list of messages which are sent between 2 phone numbers. It will be sorted by timestamp in descending order.
// @Security	 ApiKeyAuth
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        owner		query  string  	true 	"the owner's phone number" 			default(+18005550199)
// @Param        contact	query  string  	true 	"the contact's phone number" 		default(+18005550100)
// @Param        skip		query  int  	false	"number of messages to skip"		minimum(0)
// @Param        query		query  string  	false 	"filter messages containing query"
// @Param        limit		query  int  	false	"number of messages to return"		minimum(1)	maximum(20)
// @Success      200 		{object}	responses.MessagesResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
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

	messages, err := h.service.GetMessages(ctx, request.ToGetParams(h.userIDFomContext(c)))
	if err != nil {
		msg := fmt.Sprintf("cannot get messgaes with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d %s", len(*messages), h.pluralize("message", len(*messages))), messages)
}

// PostEvent registers an event on a message
// @Summary      Upsert an event for a message on the mobile phone
// @Description  Use this endpoint to send events for a message when it is failed, sent or delivered by the mobile phone.
// @Security	 ApiKeyAuth
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param 		 messageID 	path		string 							true 	"ID of the message" 			default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Param        payload   	body 		requests.MessageEvent  			true 	"Payload of the event emitted."
// @Success      200  		{object} 	responses.MessageResponse
// @Failure      400  		{object}  	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure 	 404		{object}	responses.NotFound
// @Failure      422  		{object} 	responses.UnprocessableEntity
// @Failure      500  		{object}  	responses.InternalServerError
// @Router       /messages/{messageID}/events [post]
// @Security	 ApiKeyAuth
func (h *MessageHandler) PostEvent(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.MessageEvent
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall [%s] into %T", c.Body(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	request.MessageID = c.Params("messageID")
	if errors := h.validator.ValidateMessageEvent(ctx, request); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while storing event [%s] for message [%s]", spew.Sdump(errors), c.Body(), request.MessageID)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while storing event")
	}

	message, err := h.service.GetMessage(ctx, h.userIDFomContext(c), uuid.MustParse(request.MessageID))
	if err != nil && stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, fmt.Sprintf("cannot find message with ID [%s]", request.MessageID))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", request.MessageID)
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseInternalServerError(c)
	}

	message, err = h.service.StoreEvent(ctx, message, request.ToMessageStoreEventParams(c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot store event for message [%s] with paylod [%s]", request.MessageID, c.Body())
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "message event stored successfully", message)
}

// PostReceive receives a new entities.Message
// @Summary      Receive a new SMS message from a mobile phone
// @Description  Add a new message received from a mobile phone
// @Security	 ApiKeyAuth
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        payload   body requests.MessageReceive  true  "Received message request payload"
// @Success      200  {object}  responses.MessageResponse
// @Failure      400  {object}  responses.BadRequest
// @Failure      422  {object}  responses.UnprocessableEntity
// @Failure      500  {object}  responses.InternalServerError
// @Router       /messages/receive [post]
func (h *MessageHandler) PostReceive(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.MessageReceive
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall [%s] into %T", c.Body(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateMessageReceive(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while sending payload [%s]", spew.Sdump(errors), c.Body())
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while receiving message")
	}

	message, err := h.service.ReceiveMessage(ctx, request.ToMessageReceiveParams(h.userIDFomContext(c), c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot receive message with paylod [%s]", c.Body())
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "message received successfully", message)
}
