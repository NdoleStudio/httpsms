package services

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"

	"github.com/NdoleStudio/httpsms/pkg/emails"
	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"

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
	marketingService   *MarketingService
	lemonsqueezyClient *lemonsqueezy.Client
}

// NewUserService creates a new UserService
func NewUserService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.UserRepository,
	mailer emails.Mailer,
	emailFactory emails.UserEmailFactory,
	marketingService *MarketingService,
	lemonsqueezyClient *lemonsqueezy.Client,
) (s *UserService) {
	return &UserService{
		logger:             logger.WithService(fmt.Sprintf("%T", s)),
		tracer:             tracer,
		mailer:             mailer,
		marketingService:   marketingService,
		emailFactory:       emailFactory,
		repository:         repository,
		lemonsqueezyClient: lemonsqueezyClient,
	}
}

// Get fetches or creates an entities.User
func (service *UserService) Get(ctx context.Context, authUser entities.AuthUser) (*entities.User, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	user, isNew, err := service.repository.LoadOrStore(ctx, authUser)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with from [%+#v]", user, authUser)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if isNew {
		service.marketingService.AddToList(ctx, user)
	}

	return user, nil
}

// UserUpdateParams are parameters for updating an entities.User
type UserUpdateParams struct {
	Timezone      *time.Location
	ActivePhoneID uuid.UUID
}

// Update an entities.User
func (service *UserService) Update(ctx context.Context, authUser entities.AuthUser, params UserUpdateParams) (*entities.User, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	user, isNew, err := service.repository.LoadOrStore(ctx, authUser)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with from [%+#v]", user, authUser)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if isNew {
		service.marketingService.AddToList(ctx, user)
	}

	user.Timezone = params.Timezone.String()
	user.ActivePhoneID = &params.ActivePhoneID

	if err = service.repository.Update(ctx, user); err != nil {
		msg := fmt.Sprintf("cannot save user with id [%s]", user.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("user saved with id [%s] in the userRepository", user.ID))
	return user, nil
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

	return subscription.Data.Attributes.Urls.UpdatePaymentMethod, nil
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
