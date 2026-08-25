package services

import (
	"context"
	"strconv"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type entitlementUserRepository struct {
	repositories.UserRepository
	user      *entities.User
	loadCalls int
}

func (repository *entitlementUserRepository) Load(_ context.Context, _ entities.UserID) (*entities.User, error) {
	repository.loadCalls++
	return repository.user, nil
}

func newEntitlementServiceForTest(enabled bool, repository repositories.UserRepository) *EntitlementService {
	logger := newRecordingLogger()
	return NewEntitlementService(logger, telemetry.NewOtelLogger("test", logger), enabled, repository)
}

func TestEntitlementService_CheckAdditional_DisabledAllowsWithoutLoadingUserOrCount(t *testing.T) {
	repository := &entitlementUserRepository{}
	service := newEntitlementServiceForTest(false, repository)
	countCalls := 0

	result, err := service.CheckAdditional(context.Background(), "user-id", "Contact", 1000, func() (int, error) {
		countCalls++
		return 200, nil
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Zero(t, repository.loadCalls)
	assert.Zero(t, countCalls)
}

func TestEntitlementService_CheckAdditional_UsesSubscriptionLimitForContactBatch(t *testing.T) {
	tests := []struct {
		name             string
		subscriptionName entities.SubscriptionName
		currentCount     int
		additionalCount  int
		allowed          bool
	}{
		{
			name:             "free user can reach 200 contacts",
			subscriptionName: entities.SubscriptionNameFree,
			currentCount:     199,
			additionalCount:  1,
			allowed:          true,
		},
		{
			name:             "free user cannot exceed 200 contacts",
			subscriptionName: entities.SubscriptionNameFree,
			currentCount:     199,
			additionalCount:  2,
			allowed:          false,
		},
		{
			name:             "pro user can reach 5000 contacts",
			subscriptionName: entities.SubscriptionNameProMonthly,
			currentCount:     4999,
			additionalCount:  1,
			allowed:          true,
		},
		{
			name:             "pro user cannot exceed 5000 contacts",
			subscriptionName: entities.SubscriptionNameProMonthly,
			currentCount:     4999,
			additionalCount:  2,
			allowed:          false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &entitlementUserRepository{
				user: &entities.User{SubscriptionName: test.subscriptionName},
			}
			service := newEntitlementServiceForTest(true, repository)

			result, err := service.CheckAdditional(context.Background(), "user-id", "Contact", test.additionalCount, func() (int, error) {
				return test.currentCount, nil
			})

			require.NoError(t, err)
			assert.Equal(t, test.allowed, result.Allowed)
			if !test.allowed {
				assert.Contains(t, result.Message, "more than ["+strconv.FormatUint(uint64(test.subscriptionName.Limit()), 10)+"]")
				assert.Contains(t, result.Message, "Upgrade your plan")
			}
		})
	}
}
