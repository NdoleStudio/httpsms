package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"

	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/gofiber/fiber/v3"
)

// PhoneHandler handles phone http requests.
type PhoneHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	service   *services.PhoneService
	validator *validators.PhoneHandlerValidator
}

// NewPhoneHandler creates a new PhoneHandler
func NewPhoneHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.PhoneService,
	validator *validators.PhoneHandlerValidator,
) (h *PhoneHandler) {
	return &PhoneHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the PhoneHandler
func (h *PhoneHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	h.register(router, fiber.MethodGet, "/v1/phones", middlewares, h.Index)
	h.register(router, fiber.MethodPut, "/v1/phones", middlewares, h.Upsert)
	h.register(router, fiber.MethodDelete, "/v1/phones/:phoneID", middlewares, h.Delete)
}

// RegisterPhoneAPIKeyRoutes registers the routes for the PhoneHandler
func (h *PhoneHandler) RegisterPhoneAPIKeyRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	h.register(router, fiber.MethodPut, "/v1/phones/fcm-token", middlewares, h.UpsertFCMToken)
}

// Index returns the phones of a user
// @Summary      Get phones of a user
// @Description  Get list of phones which a user has registered on the http sms application
// @Security	 ApiKeyAuth
// @Tags         Phones
// @Accept       json
// @Produce      json
// @Param        skip		query  int  	false	"number of heartbeats to skip"		minimum(0)
// @Param        query		query  string  	false 	"filter phones containing query"
// @Param        limit		query  int  	false	"number of phones to return"		minimum(1)	maximum(20)
// @Success      200 		{object}	responses.PhonesResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /phones [get]
func (h *PhoneHandler) Index(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.PhoneIndex
	if err := c.Bind().Query(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot marshall params [%s] into %T", c.OriginalURL(), request))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateIndex(ctx, request.Sanitize()); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while fetching phones [%+#v]", spew.Sdump(errors), request))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching phones")
	}

	phones, err := h.service.Index(ctx, h.userFromContext(c), request.ToIndexParams())
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot index phones with params [%+#v]", request))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d %s", len(*phones), h.pluralize("phone", len(*phones))), phones)
}

// Upsert a phone
// @Summary      Upsert Phone
// @Description  Updates properties of a user's phone. If the phone with this number does not exist, a new one will be created. Think of this method like an 'upsert'
// @Security	 ApiKeyAuth
// @Tags         Phones
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.PhoneUpsert  			true 	"Payload of new phone number."
// @Success      200 		{object}	responses.PhoneResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /phones [put]
func (h *PhoneHandler) Upsert(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.PhoneUpsert
	if err := c.Bind().Body(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot marshall params [%s] into %T", c.OriginalURL(), request))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateUpsert(ctx, h.userIDFomContext(c), request.Sanitize()); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while updating phones [%+#v]", spew.Sdump(errors), request))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating phones")
	}

	phone, err := h.service.Upsert(ctx, request.ToUpsertParams(h.userFromContext(c), c.OriginalURL(), c.Body()))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot update phones with params [%+#v]", request))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "phone updated successfully", phone)
}

// Delete a phone
// @Summary      Delete Phone
// @Description  Delete a phone that has been sored in the database
// @Security	 ApiKeyAuth
// @Tags         Phones
// @Accept       json
// @Produce      json
// @Param 		 phoneID 	path		string 							true 	"ID of the phone"	default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Success      204		{object}    responses.NoContent
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /phones/{phoneID} [delete]
func (h *PhoneHandler) Delete(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	request := requests.PhoneDelete{PhoneID: c.Params("phoneID")}
	if errors := h.validator.ValidateDelete(ctx, request); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while deleting phone [%+#v]", spew.Sdump(errors), request))
		return h.responseUnprocessableEntity(c, errors, "validation errors while deleting phone")
	}

	err := h.service.Delete(ctx, c.OriginalURL(), h.userIDFomContext(c), request.PhoneIDUuid())
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, fmt.Sprintf("cannot find phone with ID [%s]", request.PhoneID))
	}
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot delete phones with params [%+#v]", request))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "phone deleted successfully", nil)
}

// UpsertFCMToken upserts the FCM token of a phone
// @Summary      Upserts the FCM token of a phone
// @Description  Updates the FCM token of a phone. If the phone with this number does not exist, a new one will be created. Think of this method like an 'upsert'
// @Security	 ApiKeyAuth
// @Tags         Phones
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.PhoneFCMToken  			true 	"Payload of new FCM token."
// @Success      200 		{object}	responses.PhoneResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /phones/fcm-token [put]
func (h *PhoneHandler) UpsertFCMToken(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.PhoneFCMToken
	if err := c.Bind().Body(&request); err != nil {
		ctxLogger.Warn(stacktrace.Propagatef(err, "cannot marshall params [%s] into %T", c.OriginalURL(), request))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateFCMToken(ctx, request.Sanitize()); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewErrorf("validation errors [%s], while updating phones [%+#v]", spew.Sdump(errors), request))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating phones")
	}

	phone, err := h.service.UpsertFCMToken(ctx, request.ToPhoneFCMTokenParams(h.userFromContext(c), c.OriginalURL()))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagatef(err, "cannot delete phones with params [%+#v]", request))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "FCM token updated successfully", phone)
}
