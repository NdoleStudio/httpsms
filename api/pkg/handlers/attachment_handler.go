package handlers

import (
	"fmt"
	"path/filepath"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// AttachmentHandler handles attachment download requests
type AttachmentHandler struct {
	handler
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	storage repositories.AttachmentRepository
}

// NewAttachmentHandler creates a new AttachmentHandler
func NewAttachmentHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	storage repositories.AttachmentRepository,
) (h *AttachmentHandler) {
	return &AttachmentHandler{
		logger:  logger.WithService(fmt.Sprintf("%T", h)),
		tracer:  tracer,
		storage: storage,
	}
}

// RegisterRoutes registers the routes for the AttachmentHandler (no auth middleware — public endpoint)
func (h *AttachmentHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/v1/attachments/:userID/:messageID/:attachmentIndex/:filename", h.GetAttachment)
}

// GetAttachment Downloads an attachment
// @Summary      Download a message attachment
// @Description  Download an MMS attachment by its path components
// @Tags         Attachments
// @Produce      application/octet-stream
// @Param        userID           path  string  true  "User ID"
// @Param        messageID        path  string  true  "Message ID"
// @Param        attachmentIndex  path  string  true  "Attachment index"
// @Param        filename         path  string  true  "Filename with extension"
// @Success      200  {file}  binary
// @Failure      404  {object}  responses.NotFound
// @Failure      500  {object}  responses.InternalServerError
// @Router       /v1/attachments/{userID}/{messageID}/{attachmentIndex}/{filename} [get]
func (h *AttachmentHandler) GetAttachment(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	userID := c.Params("userID")
	messageID := c.Params("messageID")
	attachmentIndex := c.Params("attachmentIndex")
	filename := c.Params("filename")

	path := fmt.Sprintf("attachments/%s/%s/%s/%s", userID, messageID, attachmentIndex, filename)

	ctxLogger.Info(fmt.Sprintf("downloading attachment from path [%s]", path))

	data, err := h.storage.Download(ctx, path)
	if err != nil {
		msg := fmt.Sprintf("cannot download attachment from path [%s]", path)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			return h.responseNotFound(c, "attachment not found")
		}
		return h.responseInternalServerError(c)
	}

	ext := filepath.Ext(filename)
	contentType := repositories.ContentTypeFromExtension(ext)

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", "attachment")
	c.Set("X-Content-Type-Options", "nosniff")

	return c.Send(data)
}
