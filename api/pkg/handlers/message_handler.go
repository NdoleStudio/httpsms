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

// Send a new entities.Message
// @Summary      Send a new SMS message
// @Description  Add a new SMS message to be sent by the android phone
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Success      200  {object}  responses.MessageResponse
// @Router       /messages/send [post]
func (h *MessageHandler) Send(c *fiber.Ctx) error {
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

	message, err := h.service.Send(ctx, request.ToMessageSendParams())
	if err != nil {
		msg := fmt.Sprintf("cannot send message with paylod [%s]", c.Body())
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "Outgoing message added to queue", message)
}
