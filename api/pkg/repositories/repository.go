package repositories

import (
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/palantir/stacktrace"
)

// IndexParams parameters for indexing a database table
type IndexParams struct {
	Skip           int    `json:"skip"`
	SortBy         string `json:"sort"`
	SortDescending bool   `json:"sort_descending"`
	Query          string `json:"query"`
	Limit          int    `json:"take"`
}

const (
	// ErrCodeNotFound is thrown when an entity does not exist in storage
	ErrCodeNotFound = stacktrace.ErrorCode(1000)

	dbOperationDuration = 5 * time.Second
)

// isRetryableError checks if the error is a retryable connection error
func isRetryableError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "bad connection") ||
		strings.Contains(msg, "stream is closed") ||
		strings.Contains(msg, "driver: bad connection")
}

// executeWithRetry executes a GORM query with retry logic for transient connection errors
func executeWithRetry(fn func() error) (err error) {
	return retry.Do(
		fn,
		retry.LastErrorOnly(true),
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.RetryIf(isRetryableError),
	)
}
