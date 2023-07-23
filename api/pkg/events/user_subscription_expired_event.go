package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserSubscriptionExpired is raised when a user subscription is cancelled
const UserSubscriptionExpired = "user.subscription.expired"

// UserSubscriptionExpiredPayload stores the data for the UserSubscriptionExpired event
type UserSubscriptionExpiredPayload struct {
	UserID                entities.UserID           `json:"user_id"`
	SubscriptionExpiredAt time.Time                 `json:"subscription_expired_at"`
	SubscriptionEndsAt    time.Time                 `json:"subscription_ends_at"`
	IsCancelled           bool                      `json:"is_cancelled"`
	SubscriptionID        string                    `json:"subscription_id"`
	SubscriptionName      entities.SubscriptionName `json:"subscription_name"`
	SubscriptionStatus    string                    `json:"subscription_status"`
}
