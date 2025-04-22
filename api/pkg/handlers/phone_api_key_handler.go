package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// PhoneAPIKeyHandler handles phone API key http requests
type PhoneAPIKeyHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.PhoneAPIKeyHandlerValidator
	service   *services.PhoneAPIKeyService
}

// NewPhoneAPIKeyHandler creates a new PhoneAPIKeyHandler
func NewPhoneAPIKeyHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.PhoneAPIKeyHandlerValidator,
	service *services.PhoneAPIKeyService,
) *PhoneAPIKeyHandler {
	return &PhoneAPIKeyHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", &PhoneAPIKeyHandler{})),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the PhoneAPIKeyHandler
func (h *PhoneAPIKeyHandler) RegisterRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	router := app.Group("/v1/api-keys/")
	router.Post("/", h.computeRoute(middlewares, h.Store)...)
	router.Delete("/:phoneAPIKeyID", h.computeRoute(middlewares, h.Delete)...)
	router.Delete("/:phoneAPIKeyID/phones/:phoneID", h.computeRoute(middlewares, h.DeletePhone)...)
}

// Store a new Phone API key
// @Summary      Store phone API key
// @Description  Creates a new phone API key which can be used to log in to the httpSMS app on your Android phone
// @Security	 ApiKeyAuth
// @Tags         PhoneAPIKeys
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.PhoneAPIKeyStoreRequest 	true 	"Payload of new phone API key."
// @Success      200 		{object}	responses.Ok[*entities.PhoneAPIKey]
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /api-keys [post]
func (h *PhoneAPIKeyHandler) Store(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.PhoneAPIKeyStoreRequest
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateStore(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while updating phones [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating phones")
	}

	phone, err := h.service.Create(ctx, h.userFromContext(c), request.Name)
	if err != nil {
		msg := fmt.Sprintf("cannot update phones with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "phone updated successfully", phone)
}

// Delete a phone API Key
// @Summary      Delete a phone API key from the database.
// @Description  Delete a phone API Key from the database and cannot be used for authentication anymore.
// @Security	 ApiKeyAuth
// @Tags         PhoneAPIKeys
// @Accept       json
// @Produce      json
// @Param 		 phoneAPIKeyID 	path		string 							true 	"ID of the phone API key" 	default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Success      204  		{object} 	responses.NoContent
// @Failure      400  		{object}  	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure 	 404		{object}	responses.NotFound
// @Failure      422  		{object} 	responses.UnprocessableEntity
// @Failure      500  		{object}  	responses.InternalServerError
// @Router       /messages/{phoneAPIKeyID} [delete]
func (h *PhoneAPIKeyHandler) Delete(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	phoneAPIKeyID := c.Params("phoneAPIKeyID")
	if errors := h.validator.ValidateUUID(phoneAPIKeyID, "phoneAPIKeyID"); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while deleting a phone API key with ID [%s]", spew.Sdump(errors), phoneAPIKeyID)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while storing event")
	}

	err := h.service.Delete(ctx, h.userIDFomContext(c), uuid.MustParse(phoneAPIKeyID))
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, fmt.Sprintf("cannot find phone API key with ID [%s]", phoneAPIKeyID))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot delete phone API key with ID [%s] for user with ID [%s]", phoneAPIKeyID, h.userIDFomContext(c))
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseInternalServerError(c)
	}

	return h.responseNoContent(c, "phone API key deleted successfully")
}

// DeletePhone removes a phone from a phone API key
// @Summary      Remove the association of a phone from the phone API key.
// @Description  You will need to login again to the httpSMS app on your Android phone with a new phone API key.
// @Security	 ApiKeyAuth
// @Tags         PhoneAPIKeys
// @Accept       json
// @Produce      json
// @Param 		 phoneAPIKeyID 	path		string 							true 	"ID of the phone API key" 	default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Param 		 phoneID 		path		string 							true 	"ID of the phone" 			default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Success      204  			{object} 	responses.NoContent
// @Failure      400  			{object}  	responses.BadRequest
// @Failure 	 401    		{object}	responses.Unauthorized
// @Failure 	 404			{object}	responses.NotFound
// @Failure      422  			{object} 	responses.UnprocessableEntity
// @Failure      500  			{object}  	responses.InternalServerError
// @Router       /messages/{phoneAPIKeyID}/phones/{phoneID} [delete]
func (h *PhoneAPIKeyHandler) DeletePhone(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	phoneAPIKeyID := c.Params("phoneAPIKeyID")
	phoneID := c.Params("phoneID")
	if errors := h.mergeErrors(h.validator.ValidateUUID(phoneAPIKeyID, "phoneAPIKeyID"), h.validator.ValidateUUID(phoneID, "phoneID")); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while deleting a phone API key with ID [%s]", spew.Sdump(errors), phoneAPIKeyID)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while storing event")
	}

	err := h.service.RemovePhone(ctx, h.userIDFomContext(c), uuid.MustParse(phoneAPIKeyID), uuid.MustParse(phoneID))
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, fmt.Sprintf("cannot find phone with ID [%s] which is associated with phone API key with ID [%s]", phoneID, phoneAPIKeyID))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot remove phone with ID [%s] from phone API key with ID [%s] for user with ID [%s]", phoneID, phoneAPIKeyID, h.userIDFomContext(c))
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseInternalServerError(c)
	}

	return h.responseNoContent(c, "phone has been dissociated from phone API key successfully")
}
