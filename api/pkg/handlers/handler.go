package handlers

import (
	"net/url"

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
		"data":    nil,
	})
}

func (h *handler) responseForbidden(c *fiber.Ctx) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"status":  "error",
		"message": fiber.ErrForbidden.Message,
		"data":    nil,
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
		"data":    nil,
	})
}

func (h *handler) responseOK(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}
