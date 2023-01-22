package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// LemonsqueezyHandler handles lemonsqueezy events
type LemonsqueezyHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	service   *services.LemonsqueezyService
	validator *validators.LemonsqueezyHandlerValidator
}

// NewLemonsqueezyHandler creates a new LemonsqueezyHandler
func NewLemonsqueezyHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.LemonsqueezyService,
	validator *validators.LemonsqueezyHandlerValidator,
) (h *LemonsqueezyHandler) {
	return &LemonsqueezyHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		service:   service,
		validator: validator,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *LemonsqueezyHandler) RegisterRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	router := app.Group("lemonsqueezy")
	router.Post("/event", h.computeRoute(middlewares, h.Event)...)
}

// Event consumes a lemonsqueezy event
// @Summary      Consume a lemonsqueezy event
// @Description  Publish a lemonsqueezy event to the registered listeners
// @Tags         Lemonsqueezy
// @Accept       json
// @Produce      json
// @Success      204 		{object}	responses.NoContent
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /lemonsqueezy/event [post]
func (h *LemonsqueezyHandler) Event(c *fiber.Ctx) error {
	ctx, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	signature := c.Get("X-Signature")
	if errors := h.validator.ValidateEvent(ctx, signature, c.Body()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while storing request [%s] and signature [%s]", spew.Sdump(errors), c.Body(), signature)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while storing lemonsqueezy event")
	}

	if err := h.handleRequest(ctx, c); err != nil {
		msg := fmt.Sprintf("cannot handle lemonsqueezy event [%s]", c.Body())
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseNoContent(c, "event consumed successfully")
}

func (h *LemonsqueezyHandler) handleRequest(ctx context.Context, c *fiber.Ctx) error {
	eventName := c.Get("X-Event-Name")
	switch eventName {
	case "subscription_created":
		var request lemonsqueezy.WebHookRequestSubscription
		err := json.Unmarshal(c.Body(), &request)
		if err != nil {
			return stacktrace.Propagate(err, fmt.Sprintf("cannot marshall [%s] to [%T]", c.Body(), request))
		}
		return h.service.HandleSubscriptionCreatedEvent(ctx, c.OriginalURL(), &request)
	case "subscription_cancelled":
		var request lemonsqueezy.WebHookRequestSubscription
		err := json.Unmarshal(c.Body(), &request)
		if err != nil {
			return stacktrace.Propagate(err, fmt.Sprintf("cannot marshall [%s] to [%T]", c.Body(), request))
		}
		return h.service.HandleSubscriptionCanceledEvent(ctx, c.OriginalURL(), &request)
	default:
		return stacktrace.NewError(fmt.Sprintf("invalid event [%s] received with request [%s]", eventName, c.Body()))
	}
}
