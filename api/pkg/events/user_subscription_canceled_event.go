package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserSubscriptionCancelled is raised when a user subscription is cancelled
const UserSubscriptionCancelled = "user.subscription.cancelled"

// UserSubscriptionCancelledPayload stores the data for the UserSubscriptionCancelled event
type UserSubscriptionCancelledPayload struct {
	UserID                  entities.UserID           `json:"user_id"`
	SubscriptionCancelledAt time.Time                 `json:"subscription_cancelled_at"`
	SubscriptionEndsAt      time.Time                 `json:"subscription_ends_at"`
	SubscriptionID          string                    `json:"subscription_id"`
	SubscriptionName        entities.SubscriptionName `json:"subscription_name"`
	SubscriptionStatus      string                    `json:"subscription_status"`
}
