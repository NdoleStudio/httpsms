package emails

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserEmailFactory generates emails to a user
type UserEmailFactory interface {
	// PhoneDead sends an emails when the user's phone is not sending heartbeats
	PhoneDead(user *entities.User, lastHeartbeatTimestamp time.Time, owner string) (*Email, error)
}
