package entities

import (
	"time"

	"github.com/google/uuid"
)

// UserID is the ID of a user
type UserID string

// SubscriptionName is the name of the subscription
type SubscriptionName string

// Limit returns the limit of a subscription
func (subscription SubscriptionName) Limit() uint {
	switch subscription {
	case SubscriptionNameProMonthly, SubscriptionNameProYearly, SubscriptionNameProLifetime:
		return 5000
	case SubscriptionNameUltraMonthly, SubscriptionNameUltraYearly:
		return 10_000
	case SubscriptionName20KMonthly, SubscriptionName20KYearly:
		return 20_000
	case SubscriptionName50KMonthly:
		return 50_000
	case SubscriptionName100KMonthly:
		return 100_000
	case SubscriptionName200KMonthly:
		return 200_000
	default:
		return 200
	}
}

// SubscriptionNameFree represents a free subscription
const SubscriptionNameFree = SubscriptionName("free")

// SubscriptionNameProMonthly represents a monthly pro subscription
const SubscriptionNameProMonthly = SubscriptionName("pro-monthly")

// SubscriptionNameProYearly represents a yearly pro subscription
const SubscriptionNameProYearly = SubscriptionName("pro-yearly")

// SubscriptionNameUltraMonthly represents a monthly ultra subscription
const SubscriptionNameUltraMonthly = SubscriptionName("ultra-monthly")

// SubscriptionNameUltraYearly represents a yearly ultra subscription
const SubscriptionNameUltraYearly = SubscriptionName("ultra-yearly")

// SubscriptionNameProLifetime represents a pro lifetime subscription
const SubscriptionNameProLifetime = SubscriptionName("pro-lifetime")

// SubscriptionName20KMonthly represents a monthly 20k subscription
const SubscriptionName20KMonthly = SubscriptionName("20k-monthly")

// SubscriptionName100KMonthly represents a monthly 100k subscription
const SubscriptionName100KMonthly = SubscriptionName("100k-monthly")

// SubscriptionName50KMonthly represents a monthly 50k subscription
const SubscriptionName50KMonthly = SubscriptionName("50k-monthly")

// SubscriptionName200KMonthly represents a monthly 200k subscription
const SubscriptionName200KMonthly = SubscriptionName("200k-monthly")

// SubscriptionName20KYearly represents a yearly 20k subscription
const SubscriptionName20KYearly = SubscriptionName("20k-yearly")

// User stores information about a user
type User struct {
	ID                               UserID           `json:"id" gorm:"primaryKey;type:string;" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Email                            string           `json:"email" example:"name@email.com"`
	APIKey                           string           `json:"api_key" gorm:"uniqueIndex:idx_users_api_key" example:"x-api-key"`
	Timezone                         string           `json:"timezone" example:"Europe/Helsinki" gorm:"default:Africa/Accra"`
	ActivePhoneID                    *uuid.UUID       `json:"active_phone_id" gorm:"type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	SubscriptionName                 SubscriptionName `json:"subscription_name" example:"free"`
	SubscriptionID                   *string          `json:"subscription_id" example:"8f9c71b8-b84e-4417-8408-a62274f65a08"`
	SubscriptionStatus               *string          `json:"subscription_status" example:"on_trial"`
	SubscriptionRenewsAt             *time.Time       `json:"subscription_renews_at" example:"2022-06-05T14:26:02.302718+03:00"`
	SubscriptionEndsAt               *time.Time       `json:"subscription_ends_at" example:"2022-06-05T14:26:02.302718+03:00"`
	NotificationMessageStatusEnabled bool             `json:"notification_message_status_enabled" gorm:"default:true" example:"true"`
	NotificationWebhookEnabled       bool             `json:"notification_webhook_enabled" gorm:"default:true" example:"true"`
	NotificationHeartbeatEnabled     bool             `json:"notification_heartbeat_enabled" gorm:"default:true" example:"true"`
	CreatedAt                        time.Time        `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt                        time.Time        `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}

// IsOnProPlan checks if a user is on the pro plan
func (user User) IsOnProPlan() bool {
	return user.SubscriptionName == SubscriptionNameProLifetime || user.SubscriptionName == SubscriptionNameProMonthly || user.SubscriptionName == SubscriptionNameProYearly
}

// IsOnFreePlan checks if a user is on the free plan
func (user User) IsOnFreePlan() bool {
	return user.SubscriptionName == SubscriptionNameFree || user.SubscriptionName == ""
}

// IsOnUltraPlan checks if a user is on the ultra plan
func (user User) IsOnUltraPlan() bool {
	return user.SubscriptionName == SubscriptionNameUltraMonthly || user.SubscriptionName == SubscriptionNameUltraYearly
}

// IsOn20kPlan checks if a user is on the 20k plan
func (user User) IsOn20kPlan() bool {
	return user.SubscriptionName == SubscriptionName20KMonthly || user.SubscriptionName == SubscriptionName20KYearly
}

// UserTimeString converts the time to the user's timezone
func (user User) UserTimeString(timestamp time.Time) string {
	location, err := time.LoadLocation(user.Timezone)
	if err != nil {
		location = time.UTC
	}
	return timestamp.In(location).Format(time.RFC1123)
}

// Location gets the timezone of a user
func (user User) Location() *time.Location {
	location, err := time.LoadLocation(user.Timezone)
	if err != nil {
		location = time.UTC
	}
	return location
}
