package services

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
)

// entityLimits maps entity name → subscription plan → max count.
// A limit of 0 means unlimited. If a plan is not listed, it defaults to unlimited (0).
var entityLimits = map[string]map[entities.SubscriptionName]int{
	"MessageSendSchedule": {
		entities.SubscriptionNameFree: 1,
	},
}

// EntitlementCheckResult holds the outcome of an entitlement check.
type EntitlementCheckResult struct {
	Allowed bool
	Message string
}

// EntitlementService checks whether a user can create more of a given entity
// based on their subscription plan.
type EntitlementService struct {
	service
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	enabled        bool
	userRepository repositories.UserRepository
}

// NewEntitlementService creates a new EntitlementService.
// The enabled flag should come from the ENTITLEMENT_ENABLED environment variable.
func NewEntitlementService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	enabled bool,
	userRepository repositories.UserRepository,
) *EntitlementService {
	return &EntitlementService{
		logger:         logger.WithService(fmt.Sprintf("%T", &EntitlementService{})),
		tracer:         tracer,
		enabled:        enabled,
		userRepository: userRepository,
	}
}

// Check verifies if the user can create another instance of the given entity.
func (service *EntitlementService) Check(
	ctx context.Context,
	userID entities.UserID,
	entityName string,
	currentCount int,
) (*EntitlementCheckResult, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	if !service.enabled {
		return &EntitlementCheckResult{Allowed: true}, nil
	}

	limits, exists := entityLimits[entityName]
	if !exists {
		return &EntitlementCheckResult{Allowed: true}, nil
	}

	user, err := service.userRepository.Load(ctx, userID)
	if err != nil {
		return nil, service.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(err, fmt.Sprintf("cannot load user [%s] for entitlement check", userID)),
		)
	}

	limit, hasLimit := limits[user.SubscriptionName]
	if !hasLimit || limit == 0 {
		return &EntitlementCheckResult{Allowed: true}, nil
	}

	if currentCount >= limit {
		return &EntitlementCheckResult{
			Allowed: false,
			Message: fmt.Sprintf(
				"Upgrade to a paid plan to create more than %d send schedule. Visit https://httpsms.com/pricing for details.",
				limit,
			),
		}, nil
	}

	return &EntitlementCheckResult{Allowed: true}, nil
}
