package handlers

import (
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"

	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// UserHandler handles user http requests.
type UserHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.UserHandlerValidator
	service   *services.UserService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.UserHandlerValidator,
	service *services.UserService,
) (h *UserHandler) {
	return &UserHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *UserHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/users/me", h.Show)
	router.Put("/users/me", h.Update)
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
// @Failure 	 401    {object}		responses.Unauthorized
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

// Update an entities.User
// @Summary      Update a user
// @Description  Updates the details of the currently authenticated user
// @Security	 ApiKeyAuth
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.UserUpdate  			true 	"Payload of user details to update"
// @Success      200 		{object}	responses.PhoneResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /users/me [put]
func (h *UserHandler) Update(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.UserUpdate
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateUpdate(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while updating user [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while updating user")
	}

	user, err := h.service.Update(ctx, h.userFromContext(c), request.ToUpdateParams())
	if err != nil {
		msg := fmt.Sprintf("cannot update user with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "user updated successfully", user)
}
