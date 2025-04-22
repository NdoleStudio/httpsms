package handlers

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/middlewares"

	"github.com/gofiber/fiber/v2"
)

// handler is the base struct for handling requests
type handler struct{}

func (h *handler) responseBadRequest(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"status":  "error",
		"message": "The request isn't properly formed",
		"data":    err,
	})
}

func (h *handler) responseInternalServerError(c *fiber.Ctx) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"status":  "error",
		"message": "We ran into an internal error while handling the request.",
	})
}

func (h *handler) responseUnauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"status":  "error",
		"message": "You are not authorized to carry out this request.",
		"data":    "Make sure your API key is set in the [X-API-Key] header in the request",
	})
}

func (h *handler) responsePhoneAPIKeyUnauthorized(c *fiber.Ctx, owner string, authCtx entities.AuthContext) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"status":  "error",
		"message": "You are not authorized to carry out the request for this phone number",
		"data":    fmt.Sprintf("The phone API key is does not have permission to carry out actions on the phone number [%s]. The API key is only configured for these phone numbers [%s]", owner, strings.Join(authCtx.PhoneNumbers, ",")),
	})
}

func (h *handler) responseForbidden(c *fiber.Ctx) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"status":  "error",
		"message": fiber.ErrForbidden.Message,
	})
}

func (h *handler) responseUnprocessableEntity(c *fiber.Ctx, errors url.Values, message string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
		"status":  "error",
		"message": message,
		"data":    errors,
	})
}

func (h *handler) responseNotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"status":  "error",
		"message": message,
	})
}

func (h *handler) responsePaymentRequired(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
		"status":  "error",
		"message": message,
	})
}

func (h *handler) responseNoContent(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNoContent).JSON(fiber.Map{
		"status":  "success",
		"message": message,
	})
}

func (h *handler) responseAccepted(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"status":  "success",
		"message": message,
	})
}

func (h *handler) responseOK(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

func (h *handler) responseCreated(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

func (h *handler) pluralize(value string, count int) string {
	if count == 1 {
		return value
	}
	return value + "s"
}

func (h *handler) userFromContext(c *fiber.Ctx) entities.AuthContext {
	if tokenUser, ok := c.Locals(middlewares.ContextKeyAuthUserID).(entities.AuthContext); ok && !tokenUser.IsNoop() {
		return tokenUser
	}
	panic("user does not exist in context.")
}

func (h *handler) userIDFomContext(c *fiber.Ctx) entities.UserID {
	return h.userFromContext(c).ID
}

func (h *handler) computeRoute(middlewares []fiber.Handler, route fiber.Handler) []fiber.Handler {
	return append(append([]fiber.Handler{}, middlewares...), route)
}

func (h *handler) mergeErrors(errors ...url.Values) url.Values {
	result := url.Values{}
	for _, item := range errors {
		for key, values := range item {
			for _, value := range values {
				result.Add(key, value)
			}
		}
	}
	return result
}

func (h *handler) authorizePhoneAPIKey(c *fiber.Ctx, phoneNumber string) bool {
	user := h.userFromContext(c)
	if user.PhoneAPIKeyID == nil {
		return true
	}
	return slices.Contains(user.PhoneNumbers, phoneNumber)
}
