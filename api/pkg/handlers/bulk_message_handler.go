package handlers

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v3"
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
func (h *BulkMessageHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	h.register(router, fiber.MethodGet, "/v1/bulk-messages", middlewares, h.Index)
	h.register(router, fiber.MethodPost, "/v1/bulk-messages", middlewares, h.Store)
}

// Index fetches the bulk message order history.
// @Summary      List bulk message orders
// @Description  Fetches the last 10 bulk message order summaries for the authenticated user showing counts per status.
// @Security	 ApiKeyAuth
// @Tags         BulkSMS
// @Accept       json
// @Produce      json
// @Success      200 		{object}	responses.BulkMessagesResponse
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      500		{object}	responses.InternalServerError
// @Router       /bulk-messages [get]
func (h *BulkMessageHandler) Index(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	orders, err := h.messageService.GetBulkMessages(ctx, h.userIDFomContext(c))
	if err != nil {
		msg := fmt.Sprintf("cannot fetch bulk messages for user [%s]", h.userIDFomContext(c))
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d bulk %s", len(orders), h.pluralize("message", len(orders))), orders)
}

// Store sends bulk SMS messages from a CSV or Excel file.
// @Summary      Store bulk SMS file
// @Description  Sends bulk SMS messages to multiple users based on our [CSV template](https://httpsms.com/templates/httpsms-bulk.csv) or our [Excel template](https://httpsms.com/templates/httpsms-bulk.xlsx).
// @Security	 ApiKeyAuth
// @Tags         BulkSMS
// @Accept       multipart/form-data
// @Produce      json
// @Param        document	formData  	file   							true	"The Excel or CSV file containing the messages to be sent."
// @Success      202 		{object}	responses.NoContent
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /bulk-messages [post]
func (h *BulkMessageHandler) Store(c fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	file, err := c.FormFile("document")
	if err != nil {
		msg := fmt.Sprintf("cannot fetch file with name [%s] from request", "document")
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	messages, userLocation, validationErrors := h.validator.ValidateStore(ctx, h.userIDFomContext(c), file)
	if len(validationErrors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while sending bulk sms from CSV file [%s] for [%s]", spew.Sdump(validationErrors), file.Filename, h.userIDFomContext(c))
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, validationErrors, "validation errors while sending bulk SMS")
	}

	if msg := h.billingService.IsEntitledWithCount(ctx, h.userIDFomContext(c), uint(len(messages))); msg != nil {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] is not entitled to send [%d] messages", h.userIDFomContext(c), len(messages))))
		return h.responsePaymentRequired(c, *msg)
	}

	requestID := h.generateRequestID(file.Filename)
	wg := sync.WaitGroup{}
	count := atomic.Int64{}

	// Compute per-phone index for rate-based dispatch delay
	phoneIndexCounter := make(map[string]int)

	for _, message := range messages {
		wg.Add(1)
		var perPhoneIndex int
		if message.GetSendTime(userLocation) == nil {
			perPhoneIndex = phoneIndexCounter[message.FromPhoneNumber]
			phoneIndexCounter[message.FromPhoneNumber]++
		}

		go func(message *requests.BulkMessage, index int) {
			count.Add(1)
			_, err = h.messageService.SendMessage(
				ctx,
				message.ToMessageSendParams(h.userIDFomContext(c), requestID, c.OriginalURL(), index, userLocation),
			)
			if err != nil {
				count.Add(-1)
				msg := fmt.Sprintf("cannot send message with payload [%s] at index [%d]", spew.Sdump(message), index)
				ctxLogger.Error(stacktrace.Propagate(err, msg))
			}
			wg.Done()
		}(message, perPhoneIndex)
	}

	wg.Wait()
	return h.responseAccepted(c, fmt.Sprintf("Added %d out of %d messages to the queue", count.Load(), len(messages)))
}

func (h *BulkMessageHandler) generateRequestID(filename string) string {
	return fmt.Sprintf("bulk-%s-%s", encodeBase62(time.Now().UnixMilli()), truncateFilename(sanitizeFilename(filename), 32))
}

func sanitizeFilename(filename string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9.\-_: ]`).ReplaceAllString(filename, "")
}

func encodeBase62(n int64) string {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}
	result := make([]byte, 0, 8)
	for n > 0 {
		result = append(result, charset[n%62])
		n /= 62
	}
	// reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

func truncateFilename(filename string, maxLen int) string {
	if len(filename) <= maxLen {
		return filename
	}
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]
	available := maxLen - len(ext)
	if available <= 0 {
		return filename[:maxLen]
	}
	half := available / 2
	return name[:half] + name[len(name)-(available-half):] + ext
}
