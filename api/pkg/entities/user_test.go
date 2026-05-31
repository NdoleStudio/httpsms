package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUser_GetBillingAnchorDay_FreeUser(t *testing.T) {
	user := User{
		SubscriptionName: SubscriptionNameFree,
		CreatedAt:        time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, 20, user.GetBillingAnchorDay())
}

func TestUser_GetBillingAnchorDay_EmptySubscription(t *testing.T) {
	user := User{
		SubscriptionName: "",
		CreatedAt:        time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, 5, user.GetBillingAnchorDay())
}

func TestUser_GetBillingAnchorDay_PaidUser(t *testing.T) {
	renewsAt := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	user := User{
		SubscriptionName:     SubscriptionNameProMonthly,
		SubscriptionRenewsAt: &renewsAt,
		CreatedAt:            time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, 15, user.GetBillingAnchorDay())
}

func TestUser_GetBillingAnchorDay_PaidUserNilRenewsAt(t *testing.T) {
	user := User{
		SubscriptionName:     SubscriptionNameProMonthly,
		SubscriptionRenewsAt: nil,
		CreatedAt:            time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, 28, user.GetBillingAnchorDay())
}

func TestUser_GetBillingAnchorDay_PaidUserDay31(t *testing.T) {
	renewsAt := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	user := User{
		SubscriptionName:     SubscriptionNameUltraMonthly,
		SubscriptionRenewsAt: &renewsAt,
		CreatedAt:            time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC),
	}
	assert.Equal(t, 31, user.GetBillingAnchorDay())
}
