package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"firebase.google.com/go/auth"
	"github.com/NdoleStudio/httpsms/pkg/emails"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/lemonsqueezy-go"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// UserService is handles user requests
type UserService struct {
	service
	logger             telemetry.Logger
	tracer             telemetry.Tracer
	emailFactory       emails.UserEmailFactory
	mailer             emails.Mailer
	repository         repositories.UserRepository
	dispatcher         *EventDispatcher
	authClient         *auth.Client
	lemonsqueezyClient *lemonsqueezy.Client
	httpClient         *http.Client
}

// NewUserService creates a new UserService
func NewUserService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.UserRepository,
	mailer emails.Mailer,
	emailFactory emails.UserEmailFactory,
	lemonsqueezyClient *lemonsqueezy.Client,
	dispatcher *EventDispatcher,
	authClient *auth.Client,
	httpClient *http.Client,
) (s *UserService) {
	return &UserService{
		logger:             logger.WithService(fmt.Sprintf("%T", s)),
		tracer:             tracer,
		mailer:             mailer,
		emailFactory:       emailFactory,
		repository:         repository,
		dispatcher:         dispatcher,
		authClient:         authClient,
		lemonsqueezyClient: lemonsqueezyClient,
		httpClient:         httpClient,
	}
}

// GetSubscriptionPayments fetches the subscription payments for an entities.User
func (service *UserService) GetSubscriptionPayments(ctx context.Context, userID entities.UserID) (invoices []lemonsqueezy.ApiResponseData[lemonsqueezy.SubscriptionInvoiceAttributes, lemonsqueezy.APIResponseRelationshipsSubscriptionInvoice], err error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.repository.Load(ctx, userID)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with with ID [%s]", user, userID)
		return invoices, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if user.SubscriptionID == nil {
		ctxLogger.Info(fmt.Sprintf("no subscription ID found for [%T] with ID [%s], returning empty invoices", user, user.ID))
		return invoices, nil
	}

	ctxLogger.Info(fmt.Sprintf("fetching subscription payments for [%T] with ID [%s] and subscription [%s]", user, user.ID, *user.SubscriptionID))
	invoicesResponse, _, err := service.lemonsqueezyClient.SubscriptionInvoices.List(ctx, map[string]string{"filter[subscription_id]": *user.SubscriptionID})
	if err != nil {
		msg := fmt.Sprintf("could not get invoices for subscription [%s] for [%T] with with ID [%s]", *user.SubscriptionID, user, user.ID)
		return invoices, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] payments for [%T] with ID [%s] and subscription ID [%s]", len(invoicesResponse.Data), user, user.ID, *user.SubscriptionID))
	return invoicesResponse.Data, nil
}

// UserInvoiceGenerateParams are parameters for generating a subscription payment invoice
type UserInvoiceGenerateParams struct {
	UserID                entities.UserID
	SubscriptionInvoiceID string
	Name                  string
	Address               string
	City                  string
	State                 string
	Country               string
	ZipCode               string
	Notes                 string
}

// GenerateReceipt generates a receipt for a subscription payment.
func (service *UserService) GenerateReceipt(ctx context.Context, params *UserInvoiceGenerateParams) (io.Reader, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	payload := map[string]string{
		"name":     params.Name,
		"address":  params.Address,
		"city":     params.City,
		"state":    params.State,
		"country":  params.Country,
		"zip_code": params.ZipCode,
		"notes":    params.Notes,
		"locale":   "en",
	}

	invoice, _, err := service.lemonsqueezyClient.SubscriptionInvoices.Generate(ctx, params.SubscriptionInvoiceID, payload)
	if err != nil {
		msg := fmt.Sprintf("could not generate subscription payment invoice user with ID [%s] and subscription invoice ID [%s]", params.UserID, params.SubscriptionInvoiceID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	response, err := service.httpClient.Get(invoice.Meta.Urls.DownloadInvoice)
	if err != nil {
		msg := fmt.Sprintf("could not download subscription payment invoice for user with ID [%s] and subscription invoice ID [%s]", params.UserID, params.SubscriptionInvoiceID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("generated subscription payment invoice for user with ID [%s] and subscription invoice ID [%s]", params.UserID, params.SubscriptionInvoiceID))
	return response.Body, nil
}

// Get fetches or creates an entities.User
func (service *UserService) Get(ctx context.Context, source string, authUser entities.AuthContext) (*entities.User, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	user, isNew, err := service.repository.LoadOrStore(ctx, authUser)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with from [%+#v]", user, authUser)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if isNew {
		service.dispatchUserCreatedEvent(ctx, source, user)
	}

	return user, nil
}

func (service *UserService) dispatchUserCreatedEvent(ctx context.Context, source string, user *entities.User) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	event, err := service.createEvent(events.UserAccountCreated, source, &events.UserAccountCreatedPayload{
		UserID:    user.ID,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for user [%s]", events.UserAccountCreated, user.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch [%s] event for user [%s]", event.Type(), user.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return
	}
}

// GetByID fetches an entities.User
func (service *UserService) GetByID(ctx context.Context, userID entities.UserID) (*entities.User, error) {
	ctx, span, _ := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.repository.Load(ctx, userID)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with ID [%s]", user, userID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return user, nil
}

// UserUpdateParams are parameters for updating an entities.User
type UserUpdateParams struct {
	Timezone      *time.Location
	ActivePhoneID *uuid.UUID
}

// Update an entities.User
func (service *UserService) Update(ctx context.Context, source string, authUser entities.AuthContext, params UserUpdateParams) (*entities.User, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	user, isNew, err := service.repository.LoadOrStore(ctx, authUser)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with from [%+#v]", user, authUser)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if isNew {
		service.dispatchUserCreatedEvent(ctx, source, user)
	}

	user.Timezone = params.Timezone.String()
	user.ActivePhoneID = params.ActivePhoneID

	if err = service.repository.Update(ctx, user); err != nil {
		msg := fmt.Sprintf("cannot save user with id [%s]", user.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("user saved with id [%s] in the userRepository", user.ID))
	return user, nil
}

// UserNotificationUpdateParams are parameters for updating the notifications of a user
type UserNotificationUpdateParams struct {
	MessageStatusEnabled bool
	WebhookEnabled       bool
	HeartbeatEnabled     bool
	NewsletterEnabled    bool
}

// UpdateNotificationSettings for an entities.User
func (service *UserService) UpdateNotificationSettings(ctx context.Context, userID entities.UserID, params *UserNotificationUpdateParams) (*entities.User, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.repository.Load(ctx, userID)
	if err != nil {
		msg := fmt.Sprintf("could not load [%T] with ID [%s]", user, userID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	user.NotificationWebhookEnabled = params.WebhookEnabled
	user.NotificationHeartbeatEnabled = params.HeartbeatEnabled
	user.NotificationMessageStatusEnabled = params.MessageStatusEnabled
	user.NotificationNewsletterEnabled = params.NewsletterEnabled

	if err = service.repository.Update(ctx, user); err != nil {
		msg := fmt.Sprintf("cannot save user with id [%s] in [%T]", user.ID, service.repository)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("updated notification settings for [%T] with ID [%s] in the [%T]", user, user.ID, service.repository))
	return user, nil
}

// RotateAPIKey for an entities.User
func (service *UserService) RotateAPIKey(ctx context.Context, source string, userID entities.UserID) (*entities.User, error) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.repository.RotateAPIKey(ctx, userID)
	if err != nil {
		msg := fmt.Sprintf("could not rotate API key for [%T] with ID [%s]", user, userID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("rotated the api key for [%T] with ID [%s] in the [%T]", user, user.ID, service.repository))

	event, err := service.createEvent(events.UserAPIKeyRotated, source, &events.UserAPIKeyRotatedPayload{
		UserID:    user.ID,
		Email:     user.Email,
		Timestamp: time.Now().UTC(),
		Timezone:  user.Timezone,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for user [%s]", events.UserAPIKeyRotated, user.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return user, nil
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch [%s] event for user [%s]", event.Type(), user.ID)
		ctxLogger.Error(stacktrace.Propagate(err, msg))
		return user, nil
	}

	return user, nil
}

// Delete an entities.User
func (service *UserService) Delete(ctx context.Context, source string, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.repository.Load(ctx, userID)
	if err != nil {
		msg := fmt.Sprintf("cannot load user with ID [%s] from the [%T]", userID, service.repository)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !user.IsOnFreePlan() && user.SubscriptionRenewsAt != nil && user.SubscriptionRenewsAt.After(time.Now()) {
		msg := fmt.Sprintf("cannot delete user with ID [%s] because they are have an active [%s] subscription which renews at [%s]", userID, user.SubscriptionName, user.SubscriptionRenewsAt)
		return service.tracer.WrapErrorSpan(span, stacktrace.NewError(msg))
	}

	if err = service.repository.Delete(ctx, user); err != nil {
		msg := fmt.Sprintf("could not delete user with ID [%s] from the [%T]", userID, service.repository)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("sucessfully deleted user with ID [%s] in the [%T]", userID, service.repository))

	event, err := service.createEvent(events.UserAccountDeleted, source, &events.UserAccountDeletedPayload{
		UserID:    userID,
		UserEmail: user.Email,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create event [%s] for user [%s]", events.UserAccountDeleted, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.dispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch [%s] event for user [%s]", event.Type(), userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// SendAPIKeyRotatedEmail sends an email to an entities.User when the API key is rotated
func (service *UserService) SendAPIKeyRotatedEmail(ctx context.Context, payload *events.UserAPIKeyRotatedPayload) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	email, err := service.emailFactory.APIKeyRotated(payload.Email, payload.Timestamp, payload.Timezone)
	if err != nil {
		msg := fmt.Sprintf("cannot create api key rotated email for user [%s]", payload.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.mailer.Send(ctx, email); err != nil {
		msg := fmt.Sprintf("canot create api key rotated email to user [%s]", payload.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("api key rotated email sent successfully to [%s] with user ID  [%s]", payload.Email, payload.UserID))
	return nil
}

// UserSendPhoneDeadEmailParams are parameters for notifying a user when a phone is dead
type UserSendPhoneDeadEmailParams struct {
	UserID                 entities.UserID
	PhoneID                uuid.UUID
	Owner                  string
	LastHeartbeatTimestamp time.Time
}

// SendPhoneDeadEmail sends an email to an entities.User when a phone is dead
func (service *UserService) SendPhoneDeadEmail(ctx context.Context, params *UserSendPhoneDeadEmailParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	user, err := service.repository.Load(ctx, params.UserID)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with ID [%s]", user, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !user.NotificationHeartbeatEnabled {
		ctxLogger.Info(fmt.Sprintf("[%s] email notifications disabled for user [%s] with owner [%s]", events.EventTypePhoneHeartbeatOffline, params.UserID, params.Owner))
		return nil
	}

	email, err := service.emailFactory.PhoneDead(user, params.LastHeartbeatTimestamp, params.Owner)
	if err != nil {
		msg := fmt.Sprintf("cannot create phone dead email for user [%s]", params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.mailer.Send(ctx, email); err != nil {
		msg := fmt.Sprintf("canot send phone dead notification to user [%s]", params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("phone dead notification sent successfully to [%s] about [%s]", user.Email, params.Owner))
	return nil
}

// StartSubscription starts a subscription for an entities.User
func (service *UserService) StartSubscription(ctx context.Context, params *events.UserSubscriptionCreatedPayload) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	user, err := service.repository.Load(ctx, params.UserID)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with with ID [%s]", user, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	user.SubscriptionID = &params.SubscriptionID
	user.SubscriptionName = params.SubscriptionName
	user.SubscriptionRenewsAt = &params.SubscriptionRenewsAt
	user.SubscriptionStatus = &params.SubscriptionStatus
	user.SubscriptionEndsAt = nil

	if err = service.repository.Update(ctx, user); err != nil {
		msg := fmt.Sprintf("could not update [%T] with with ID [%s] after update", user, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// InitiateSubscriptionCancel initiates the cancelling of a subscription on lemonsqueezy
func (service *UserService) InitiateSubscriptionCancel(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.repository.Load(ctx, userID)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with with ID [%s]", user, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if _, _, err = service.lemonsqueezyClient.Subscriptions.Cancel(ctx, *user.SubscriptionID); err != nil {
		msg := fmt.Sprintf("could not cancel subscription [%s] for [%T] with with ID [%s]", *user.SubscriptionID, user, user.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("cancelled subscription [%s] for user [%s]", *user.SubscriptionID, user.ID))
	return nil
}

// GetSubscriptionUpdateURL initiates the cancelling of a subscription on lemonsqueezy
func (service *UserService) GetSubscriptionUpdateURL(ctx context.Context, userID entities.UserID) (url string, err error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	user, err := service.repository.Load(ctx, userID)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with with ID [%s]", user, userID)
		return "", service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	subscription, _, err := service.lemonsqueezyClient.Subscriptions.Get(ctx, *user.SubscriptionID)
	if err != nil {
		msg := fmt.Sprintf("could not get subscription [%s] for [%T] with with ID [%s]", *user.SubscriptionID, user, user.ID)
		return url, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return subscription.Data.Attributes.Urls.CustomerPortal, nil
}

// CancelSubscription starts a subscription for an entities.User
func (service *UserService) CancelSubscription(ctx context.Context, params *events.UserSubscriptionCancelledPayload) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	user, err := service.repository.Load(ctx, params.UserID)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with with ID [%s]", user, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	user.SubscriptionID = &params.SubscriptionID
	user.SubscriptionName = params.SubscriptionName
	user.SubscriptionRenewsAt = nil
	user.SubscriptionStatus = &params.SubscriptionStatus
	user.SubscriptionEndsAt = &params.SubscriptionEndsAt

	if err = service.repository.Update(ctx, user); err != nil {
		msg := fmt.Sprintf("could not update [%T] with with ID [%s] after update", user, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// ExpireSubscription finishes a subscription for an entities.User
func (service *UserService) ExpireSubscription(ctx context.Context, params *events.UserSubscriptionExpiredPayload) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	user, err := service.repository.Load(ctx, params.UserID)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with with ID [%s]", user, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	user.SubscriptionID = nil
	user.SubscriptionName = entities.SubscriptionNameFree
	user.SubscriptionRenewsAt = nil
	user.SubscriptionStatus = nil
	user.SubscriptionEndsAt = nil

	if err = service.repository.Update(ctx, user); err != nil {
		msg := fmt.Sprintf("could not update [%T] with with ID [%s] after expired subscription update", user, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// UpdateSubscription updates a subscription for an entities.User
func (service *UserService) UpdateSubscription(ctx context.Context, params *events.UserSubscriptionUpdatedPayload) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	user, err := service.repository.Load(ctx, params.UserID)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with with ID [%s]", user, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if params.SubscriptionStatus != "active" {
		msg := fmt.Sprintf("subscription status is [%s] for [%T] with with ID [%s]", params.SubscriptionStatus, user, params.UserID)
		ctxLogger.Info(msg)
		return nil
	}

	user.SubscriptionID = &params.SubscriptionID
	user.SubscriptionName = params.SubscriptionName
	user.SubscriptionEndsAt = params.SubscriptionEndsAt
	user.SubscriptionRenewsAt = &params.SubscriptionRenewsAt
	user.SubscriptionStatus = &params.SubscriptionStatus

	if err = service.repository.Update(ctx, user); err != nil {
		msg := fmt.Sprintf("could not update [%T] with with ID [%s] after subscription update", user, params.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// DeleteAuthUser deletes an entities.AuthContext from firebase
func (service *UserService) DeleteAuthUser(ctx context.Context, userID entities.UserID) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.authClient.DeleteUser(ctx, userID.String()); err != nil {
		msg := fmt.Sprintf("could not delete [entities.AuthContext] from firebase with ID [%s]", userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("deleted [entities.AuthContext] from firebase for user with ID [%s]", userID))
	return nil
}
