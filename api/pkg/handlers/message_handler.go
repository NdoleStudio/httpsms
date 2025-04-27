package handlers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"

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
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	billingService *services.BillingService
	validator      *validators.MessageHandlerValidator
	service        *services.MessageService
}

// NewMessageHandler creates a new MessageHandler
func NewMessageHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.MessageHandlerValidator,
	billingService *services.BillingService,
	service *services.MessageService,
) (h *MessageHandler) {
	return &MessageHandler{
		logger:         logger.WithService(fmt.Sprintf("%T", h)),
		tracer:         tracer,
		validator:      validator,
		billingService: billingService,
		service:        service,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *MessageHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	router.Post("/v1/messages/send", h.computeRoute(middlewares, h.PostSend)...)
	router.Post("/v1/messages/bulk-send", h.computeRoute(middlewares, h.BulkSend)...)
	router.Get("/v1/messages", h.computeRoute(middlewares, h.Index)...)
	router.Get("/v1/messages/search", h.computeRoute(middlewares, h.Search)...)
	router.Delete("/v1/messages/:messageID", h.computeRoute(middlewares, h.Delete)...)
}

// RegisterPhoneAPIKeyRoutes registers the routes for the MessageHandler
func (h *MessageHandler) RegisterPhoneAPIKeyRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	router.Post("/v1/messages/:messageID/events", h.computeRoute(middlewares, h.PostEvent)...)
	router.Post("/v1/messages/receive", h.computeRoute(middlewares, h.PostReceive)...)
	router.Post("/v1/messages/calls/missed", h.computeRoute(middlewares, h.PostCallMissed)...)
	router.Get("/v1/messages/outstanding", h.computeRoute(middlewares, h.GetOutstanding)...)
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

	if msg := h.billingService.IsEntitled(ctx, h.userIDFomContext(c)); msg != nil {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] can't send a message", h.userIDFomContext(c))))
		return h.responsePaymentRequired(c, *msg)
	}

	message, err := h.service.SendMessage(ctx, request.ToMessageSendParams(h.userIDFomContext(c), c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot send message with paylod [%s]", c.Body())
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "message added to queue", message)
}

// BulkSend a bulk entities.Message
// @Summary      Send bulk SMS messages
// @Description  Add bulk SMS messages to be sent by the android phone
// @Security	 ApiKeyAuth
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        payload   body requests.MessageBulkSend  true  "Bulk send message request payload"
// @Success      200  {object}  []responses.MessagesResponse
// @Failure      400  {object}  responses.BadRequest
// @Failure 	 401  {object}	responses.Unauthorized
// @Failure      422  {object}  responses.UnprocessableEntity
// @Failure      500  {object}  responses.InternalServerError
// @Router       /messages/bulk-send [post]
func (h *MessageHandler) BulkSend(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.MessageBulkSend
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall [%s] into %T", c.Body(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateMessageBulkSend(ctx, h.userIDFomContext(c), request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while sending payload [%s]", spew.Sdump(errors), c.Body())
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while sending messages")
	}

	if msg := h.billingService.IsEntitledWithCount(ctx, h.userIDFomContext(c), uint(len(request.To))); msg != nil {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] is not entitled to send [%d] messages", h.userIDFomContext(c), len(request.To))))
		return h.responsePaymentRequired(c, *msg)
	}

	wg := sync.WaitGroup{}
	params := request.ToMessageSendParams(h.userIDFomContext(c), c.OriginalURL())
	responses := make([]*entities.Message, len(params))

	for index, message := range params {
		wg.Add(1)
		go func(message services.MessageSendParams, index int) {
			if message.SendAt == nil {
				sentAt := time.Now().UTC().Add(time.Duration(index) * time.Second)
				message.SendAt = &sentAt
			}

			response, err := h.service.SendMessage(ctx, message)
			if err != nil {
				msg := fmt.Sprintf("cannot send message with paylod [%s]", c.Body())
				ctxLogger.Error(stacktrace.Propagate(err, msg))
			}
			responses[index] = response
			wg.Done()
		}(message, index)
	}

	wg.Wait()
	return h.responseOK(c, fmt.Sprintf("[%d] messages processed successfully", len(responses)), responses)
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

	message, err := h.service.GetOutstanding(ctx, request.ToGetOutstandingParams(c.Path(), h.userFromContext(c), timestamp))
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		msg := fmt.Sprintf("Cannot find outstanding message with ID [%s]", request.MessageID)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseNotFound(c, msg)
	}

	if err != nil {
		msg := fmt.Sprintf("cannot get outstanding messgage with ID [%s]", request.MessageID)
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
	if strings.Contains(request.MessageID, ".") {
		return h.responseNoContent(c, "duplicate send event received.")
	}

	if errors := h.validator.ValidateMessageEvent(ctx, request.Sanitize()); len(errors) != 0 {
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

	if !h.authorizePhoneAPIKey(c, message.Owner) {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] is not authorized to send event for message with ID [%s]", h.userIDFomContext(c), request.MessageID)))
		return h.responsePhoneAPIKeyUnauthorized(c, message.Owner, h.userFromContext(c))
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

	if msg := h.billingService.IsEntitled(ctx, h.userIDFomContext(c)); msg != nil {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] can't receive a message becasuse they have exceeded the limit", h.userIDFomContext(c))))
		return h.responsePaymentRequired(c, *msg)
	}

	if !h.authorizePhoneAPIKey(c, request.To) {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] is not authorized to receive message to phone number [%s]", h.userIDFomContext(c), request.To)))
		return h.responsePhoneAPIKeyUnauthorized(c, request.To, h.userFromContext(c))
	}

	message, err := h.service.ReceiveMessage(ctx, request.ToMessageReceiveParams(h.userIDFomContext(c), c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot receive message with paylod [%s]", c.Body())
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "message received successfully", message)
}

// Delete a message
// @Summary      Delete a message from the database.
// @Description  Delete a message from the database and removes the message content from the list of threads.
// @Security	 ApiKeyAuth
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param 		 messageID 	path		string 							true 	"ID of the message" 			default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Success      204  		{object} 	responses.NoContent
// @Failure      400  		{object}  	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure 	 404		{object}	responses.NotFound
// @Failure      422  		{object} 	responses.UnprocessableEntity
// @Failure      500  		{object}  	responses.InternalServerError
// @Router       /messages/{messageID} [delete]
func (h *MessageHandler) Delete(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	messageID := c.Params("messageID")
	if errors := h.validator.ValidateUUID(messageID, "messageID"); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while deleting a message with ID [%s]", spew.Sdump(errors), messageID)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while storing event")
	}

	message, err := h.service.GetMessage(ctx, h.userIDFomContext(c), uuid.MustParse(messageID))
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, fmt.Sprintf("cannot find message with ID [%s]", messageID))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", messageID)
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseInternalServerError(c)
	}

	if err = h.service.DeleteMessage(ctx, c.OriginalURL(), message); err != nil {
		msg := fmt.Sprintf("cannot delete message with ID [%s] for user with ID [%s]", messageID, message.UserID)
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseInternalServerError(c)
	}

	return h.responseNoContent(c, "message deleted successfully")
}

// PostCallMissed registers a missed phone call
// @Summary      Register a missed call event on the mobile phone
// @Description  This endpoint is called by the httpSMS android app to register a missed call event on the mobile phone.
// @Security	 ApiKeyAuth
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.MessageCallMissed  	true	"Payload of the missed call event."
// @Success      200  		{object} 	responses.MessageResponse
// @Failure      400  		{object}  	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure 	 404		{object}	responses.NotFound
// @Failure      422  		{object} 	responses.UnprocessableEntity
// @Failure      500  		{object}  	responses.InternalServerError
// @Router       /messages/calls/missed [post]
func (h *MessageHandler) PostCallMissed(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.MessageCallMissed
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall [%s] into %T", c.Body(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateCallMissed(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], for missed call event [%s]", spew.Sdump(errors), c.Body())
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while storing missed call event")
	}

	if !h.authorizePhoneAPIKey(c, request.To) {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] is not authorized to register missed phone call for phone number [%s]", h.userIDFomContext(c), request.To)))
		return h.responsePhoneAPIKeyUnauthorized(c, request.To, h.userFromContext(c))
	}

	message, err := h.service.RegisterMissedCall(ctx, request.ToCallMissedParams(h.userIDFomContext(c), c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot store missed call event for user [%s] with paylod [%s]", h.userIDFomContext(c), c.Body())
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "missed call event stored successfully", message)
}

// Search returns a filtered list of messages of a user
// @Summary      Search all messages of a user
// @Description  This returns the list of all messages based on the filter criteria including missed calls
// @Security	 ApiKeyAuth
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        token    	header string   true   	"Cloudflare turnstile token https://www.cloudflare.com/en-gb/application-services/products/turnstile/"
// @Param        owners		query  string  	true 	"the owner's phone numbers" 		default(+18005550199,+18005550100)
// @Param        skip		query  int  	false	"number of messages to skip"		minimum(0)
// @Param        query		query  string  	false 	"filter messages containing query"
// @Param        limit		query  int  	false	"number of messages to return"		minimum(1)	maximum(200)
// @Success      200 		{object}	responses.MessagesResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /messages/search [get]
func (h *MessageHandler) Search(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	var request requests.MessageSearch
	if err := c.QueryParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params in [%s] into [%T]", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	request.IPAddress = c.IP()
	request.Token = c.Get("token")

	if errors := h.validator.ValidateMessageSearch(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while searching messages [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while searching messages")
	}

	messages, err := h.service.SearchMessages(ctx, request.ToSearchParams(h.userIDFomContext(c)))
	if err != nil {
		msg := fmt.Sprintf("cannot search messages with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("found %d %s", len(messages), h.pluralize("message", len(messages))), messages)
}
