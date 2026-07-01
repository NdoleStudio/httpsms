package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// USSDHandler handles USSD http requests.
type USSDHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	service   *services.USSDService
	validator *validators.USSDHandlerValidator
}

// NewUSSDHandler creates a new USSDHandler
func NewUSSDHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.USSDService,
	validator *validators.USSDHandlerValidator,
) (h *USSDHandler) {
	return &USSDHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the USSDHandler
func (h *USSDHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	h.register(router, fiber.MethodPost, "/v1/ussd/receive", middlewares, h.Receive)
	h.register(router, fiber.MethodPost, "/v1/ussd/send", middlewares, h.Send)
	h.register(router, fiber.MethodGet, "/v1/ussd", middlewares, h.Index)
	h.register(router, fiber.MethodDelete, "/v1/ussd/:ussdID", middlewares, h.Delete)
}

// RegisterPhoneAPIKeyRoutes registers the phone API key routes for the USSDHandler
func (h *USSDHandler) RegisterPhoneAPIKeyRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	h.register(router, fiber.MethodPost, "/v1/ussd/receive", middlewares, h.Receive)
	h.register(router, fiber.MethodPost, "/v1/ussd/send", middlewares, h.Send)
}

// Receive handles incoming USSD requests from a mobile phone
// @Summary      Receive USSD request
// @Description  Receive a USSD request from a mobile phone
// @Security	 ApiKeyAuth
// @Tags         USSD
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.USSDReceive  			true 	"USSD request payload"
// @Success      200 		{object}	responses.USSDResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /ussd/receive [post]
func (h *USSDHandler) Receive(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.USSDReceive
	if err := c.Bind().Body(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateReceive(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while receiving USSD [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while receiving USSD")
	}

	authUser := h.userFromContext(c)

	ussd, err := h.service.Receive(ctx, request.ToUSSDReceiveParams(authUser.ID, c.OriginalURL()), uuid.Nil)
	if err != nil {
		msg := fmt.Sprintf("cannot receive USSD with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "USSD request received successfully", ussd)
}

// Send handles sending a USSD response to a mobile phone
// @Summary      Send USSD response
// @Description  Send a USSD response to a mobile phone
// @Security	 ApiKeyAuth
// @Tags         USSD
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.USSDSend  			true 	"USSD response payload"
// @Success      200 		{object}	responses.USSDResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /ussd/send [post]
func (h *USSDHandler) Send(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.USSDSend
	if err := c.Bind().Body(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateSend(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while sending USSD [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while sending USSD")
	}

	authUser := h.userFromContext(c)

	ussd, err := h.service.Send(ctx, request.ToUSSDSendParams(authUser.ID, c.OriginalURL()))
	if err != nil {
		msg := fmt.Sprintf("cannot send USSD with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "USSD response sent successfully", ussd)
}

// Index returns the USSD session history for a user
// @Summary      Get USSD session history
// @Description  Get list of USSD sessions for a user
// @Security	 ApiKeyAuth
// @Tags         USSD
// @Accept       json
// @Produce      json
// @Param        skip		query  int  	false	"number of USSD sessions to skip"		minimum(0)
// @Param        query		query  string  	false 	"filter USSD sessions containing query"
// @Param        limit		query  int  	false	"number of USSD sessions to return"		minimum(1)	maximum(20)
// @Param        phone_id	query  string  	false 	"filter USSD sessions by phone ID"
// @Success      200 		{object}	responses.USSDsResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /ussd [get]
func (h *USSDHandler) Index(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.USSDIndex
	if err := c.Bind().Query(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	sanitized := request.Sanitize()
	if errors := h.validator.ValidateIndex(ctx, sanitized); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while fetching USSD sessions [%+#v]", spew.Sdump(errors), sanitized)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching USSD sessions")
	}

	var phoneID *uuid.UUID
	if sanitized.PhoneID != "" {
		pid, err := uuid.Parse(sanitized.PhoneID)
		if err == nil {
			phoneID = &pid
		}
	}

	ussds, err := h.service.Index(ctx, h.userFromContext(c), sanitized.ToIndexParams(), phoneID)
	if err != nil {
		msg := fmt.Sprintf("cannot index USSD sessions with params [%+#v]", sanitized)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d %s", len(*ussds), h.pluralize("USSD session", len(*ussds))), ussds)
}

// Delete a USSD session
// @Summary      Delete USSD session
// @Description  Delete a USSD session from the database
// @Security	 ApiKeyAuth
// @Tags         USSD
// @Accept       json
// @Produce      json
// @Param 		 ussdID 	path		string 							true 	"ID of the USSD session"	default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Success      204		{object}    responses.NoContent
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /ussd/{ussdID} [delete]
func (h *USSDHandler) Delete(c fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	request := requests.USSDDelete{USSDID: c.Params("ussdID")}
	if errors := h.validator.ValidateDelete(ctx, request); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while deleting USSD session [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while deleting USSD session")
	}

	ussdID, err := uuid.Parse(request.USSDID)
	if err != nil {
		msg := fmt.Sprintf("invalid USSD session ID [%s]", request.USSDID)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	err = h.service.Delete(ctx, c.OriginalURL(), h.userIDFomContext(c), ussdID)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, fmt.Sprintf("cannot find USSD session with ID [%s]", request.USSDID))
	}
	if err != nil {
		msg := fmt.Sprintf("cannot delete USSD session with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "USSD session deleted successfully", nil)
}