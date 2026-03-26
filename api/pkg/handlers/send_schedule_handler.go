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

type SendScheduleHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.SendScheduleHandlerValidator
	service   *services.SendScheduleService
}

func NewSendScheduleHandler(logger telemetry.Logger, tracer telemetry.Tracer, validator *validators.SendScheduleHandlerValidator, service *services.SendScheduleService) *SendScheduleHandler {
	return &SendScheduleHandler{logger: logger.WithService(fmt.Sprintf("%T", &SendScheduleHandler{})), tracer: tracer, validator: validator, service: service}
}

func (h *SendScheduleHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	router.Get("/v1/send-schedules", h.computeRoute(middlewares, h.Index)...)
	router.Post("/v1/send-schedules", h.computeRoute(middlewares, h.Create)...)
	router.Get("/v1/send-schedules/:scheduleID", h.computeRoute(middlewares, h.Show)...)
	router.Put("/v1/send-schedules/:scheduleID", h.computeRoute(middlewares, h.Update)...)
	router.Delete("/v1/send-schedules/:scheduleID", h.computeRoute(middlewares, h.Delete)...)
	router.Post("/v1/send-schedules/:scheduleID/default", h.computeRoute(middlewares, h.SetDefault)...)
}

func (h *SendScheduleHandler) Index(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()
	items, err := h.service.Index(ctx, h.userIDFomContext(c))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot list send schedules"))
		return h.responseInternalServerError(c)
	}
	return h.responseOK(c, "send schedules fetched successfully", items)
}

func (h *SendScheduleHandler) Show(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()
	scheduleID, err := uuid.Parse(c.Params("scheduleID"))
	if err != nil {
		return h.responseBadRequest(c, err)
	}
	item, err := h.service.Load(ctx, h.userIDFomContext(c), scheduleID)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, "send schedule not found")
	}
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot load send schedule"))
		return h.responseInternalServerError(c)
	}
	return h.responseOK(c, "send schedule fetched successfully", item)
}

func (h *SendScheduleHandler) Create(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()
	ctxLogger := h.tracer.CtxLogger(h.logger, span)
	var request requests.SendScheduleUpsert
	if err := c.BodyParser(&request); err != nil {
		return h.responseBadRequest(c, err)
	}
	if errors := h.validator.ValidateUpsert(ctx, request.Sanitize()); len(errors) != 0 {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("validation errors [%s]", spew.Sdump(errors))))
		return h.responseUnprocessableEntity(c, errors, "validation errors while creating send schedule")
	}
	item, err := h.service.Create(ctx, h.userIDFomContext(c), request.ToUpsertParams())
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot create send schedule"))
		return h.responseInternalServerError(c)
	}
	return h.responseCreated(c, "send schedule created successfully", item)
}

func (h *SendScheduleHandler) Update(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()
	ctxLogger := h.tracer.CtxLogger(h.logger, span)
	scheduleID, err := uuid.Parse(c.Params("scheduleID"))
	if err != nil {
		return h.responseBadRequest(c, err)
	}
	var request requests.SendScheduleUpsert
	if err := c.BodyParser(&request); err != nil {
		return h.responseBadRequest(c, err)
	}
	if errors := h.validator.ValidateUpsert(ctx, request.Sanitize()); len(errors) != 0 {
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating send schedule")
	}
	item, err := h.service.Update(ctx, h.userIDFomContext(c), scheduleID, request.ToUpsertParams())
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, "send schedule not found")
	}
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot update send schedule"))
		return h.responseInternalServerError(c)
	}
	return h.responseOK(c, "send schedule updated successfully", item)
}

func (h *SendScheduleHandler) Delete(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()
	scheduleID, err := uuid.Parse(c.Params("scheduleID"))
	if err != nil {
		return h.responseBadRequest(c, err)
	}
	if err := h.service.Delete(ctx, h.userIDFomContext(c), scheduleID); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot delete send schedule"))
		return h.responseInternalServerError(c)
	}
	return h.responseNoContent(c, "send schedule deleted successfully")
}

func (h *SendScheduleHandler) SetDefault(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()
	scheduleID, err := uuid.Parse(c.Params("scheduleID"))
	if err != nil {
		return h.responseBadRequest(c, err)
	}
	item, err := h.service.SetDefault(ctx, h.userIDFomContext(c), scheduleID)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		return h.responseNotFound(c, "send schedule not found")
	}
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot set default send schedule"))
		return h.responseInternalServerError(c)
	}
	return h.responseOK(c, "default send schedule updated successfully", item)
}
