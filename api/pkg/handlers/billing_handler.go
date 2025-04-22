package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// BillingHandler handles billing http requests.
type BillingHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.BillingHandlerValidator
	service   *services.BillingService
}

// NewBillingHandler creates a new BillingHandler
func NewBillingHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.BillingHandlerValidator,
	service *services.BillingService,
) (h *BillingHandler) {
	return &BillingHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *BillingHandler) RegisterRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	router.Get("/v1/billing/usage-history", h.computeRoute(middlewares, h.UsageHistory)...)
	router.Get("/v1/billing/usage", h.computeRoute(middlewares, h.Usage)...)
}

// UsageHistory returns the usage history of a user
// @Summary      Get billing usage history.
// @Description  Get billing usage records of sent and received messages for a user in the past. It will be sorted by timestamp in descending order.
// @Security	 ApiKeyAuth
// @Tags         Billing
// @Accept       json
// @Produce      json
// @Param        skip		query  int  	false	"number of heartbeats to skip"		minimum(0)
// @Param        limit		query  int  	false	"number of heartbeats to return"	minimum(1)	maximum(100)
// @Success      200 		{object}	responses.BillingUsagesResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /billing/usage-history [get]
func (h *BillingHandler) UsageHistory(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.BillingUsageHistory
	if err := c.QueryParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateHistory(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while fetching heartbeats [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching usage history")
	}

	heartbeats, err := h.service.GetUsageHistory(ctx, h.userIDFomContext(c), request.ToIndexParams())
	if err != nil {
		msg := fmt.Sprintf("cannot get billing usage history with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d billing usage %s", len(*heartbeats), h.pluralize("record", len(*heartbeats))), heartbeats)
}

// Usage returns the current usage history of a user
// @Summary      Get Billing Usage.
// @Description  Get the summary of sent and received messages for a user in the current month
// @Security	 ApiKeyAuth
// @Tags         Billing
// @Accept       json
// @Produce      json
// @Success      200 		{object}	responses.BillingUsageResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /billing/usage [get]
func (h *BillingHandler) Usage(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	billingUsage, err := h.service.GetCurrentUsage(ctx, h.userIDFomContext(c))
	if err != nil {
		msg := fmt.Sprintf("cannot get current usage record for user [%s]", h.userFromContext(c))
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "fetched current billing usage", billingUsage)
}
