package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlmjohnson/requests"

	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// MarketingService is handles marketing requests
type MarketingService struct {
	logger      telemetry.Logger
	tracer      telemetry.Tracer
	authClient  *auth.Client
	brevoAPIKey string
}

// NewMarketingService creates a new instance of the MarketingService
func NewMarketingService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	authClient *auth.Client,
	brevoAPIKey string,
) *MarketingService {
	return &MarketingService{
		logger:      logger.WithService(fmt.Sprintf("%T", &MarketingService{})),
		tracer:      tracer,
		authClient:  authClient,
		brevoAPIKey: brevoAPIKey,
	}
}

// DeleteUser a user if exists in the sendgrid list
func (service *MarketingService) DeleteUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	err := requests.URL(fmt.Sprintf("https://api.brevo.com/v3/contacts/%s?identifierType=ext_id", userID)).
		Header("api-key", service.brevoAPIKey).
		Delete().
		CheckStatus(fiber.StatusNoContent).
		Fetch(ctx)
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot delete user with id [%s] from brevo list", userID)))
	}

	ctxLogger.Info(fmt.Sprintf("deleted user with ID [%s] from brevo list with status [%s]", userID, fiber.StatusNoContent))
	return nil
}

// AddToList adds a new user on the onboarding automation.
func (service *MarketingService) AddToList(ctx context.Context, user *entities.User) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	userRecord, err := service.authClient.GetUser(ctx, string(user.ID))
	if err != nil {
		msg := fmt.Sprintf("cannot get auth user with id [%s]", user.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return
	}

	var response string
	err = requests.URL("https://api.brevo.com/v3/contacts").
		Header("api-key", service.brevoAPIKey).
		Post().
		BodyJSON(fiber.Map{
			"email":         userRecord.Email,
			"ext_id":        userRecord.UID,
			"attributes":    service.brevoAttributes(userRecord),
			"listIds":       []int64{9},
			"updateEnabled": true,
		}).
		CheckStatus(fiber.StatusCreated, fiber.StatusNoContent).
		ToString(&response).
		Fetch(ctx)
	if err != nil {
		msg := fmt.Sprintf("cannot add user with id [%s] to brevo list", user.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return
	}

	ctxLogger.Info(fmt.Sprintf("user [%s] added to list brevo list with brevo response [%s]", user.ID, response))
}

func (service *MarketingService) brevoAttributes(user *auth.UserRecord) map[string]any {
	name := strings.TrimSpace(user.DisplayName)
	if name == "" {
		return fiber.Map{}
	}

	parts := strings.Split(name, " ")
	if len(parts) == 1 {
		return fiber.Map{"FIRSTNAME": name}
	}

	return fiber.Map{
		"FIRSTNAME": strings.Join(parts[0:len(parts)-1], " "),
		"LASTNAME":  parts[len(parts)-1],
	}
}
