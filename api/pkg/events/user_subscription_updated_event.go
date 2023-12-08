package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserSubscriptionUpdated is raised when a user subscription is updated
const UserSubscriptionUpdated = "user.subscription.updated"

// UserSubscriptionUpdatedPayload stores the data for the UserSubscriptionUpdated event
type UserSubscriptionUpdatedPayload struct {
	UserID                entities.UserID           `json:"user_id"`
	SubscriptionUpdatedAt time.Time                 `json:"subscription_updated_at"`
	SubscriptionEndsAt    *time.Time                `json:"subscription_ends_at"`
	SubscriptionRenewsAt  time.Time                 `json:"subscription_renews_at"`
	SubscriptionID        string                    `json:"subscription_id"`
	SubscriptionName      entities.SubscriptionName `json:"subscription_name"`
	SubscriptionStatus    string                    `json:"subscription_status"`
}
