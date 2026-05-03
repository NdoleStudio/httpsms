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

// MessageSendScheduleHandler handles HTTP requests for message send schedules.
type MessageSendScheduleHandler struct {
	handler
	logger             telemetry.Logger
	tracer             telemetry.Tracer
	validator          *validators.MessageSendScheduleHandlerValidator
	service            *services.MessageSendScheduleService
	entitlementService *services.EntitlementService
}

// NewMessageSendScheduleHandler creates a new MessageSendScheduleHandler.
func NewMessageSendScheduleHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.MessageSendScheduleHandlerValidator,
	service *services.MessageSendScheduleService,
	entitlementService *services.EntitlementService,
) *MessageSendScheduleHandler {
	return &MessageSendScheduleHandler{
		logger:             logger.WithService(fmt.Sprintf("%T", &MessageSendScheduleHandler{})),
		tracer:             tracer,
		validator:          validator,
		service:            service,
		entitlementService: entitlementService,
	}
}

// RegisterRoutes registers send schedule routes.
func (h *MessageSendScheduleHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	router.Get("/v1/send-schedules", h.computeRoute(middlewares, h.Index)...)
	router.Post("/v1/send-schedules", h.computeRoute(middlewares, h.Store)...)
	router.Put("/v1/send-schedules/:scheduleID", h.computeRoute(middlewares, h.Update)...)
	router.Delete("/v1/send-schedules/:scheduleID", h.computeRoute(middlewares, h.Delete)...)
}

// Index lists all send schedules for the authenticated user.
//
// @Summary List send schedules
// @Description List all send schedules owned by the authenticated user.
// @Security ApiKeyAuth
// @Tags Send Schedules
// @Produce json
// @Success 200 {object} responses.MessageSendSchedulesResponse
// @Failure 401 {object} responses.Unauthorized
// @Failure 500 {object} responses.InternalServerError
// @Router /send-schedules [get]
func (h *MessageSendScheduleHandler) Index(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	schedules, err := h.service.Index(ctx, h.userIDFomContext(c))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot list send schedules"))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "send schedules fetched successfully", schedules)
}

// Store creates a new send schedule for the authenticated user.
//
// @Summary Create send schedule
// @Description Create a new send schedule for the authenticated user.
// @Security ApiKeyAuth
// @Tags Send Schedules
// @Accept json
// @Produce json
// @Param payload body requests.MessageSendScheduleStore true "Payload of new send schedule."
// @Success 201 {object} responses.MessageSendScheduleResponse
// @Failure 400 {object} responses.BadRequest
// @Failure 401 {object} responses.Unauthorized
// @Failure 402 {object} responses.PaymentRequired
// @Failure 422 {object} responses.UnprocessableEntity
// @Failure 500 {object} responses.InternalServerError
// @Router /send-schedules [post]
func (h *MessageSendScheduleHandler) Store(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	userID := h.userIDFomContext(c)

	count, err := h.service.CountByUser(ctx, userID)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot count send schedules for entitlement check"))
		return h.responseInternalServerError(c)
	}

	result, err := h.entitlementService.Check(ctx, userID, "MessageSendSchedule", count)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot check entitlement for send schedules"))
		return h.responseInternalServerError(c)
	}
	if !result.Allowed {
		return h.responsePaymentRequired(c, result.Message)
	}

	var request requests.MessageSendScheduleStore
	if err := c.BodyParser(&request); err != nil {
		return h.responseBadRequest(c, err)
	}

	request = request.Sanitize()
	if errors := h.validator.ValidateStore(ctx, request); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewError(
			"validation errors [%s], while storing send schedule [%+#v]",
			spew.Sdump(errors),
			request,
		))
		return h.responseUnprocessableEntity(c, errors, "validation errors while saving send schedule")
	}

	schedule, err := h.service.Store(ctx, request.ToParams(h.userFromContext(c)))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot create send schedule"))
		return h.responseInternalServerError(c)
	}

	return h.responseCreated(c, "send schedule created successfully", schedule)
}

// Update updates a send schedule owned by the authenticated user.
//
// @Summary Update send schedule
// @Description Update a send schedule owned by the authenticated user.
// @Security ApiKeyAuth
// @Tags Send Schedules
// @Accept json
// @Produce json
// @Param scheduleID path string true "Schedule ID"
// @Param payload body requests.MessageSendScheduleStore true "Payload of updated send schedule."
// @Success 200 {object} responses.MessageSendScheduleResponse
// @Failure 400 {object} responses.BadRequest
// @Failure 401 {object} responses.Unauthorized
// @Failure 404 {object} responses.NotFound
// @Failure 422 {object} responses.UnprocessableEntity
// @Failure 500 {object} responses.InternalServerError
// @Router /send-schedules/{scheduleID} [put]
func (h *MessageSendScheduleHandler) Update(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	scheduleID, err := uuid.Parse(c.Params("scheduleID"))
	if err != nil {
		return h.responseBadRequest(c, err)
	}

	var request requests.MessageSendScheduleStore
	if err = c.BodyParser(&request); err != nil {
		return h.responseBadRequest(c, err)
	}

	request = request.Sanitize()
	if errors := h.validator.ValidateStore(ctx, request); len(errors) != 0 {
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating send schedule")
	}

	schedule, err := h.service.Update(
		ctx,
		h.userIDFomContext(c),
		scheduleID,
		request.ToParams(h.userFromContext(c)),
	)
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot update send schedule"))
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			return h.responseNotFound(c, err.Error())
		}
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "send schedule updated successfully", schedule)
}

// Delete removes a send schedule owned by the authenticated user.
//
// @Summary Delete send schedule
// @Description Delete a send schedule owned by the authenticated user.
// @Security ApiKeyAuth
// @Tags Send Schedules
// @Produce json
// @Param scheduleID path string true "Schedule ID"
// @Success 204
// @Failure 400 {object} responses.BadRequest
// @Failure 401 {object} responses.Unauthorized
// @Failure 404 {object} responses.NotFound
// @Failure 500 {object} responses.InternalServerError
// @Router /send-schedules/{scheduleID} [delete]
func (h *MessageSendScheduleHandler) Delete(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	scheduleID, err := uuid.Parse(c.Params("scheduleID"))
	if err != nil {
		return h.responseBadRequest(c, err)
	}

	if _, err = h.service.Load(ctx, h.userIDFomContext(c), scheduleID); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot load send schedule for deletion"))
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			return h.responseNotFound(c, err.Error())
		}
		return h.responseInternalServerError(c)
	}

	if err = h.service.Delete(ctx, h.userIDFomContext(c), scheduleID); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot delete send schedule"))
		if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
			return h.responseNotFound(c, err.Error())
		}
		return h.responseInternalServerError(c)
	}

	return h.responseNoContent(c, "send schedule deleted successfully")
}
