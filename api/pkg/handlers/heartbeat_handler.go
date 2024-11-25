package handlers

import (
	"fmt"
	"sync"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// HeartbeatHandler handles heartbeat http requests.
type HeartbeatHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.HeartbeatHandlerValidator
	service   *services.HeartbeatService
}

// NewHeartbeatHandler creates a new HeartbeatHandler
func NewHeartbeatHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.HeartbeatHandlerValidator,
	service *services.HeartbeatService,
) (h *HeartbeatHandler) {
	return &HeartbeatHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *HeartbeatHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/heartbeats", h.Index)
	router.Post("/heartbeats", h.Store)
}

// Index returns the heartbeats of a phone number
// @Summary      Get heartbeats of an owner phone number
// @Description  Get the last time a phone number requested for outstanding messages. It will be sorted by timestamp in descending order.
// @Security	 ApiKeyAuth
// @Tags         Heartbeats
// @Accept       json
// @Produce      json
// @Param        owner		query  string  	true 	"the owner's phone number" 			default(+18005550199)
// @Param        skip		query  int  	false	"number of heartbeats to skip"		minimum(0)
// @Param        query		query  string  	false 	"filter containing query"
// @Param        limit		query  int  	false	"number of heartbeats to return"	minimum(1)	maximum(20)
// @Success      200 		{object}	responses.HeartbeatsResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /heartbeats [get]
func (h *HeartbeatHandler) Index(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.HeartbeatIndex
	if err := c.QueryParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateIndex(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while fetching heartbeats [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while fetching heartbeats")
	}

	heartbeats, err := h.service.Index(ctx, h.userIDFomContext(c), request.Owner, request.ToIndexParams())
	if err != nil {
		msg := fmt.Sprintf("cannot get messgaes with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, fmt.Sprintf("fetched %d %s", len(*heartbeats), h.pluralize("heartbeat", len(*heartbeats))), heartbeats)
}

// Store the heartbeat of a phone number
// @Summary      Register heartbeat of an owner phone number
// @Description  Store the heartbeat to make notify that a phone number is still active
// @Security	 ApiKeyAuth
// @Tags         Heartbeats
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.HeartbeatStore  		true "Payload of the heartbeat request"
// @Success      200 		{object}	responses.HeartbeatResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /heartbeats [post]
func (h *HeartbeatHandler) Store(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.HeartbeatStore
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateStore(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while storing heartbeat [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while storing heartbeat")
	}

	params := request.ToStoreParams(h.userFromContext(c), c.OriginalURL(), c.Get("X-Client-Version"))

	wg := sync.WaitGroup{}
	responses := make([]*entities.Heartbeat, len(params))
	for index, value := range params {
		wg.Add(1)
		go func(input services.HeartbeatStoreParams, index int) {
			response, err := h.service.Store(ctx, input)
			if err != nil {
				msg := fmt.Sprintf("cannot store heartbeat with params [%+#v]", request)
				ctxLogger.Error(stacktrace.Propagate(err, msg))
			}
			responses[index] = response
			wg.Done()
		}(value, index)
	}

	wg.Wait()
	return h.responseCreated(c, fmt.Sprintf("[%d] heartbeats received successfully", len(responses)), responses)
}
