package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserSubscriptionCreated is raised when a user subscription is created
const UserSubscriptionCreated = "user.subscription.created"

// UserSubscriptionCreatedPayload stores the data for the user created event
type UserSubscriptionCreatedPayload struct {
	UserID                entities.UserID           `json:"user_id"`
	SubscriptionCreatedAt time.Time                 `json:"subscription_created_at"`
	SubscriptionID        string                    `json:"subscription_id"`
	SubscriptionName      entities.SubscriptionName `json:"subscription_name"`
	SubscriptionRenewsAt  time.Time                 `json:"subscription_renews_at"`
	SubscriptionStatus    string                    `json:"subscription_status"`
}
