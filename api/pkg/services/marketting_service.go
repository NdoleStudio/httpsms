package services

import (
	"context"
	"fmt"
	"strings"

	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	brevo "github.com/getbrevo/brevo-go/lib"
	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"
)

// MarketingService is handles marketing requests
type MarketingService struct {
	logger      telemetry.Logger
	tracer      telemetry.Tracer
	authClient  *auth.Client
	brevoClient *brevo.APIClient
}

// NewMarketingService creates a new instance of the MarketingService
func NewMarketingService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	authClient *auth.Client,
	brevoClient *brevo.APIClient,
) *MarketingService {
	return &MarketingService{
		logger:      logger.WithService(fmt.Sprintf("%T", &MarketingService{})),
		tracer:      tracer,
		authClient:  authClient,
		brevoClient: brevoClient,
	}
}

// DeleteUser a user if exists in the sendgrid list
func (service *MarketingService) DeleteUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if service.brevoClient == nil {
		ctxLogger.Warn(stacktrace.NewError("brevo client is not initialized, skipping adding user to list"))
		return
	}

	response, err := service.brevoClient.ContactsApi.DeleteContact(ctx, userID.String())
	if err != nil {
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot delete user with id [%s] from brevo list", userID)))
	}

	ctxLogger.Info(fmt.Sprintf("deleted user with ID [%s] from brevo list with status [%s]", userID, response.Status))
	return nil
}

// AddToList adds a new user on the onboarding automation.
func (service *MarketingService) AddToList(ctx context.Context, user *entities.User) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if service.brevoClient == nil {
		ctxLogger.Warn(stacktrace.NewError("brevo client is not initialized, skipping adding user to list"))
		return
	}

	userRecord, err := service.authClient.GetUser(ctx, string(user.ID))
	if err != nil {
		msg := fmt.Sprintf("cannot get auth user with id [%s]", user.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return
	}

	contact, _, err := service.brevoClient.ContactsApi.CreateContact(ctx, brevo.CreateContact{
		Email:         userRecord.Email,
		Attributes:    service.brevoAttributes(userRecord),
		ExtId:         userRecord.UID,
		ListIds:       []int64{9},
		UpdateEnabled: true,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot add user with id [%s] to brevo list", user.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return
	}

	ctxLogger.Info(fmt.Sprintf("user [%s] added to list brevo list with brevo ID [%d]", user.ID, contact.Id))
}

func (service *MarketingService) brevoAttributes(user *auth.UserRecord) map[string]any {
	name := strings.TrimSpace(user.DisplayName)
	if name == "" {
		return fiber.Map{}
	}

	parts := strings.Split(name, " ")
	if len(parts) == 1 {
		return fiber.Map{"FNAME": name}
	}

	return fiber.Map{
		"FNAME": strings.Join(parts[0:len(parts)-1], " "),
		"LNAME": parts[len(parts)-1],
	}
}
