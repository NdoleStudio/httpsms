package handlers

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/httpsms/pkg/validators"
	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// DiscordHandler handles discord events
type DiscordHandler struct {
	handler
	logger    telemetry.Logger
	tracer    telemetry.Tracer
	validator *validators.DiscordHandlerValidator
	service   *services.DiscordService
}

// NewDiscordHandler creates a new DiscordHandler
func NewDiscordHandler(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	validator *validators.DiscordHandlerValidator,
	service *services.DiscordService,
) (h *DiscordHandler) {
	return &DiscordHandler{
		logger:    logger.WithService(fmt.Sprintf("%T", h)),
		tracer:    tracer,
		validator: validator,
		service:   service,
	}
}

// RegisterRoutes registers the routes for the MessageHandler
func (h *DiscordHandler) RegisterRoutes(app *fiber.App, authMiddleware fiber.Handler, middlewares ...fiber.Handler) {
	router := app.Group("discord")
	router.Post("/event", h.computeRoute(middlewares, h.Event)...)

	authRouter := app.Group("v1/discord")
	authRouter.Post("/", h.computeRoute(append(middlewares, authMiddleware), h.Event)...)
}

// Store a webhook
// @Summary      Store a webhook
// @Description  Store a webhook for the authenticated user
// @Security	 ApiKeyAuth
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Param        payload   	body 		requests.WebhookStore  		true "Payload of the webhook request"
// @Success      200 		{object}	responses.WebhookResponse
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401	    {object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /webhooks [post]
func (h *DiscordHandler) Store(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartFromFiberCtx(c)
	defer span.End()

	ctxLogger := h.tracer.CtxLogger(h.logger, span)

	var request requests.DiscordStore
	if err := c.BodyParser(&request); err != nil {
		msg := fmt.Sprintf("cannot marshall body [%s] into [%T]", c.Body(), request)
		ctxLogger.Warn(stacktrace.Propagate(err, msg))
		return h.responseBadRequest(c, err)
	}

	if errors := h.validator.ValidateStore(ctx, request.Sanitize()); len(errors) != 0 {
		msg := fmt.Sprintf("validation errors [%s], while storing discord integration [%+#v]", spew.Sdump(errors), request)
		ctxLogger.Warn(stacktrace.NewError(msg))
		return h.responseUnprocessableEntity(c, errors, "validation errors while storing discord integration")
	}

	discordIntegrations, err := h.service.Index(ctx, h.userIDFomContext(c), repositories.IndexParams{Skip: 0, Limit: 1})
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot index discord integrations for user [%s]", h.userIDFomContext(c))))
		return h.responseInternalServerError(c)
	}

	if len(discordIntegrations) > 0 {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("user with ID [%s] wants to create more than 1 discord integration", h.userIDFomContext(c))))
		return h.responsePaymentRequired(c, "You can't create more than 1 discord integration contact us to upgrade your account.")
	}

	discordIntegration, err := h.service.Store(ctx, request.ToStoreParams(h.userFromContext(c)))
	if err != nil {
		msg := fmt.Sprintf("cannot store discord integration with params [%+#v]", request)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return h.responseInternalServerError(c)
	}

	return h.responseCreated(c, "discord integration created successfully", discordIntegration)
}

// Event consumes a discord event
// @Summary      Consume a discord event
// @Description  Publish a discord event to the registered listeners
// @Tags         Discord
// @Accept       json
// @Produce      json
// @Success      204 		{object}	responses.NoContent
// @Failure      400		{object}	responses.BadRequest
// @Failure 	 401    	{object}	responses.Unauthorized
// @Failure      422		{object}	responses.UnprocessableEntity
// @Failure      500		{object}	responses.InternalServerError
// @Router       /discord/event [post]
func (h *DiscordHandler) Event(c *fiber.Ctx) error {
	_, span, ctxLogger := h.tracer.StartFromFiberCtxWithLogger(c, h.logger)
	defer span.End()

	if verified := h.verifyInteraction(ctxLogger, c); !verified {
		return h.responseUnauthorized(c)
	}

	var payload map[string]any
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		msg := fmt.Sprintf("cannot unmarshall [%s] to [%T]", string(c.Body()), payload)
		ctxLogger.Error(h.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
		return h.responseBadRequest(c, err)
	}

	ctxLogger.Info(string(c.Body()))

	if payload["type"].(float64) == 1 {
		return c.JSON(fiber.Map{"type": 1})
	}

	if payload["type"].(float64) == 2 {
		return c.JSON(
			fiber.Map{
				"type": 4,
				//"data": fiber.Map{
				//	"content": "✔ sending sms*",
				//},
				"data": fiber.Map{
					"content": "*⚠ could not send SMS message*",
					"embeds": []fiber.Map{
						{
							"title": "The to field is not a valid phone number",
							"color": 14681092,
						},
						{
							"title": "The from field is not a valid phone number",
							"color": 14681092,
						},
						{
							"fields": []fiber.Map{
								{
									"name":   "From:",
									"value":  "+37259139660",
									"inline": true,
								},
								{
									"name":   "To:",
									"value":  "+37259139661",
									"inline": true,
								},
								{
									"name":  "Content:",
									"value": "Hello World",
								},
							},
						},
					},
				},
			},
		)
	}

	return h.responseBadRequest(c, stacktrace.NewError(fmt.Sprintf("unknown type [%d]", payload["type"])))
}

// verifyInteraction implements message verification of the discord interactions api
// signing algorithm, as documented here:
// https://discord.com/developers/docs/interactions/receiving-and-responding#security-and-authorization
func (h *DiscordHandler) verifyInteraction(ctxLogger telemetry.Logger, c *fiber.Ctx) bool {
	var msg bytes.Buffer

	signature := c.Get("X-Signature-Ed25519")
	if signature == "" {
		ctxLogger.Info("X-Signature-Ed25519 header is empty")
		return false
	}

	sig, err := hex.DecodeString(signature)
	if err != nil {
		ctxLogger.Info(fmt.Sprintf("cannot decode X-Signature-Ed25519 [%s]", signature))
		return false
	}

	if len(sig) != ed25519.SignatureSize {
		ctxLogger.Info(fmt.Sprintf("invalid signature size [%d]", len(sig)))
		return false
	}

	timestamp := c.Get("X-Signature-Timestamp")
	if timestamp == "" {
		ctxLogger.Info("X-Signature-Timestamp header is empty")
		return false
	}

	msg.WriteString(timestamp)
	msg.Write(c.Body())

	key, err := hex.DecodeString(os.Getenv("DISCORD_PUBLIC_KEY"))
	if err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, "cannot decode DISCORD_PUBLIC_KEY env variable [%s]", os.Getenv("DISCORD_PUBLIC_KEY")))
		return false
	}

	return ed25519.Verify(key, msg.Bytes(), sig)
}
