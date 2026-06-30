package events

import (
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// RateLimitExceeded is raised when a user exceeds their daily API rate limit.
const RateLimitExceeded = "rate.limit.exceeded"

// RateLimitExceededPayload stores the data for the RateLimitExceeded event
type RateLimitExceededPayload struct {
	UserID    entities.UserID `json:"user_id"`
	Count     int64           `json:"count"`
	Limit     uint            `json:"limit"`
	Plan      string          `json:"plan"`
	Timestamp time.Time       `json:"timestamp"`
}
