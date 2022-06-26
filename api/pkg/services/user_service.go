package services

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
)

// UserService is handles heartbeat requests
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
