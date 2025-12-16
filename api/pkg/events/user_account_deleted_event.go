package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// UserAccountDeleted is raised when a user's account is deleted.
const UserAccountDeleted = "user.account.deleted"

// UserAccountDeletedPayload stores the data for the UserAccountDeleted event
type UserAccountDeletedPayload struct {
	UserID    entities.UserID `json:"user_id"`
	UserEmail string          `json:"user_email"`
	Timestamp time.Time       `json:"timestamp"`
}
