package handlers

import (
	"fmt"
	"sync"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// BulkMessageHandler handles bulk SMS http requests
type BulkMessageHandler struct {
	handler
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	validator      *validators.BulkMessageHandlerValidator
	messageService *services.MessageService
	billingService *services.BillingService
}

// NewBulkMessageHandler creates a new BulkMessageHandler
func NewBulkMessageHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.BulkMessageHandlerValidator,
	billingService *services.BillingService,
	messageService *services.MessageService,
) (h *BulkMessageHandler) {
	return &BulkMessageHandler{
		logger:         logger.WithService(fmt.Sprintf("%T", h)),
		tracer:         tracer,
		validator:      validator,
		messageService: messageService,
		billingService: billingService,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *BulkMessageHandler) RegisterRoutes(router fiber.Router) {
	router.Post("/bulk-messages", h.Store)
}

// Store sends bulk SMS messages from a CSV file.
// @Summary      Store bulk SMS file
// @Description  Sends bulk SMS messages to multiple users from a CSV file.
// @Security	 ApiKeyAuth
// @Tags         BulkSMS
// @Accept       json
// @Produce      json
// @Success      202 		{object}	responses.NoContent
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /bulk-messages [post]
func (h *BulkMessageHandler) Store(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	file, err := c.FormFile("document")
	if err != nil {
		msg := fmt.Sprintf("cannot fetch file with name [%s] from request", "document")
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	messages, validationErrors := h.validator.ValidateStore(ctx, h.userIDFomContext(c), file)
	if len(validationErrors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while sending bulk sms from CSV file [%s] for [%s]", spew.Sdump(validationErrors), file.Filename, h.userIDFomContext(c))
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, validationErrors, "validation errors while sending bulk SMS")
	}

	if msg := h.billingService.IsEntitledWithCount(ctx, h.userIDFomContext(c), uint(len(messages))); msg != nil {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] is not entitled to send [%d] messages", h.userIDFomContext(c), len(messages))))
		return h.responsePaymentRequired(c, *msg)
	}

	requestID := uuid.New()
	wg := sync.WaitGroup{}
	for _, message := range messages {
		wg.Add(1)
		go func(message *requests.BulkMessage) {
			_, err = h.messageService.SendMessage(
				ctx,
				message.ToMessageSendParams(h.userIDFomContext(c), requestID, c.OriginalURL()),
			)

			if err != nil {
				msg := fmt.Sprintf("cannot send message with paylod [%s]", c.Body())
				ctxLogger.Error(stacktrace.Propagate(err, msg))
			}
			wg.Done()
		}(message)
	}

	wg.Wait()
	return h.responseAccepted(c, fmt.Sprintf("Added %d messages to the queue", len(messages)))
}
