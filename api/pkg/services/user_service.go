package services

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/emails"

	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// UserService is handles user requests
type UserService struct {
	logger       telemetry.Logger
	tracer       telemetry.Tracer
	emailFactory emails.UserEmailFactory
	mailer       emails.Mailer
	repository   repositories.UserRepository
}

// NewUserService creates a new UserService
func NewUserService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.UserRepository,
	mailer emails.Mailer,
	emailFactory emails.UserEmailFactory,
) (s *UserService) {
	return &UserService{
		logger:       logger.WithService(fmt.Sprintf("%T", s)),
		tracer:       tracer,
		mailer:       mailer,
		emailFactory: emailFactory,
		repository:   repository,
	}
}

// Get fetches or creates an entities.User
func (service *UserService) Get(ctx context.Context, authUser entities.AuthUser) (*entities.User, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	user, err := service.repository.LoadOrStore(ctx, authUser)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with from [%+#v]", user, authUser)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return user, nil
}

// UserUpdateParams are parameters for updating an entities.User
type UserUpdateParams struct {
	ActivePhoneID uuid.UUID
}

// Update an entities.User
func (service *UserService) Update(ctx context.Context, authUser entities.AuthUser, params UserUpdateParams) (*entities.User, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	user, err := service.repository.LoadOrStore(ctx, authUser)
	if err != nil {
		msg := fmt.Sprintf("could not get [%T] with from [%+#v]", user, authUser)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

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
