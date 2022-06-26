package handlers

import (
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/services"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// UserHandler handles user http requests.
type UserHandler struct {
	handler
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.UserService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.UserService,
) (h *UserHandler) {
	return &UserHandler{
		logger:  logger.WithService(fmt.Sprintf("%T", h)),
		tracer:  tracer,
		service: service,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *UserHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/users/me", h.Show)
}

// Show returns an entities.User
// @Summary      Get current user
// @Description  Get details of the currently authenticated user
// @Security	 ApiKeyAuth
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200 	{object}		responses.UserResponse
// @Failure      400	{object}		responses.BadRequest
// @Failure 	 403    {object}		responses.Unauthorized
// @Failure      422	{object}		responses.UnprocessableEntity
// @Failure      500	{object}		responses.InternalServerError
// @Router       /users/me [get]
func (h *UserHandler) Show(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	authUser := h.userFromContext(c)

	user, err := h.service.Get(ctx, authUser)
	if err != nil {
		msg := fmt.Sprintf("cannot get user with ID [%s]", authUser.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "user fetched successfully", user)
}
