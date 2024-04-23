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
	router.Delete("/users/:userID/api-keys", h.DeleteAPIKey)
	router.Put("/users/:userID/notifications", h.UpdateNotifications)
	router.Get("/users/subscription-update-url", h.subscriptionUpdateURL)
	router.Delete("/users/subscription", h.cancelSubscription)
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

// UpdateNotifications an entities.User
// @Summary      Update notification settings
// @Description  Update the email notification settings for a user
// @Security	 ApiKeyAuth
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param 		 userID 	path		string 							true 	"ID of the user to update" 				default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Param        payload   	body 		requests.UserNotificationUpdate	true 	"User notification details to update"
// @Success      200 		{object}	responses.UserResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /users/{userID}/notifications [put]
func (h *UserHandler) UpdateNotifications(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.UserNotificationUpdate
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall params [%s] into %T", c.OriginalURL(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	user, err := h.service.UpdateNotificationSettings(ctx, h.userIDFomContext(c), request.ToUserNotificationUpdateParams())
	if err != nil {
		msg := fmt.Sprintf("cannot update notification for [%T] with ID [%s]", user, h.userIDFomContext(c))
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "user notification settings updated successfully", user)
}

// subscriptionUpdateURL returns the subscription update URL for the authenticated entities.User
// @Summary      Currently authenticated user subscription update URL
// @Description  Fetches the subscription URL of the authenticated user.
// @Security	 ApiKeyAuth
// @Tags         Users
// @Produce      json
// @Success      200 		{object}	responses.OkString
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /users/subscription-update-url 	[get]
func (h *UserHandler) subscriptionUpdateURL(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)
	authUser := h.userFromContext(c)

	url, err := h.service.GetSubscriptionUpdateURL(ctx, authUser.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot get user with ID [%s]", authUser.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "Subscription update URL fetched successfully", url)
}

// cancelSubscription cancels the subscription for the authenticated entities.User
// @Summary      Cancel the user's subscription
// @Description  Cancel the subscription of the authenticated user.
// @Security	 ApiKeyAuth
// @Tags         Users
// @Produce      json
// @Success      200 		{object}	responses.NoContent
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /users/subscription 	[delete]
func (h *UserHandler) cancelSubscription(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)
	authUser := h.userFromContext(c)

	err := h.service.InitiateSubscriptionCancel(ctx, authUser.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot get user with ID [%s]", authUser.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseNoContent(c, "Subscription cancelled successfully")
}

// DeleteAPIKey rotates the API Key for a user
// @Summary      Rotate the user's API Key
// @Description  Rotate the user's API key in case the current API Key is compromised
// @Security	 ApiKeyAuth
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param 		 userID 	path		string 							true 	"ID of the user to update" 	default(32343a19-da5e-4b1b-a767-3298a73703ca)
// @Success      200 		{object}	responses.UserResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /users/{userID}/api-keys [delete]
func (h *UserHandler) DeleteAPIKey(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	if c.Params("userID") != string(h.userIDFomContext(c)) {
		return h.responseUnauthorized(c)
	}

	user, err := h.service.RotateAPIKey(ctx, c.OriginalURL(), h.userIDFomContext(c))
	if err != nil {
		msg := fmt.Sprintf("cannot rotate the api key for [%T] with ID [%s]", user, h.userIDFomContext(c))
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseInternalServerError(c)
	}

	return h.responseOK(c, "API Key rotated successfully", user)
}
