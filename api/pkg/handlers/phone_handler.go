package handlers

import (
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/requests"
	"github.com/NdoleStudio/http-sms-manager/pkg/validators"
	"github.com/davecgh/go-spew/spew"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
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
func (h *PhoneHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/phones", h.Index)
	router.Put("/phones", h.Upsert)
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
func (h *PhoneHandler) Index(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.PhoneIndex
	if err := c.QueryParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateIndex(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while fetching phones [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching phones")
	}

	phones, err := h.service.Index(ctx, h.userFromContext(c), request.ToIndexParams())
	if err != nil {
		msg := fmt.Sprintf("cannot index phones with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
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
func (h *PhoneHandler) Upsert(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.PhoneUpsert
	if err := c.QueryParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateUpsert(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while fetching phones [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching phones")
	}

	phone, err := h.service.Upsert(ctx, request.ToUpsertParams(h.userFromContext(c)))
	if err != nil {
		msg := fmt.Sprintf("cannot update phones with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "phone updated successfully", phone)
}
