package emails

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserEmailFactory generates emails to a user
type UserEmailFactory interface {
	// PhoneDead sends an emails when the user's phone is not sending heartbeats
	PhoneDead(user *entities.User, lastHeartbeatTimestamp time.Time, owner string) (*Email, error)

	// UsageLimitExceeded sends an email when the user's limit is exceeded
	UsageLimitExceeded(user *entities.User) (*Email, error)

	// UsageLimitAlert sends an email when a user is approaching the limit
	UsageLimitAlert(user *entities.User, usage *entities.BillingUsage) (*Email, error)

	// APIKeyRotated sends an email when the API key is rotated
	APIKeyRotated(email string, timestamp time.Time, timezone string) (*Email, error)
}
