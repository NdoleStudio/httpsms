package services

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
)

// UserService is handles user requests
type UserService struct {
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.UserRepository
}

// NewUserService creates a new UserService
func NewUserService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.UserRepository,
) (s *UserService) {
	return &UserService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		repository: repository,
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
