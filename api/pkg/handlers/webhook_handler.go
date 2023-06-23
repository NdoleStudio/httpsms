package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/repositories"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// WebhookHandler handles webhook requests
type WebhookHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	service   *services.WebhookService
	validator *validators.WebhookHandlerValidator
}

// NewWebhookHandler creates a new WebhookHandler
func NewWebhookHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.WebhookService,
	validator *validators.WebhookHandlerValidator,
) (h *WebhookHandler) {
	return &WebhookHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		service:   service,
		validator: validator,
	}
}

// RegisterRoutes registers the routes for the WebhookHandler
func (h *WebhookHandler) RegisterRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	router := app.Group("/v1/webhooks")
	router.Get("/", h.computeRoute(middlewares, h.Index)...)
	router.Post("/", h.computeRoute(middlewares, h.Store)...)
	router.Put("/:webhookID", h.computeRoute(middlewares, h.Update)...)
	router.Delete("/:webhookID", h.computeRoute(middlewares, h.Delete)...)
}

// Index returns the webhooks of a user
// @Summary      Get webhooks of a user
// @Description  Get the webhooks of a user
// @Security	 ApiKeyAuth
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Param        skip		query  int  	false	"number of webhooks to skip"		minimum(0)
// @Param        query		query  string  	false 	"filter webhooks containing query"
// @Param        limit		query  int  	false	"number of webhooks to return"	minimum(1)	maximum(20)
// @Success      200 		{object}	responses.WebhooksResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /webhooks 	[get]
func (h *WebhookHandler) Index(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	var request requests.WebhookIndex
	if err := c.QueryParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall URL [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateIndex(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while fetching webhooks [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching webhooks")
	}

	webhooks, err := h.service.Index(ctx, h.userIDFomContext(c), request.ToIndexParams())
	if err != nil {
		msg := fmt.Sprintf("cannot get webhooks with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d %s", len(webhooks), h.pluralize("webhook", len(webhooks))), webhooks)
}

// Delete a webhook
// @Summary      Delete webhook
// @Description  Delete a webhook for a user
// @Security	 ApiKeyAuth
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Param 		 webhookID 	path		string 							true 	"ID of the webhook"	default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Success      204		{object}    responses.NoContent
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /webhooks/{webhookID} [delete]
func (h *WebhookHandler) Delete(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	webhookID := c.Params("webhookID")
	if errors := h.validator.ValidateUUID(ctx, webhookID, "webhookID"); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while deleting webhook with ID [%s]", spew.Sdump(errors), webhookID)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while deleting webhook")
	}

	err := h.service.Delete(ctx, h.userIDFomContext(c), uuid.MustParse(webhookID))
	if err != nil {
		msg := fmt.Sprintf("cannot delete webhook with ID [%+#v]", webhookID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "webhook deleted successfully", nil)
}

// Store a webhook
// @Summary      Store a webhook
// @Description  Store a webhook for the authenticated user
// @Security	 ApiKeyAuth
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.WebhookStore  		true "Payload of the webhook request"
// @Success      200 		{object}	responses.WebhookResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /webhooks [post]
func (h *WebhookHandler) Store(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.WebhookStore
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall body [%s] into [%T]", c.Body(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateStore(ctx, h.userIDFomContext(c), request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while storing webhook [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while storing webhook")
	}

	webhooks, err := h.service.Index(ctx, h.userIDFomContext(c), repositories.IndexParams{Skip: 0, Limit: 3})
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot index webhooks for user [%s]", h.userIDFomContext(c))))
		return h.responsePaymentRequired(c, "You can't create more than 1 webhook contact us to upgrade your account.")
	}

	if len(webhooks) > 1 {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] wants to create more than 2 webhooks", h.userIDFomContext(c))))
		return h.responsePaymentRequired(c, "You can't create more than 2 webhooks contact us to upgrade your account.")
	}

	webhook, err := h.service.Store(ctx, request.ToStoreParams(h.userFromContext(c)))
	if err != nil {
		msg := fmt.Sprintf("cannot store webhoook with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseCreated(c, "webhook created successfully", webhook)
}

// Update an entities.Webhook
// @Summary      Update a webhook
// @Description  Update a webhook for the currently authenticated user
// @Security	 ApiKeyAuth
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Param 		 webhookID	path		string 							true 	"ID of the webhook" 					default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Param        payload   	body 		requests.WebhookUpdate  		true 	"Payload of webhook details to update"
// @Success      200 		{object}	responses.WebhookResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /webhooks/{webhookID} 	[put]
func (h *WebhookHandler) Update(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	var request requests.WebhookUpdate
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into [%T]", c.Body(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	request.WebhookID = c.Params("webhookID")
	if errors := h.validator.ValidateUpdate(ctx, h.userIDFomContext(c), request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while updating user [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating webhook")
	}

	user, err := h.service.Update(ctx, request.ToUpdateParams(h.userFromContext(c)))
	if err != nil {
		msg := fmt.Sprintf("cannot update user with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "webhook updated successfully", user)
}
