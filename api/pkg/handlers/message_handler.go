package handlers

import "github.com/gofiber/fiber/v2"

// MessageHandler handles message http requests.
type MessageHandler struct {
	handler
}

// NewMessageHandler creates a new MessageHandler
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{}
}

// Send a new entities.Message
// @Summary      Send a new SMS message
// @Description  Add a new SMS message to be sent by the android phone
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Success      200  {object}  responses.MessageResponse
// @Router       /messages/send [post]
func (handler *MessageHandler) Send(c *fiber.Ctx) error {
	return nil
}
