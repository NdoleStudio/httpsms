package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserAPIKeyRotated is raised when a user's API key is rotated
const UserAPIKeyRotated = "user.api-key.rotated"

// UserAPIKeyRotatedPayload stores the data for the UserAPIKeyRotated event
type UserAPIKeyRotatedPayload struct {
	UserID    entities.UserID `json:"user_id"`
	Email     string          `json:"email"`
	Timestamp time.Time       `json:"timestamp"`
	Timezone  string          `json:"timezone"`
}
