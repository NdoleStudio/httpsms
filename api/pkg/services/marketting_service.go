package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/sendgrid/sendgrid-go"

	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/davecgh/go-spew/spew"
	"github.com/palantir/stacktrace"
)

// MarketingService is handles marketing requests
type MarketingService struct {
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	authClient     *auth.Client
	sendgridAPIKey string
	sendgridListID string
}

type sendgridContact struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type sendgridContactRequest struct {
	ListIDs  []string          `json:"list_ids"`
	Contacts []sendgridContact `json:"contacts"`
}

// NewMarketingService creates a new instance of the MarketingService
func NewMarketingService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	authClient *auth.Client,
	sendgridAPIKey string,
	sendgridListID string,
) *MarketingService {
	return &MarketingService{
		logger:         logger.WithService(fmt.Sprintf("%T", &MarketingService{})),
		tracer:         tracer,
		authClient:     authClient,
		sendgridAPIKey: sendgridAPIKey,
		sendgridListID: sendgridListID,
	}
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

	id, err := service.addContact(sendgridContactRequest{
		ListIDs:  []string{service.sendgridListID},
		Contacts: []sendgridContact{service.toSendgridContact(userRecord)},
	})
	if err != nil {
		msg := fmt.Sprintf("cannot add user with id [%s] to list [%s]", user.ID, service.sendgridListID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return
	}

	ctxLogger.Info(fmt.Sprintf("user [%s] added to list [%s] with job [%s]", user.ID, service.sendgridListID, id))
}

// DeleteContacts deletes contacts from sendgrid
func (service *MarketingService) DeleteContacts(ctx context.Context, contactIDs []string) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	request := sendgrid.GetRequest(service.sendgridAPIKey, "/v3/marketing/contacts", "https://api.sendgrid.com")
	request.Method = "DELETE"
	request.QueryParams = map[string]string{
		"ids": strings.Join(contactIDs, ","),
	}

	response, err := sendgrid.API(request)
	if err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot delete contacts in a sendgrid list [%s]", service.sendgridListID))
	}

	ctxLogger.Info(spew.Sdump(response.Body))
	return nil
}

func (service *MarketingService) toSendgridContact(user *auth.UserRecord) sendgridContact {
	name := strings.TrimSpace(user.DisplayName)
	if name == "" {
		return sendgridContact{
			FirstName: "",
			LastName:  "",
			Email:     user.Email,
		}
	}

	parts := strings.Split(name, " ")
	if len(parts) == 1 {
		return sendgridContact{
			FirstName: name,
			LastName:  "",
			Email:     user.Email,
		}
	}

	return sendgridContact{
		FirstName: strings.Join(parts[0:len(parts)-1], " "),
		LastName:  parts[len(parts)-1],
		Email:     user.Email,
	}
}

func (service *MarketingService) addContact(contactRequest sendgridContactRequest) (string, error) {
	request := sendgrid.GetRequest(service.sendgridAPIKey, "/v3/marketing/contacts", "https://api.sendgrid.com")
	request.Method = "PUT"

	body, err := json.Marshal(contactRequest)
	if err != nil {
		log.Fatal(stacktrace.Propagate(err, fmt.Sprintf("cannot marshal [%s]", spew.Sdump(contactRequest))))
	}

	request.Body = body
	response, err := sendgrid.API(request)
	if err != nil {
		return "", stacktrace.Propagate(err, fmt.Sprintf("cannot add contact to sendgrid list [%s]", spew.Sdump(contactRequest)))
	}
	return response.Body, nil
}
