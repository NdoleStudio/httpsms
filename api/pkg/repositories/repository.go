package repositories

import "github.com/palantir/stacktrace"

// IndexParams parameters for indexing a database table
type IndexParams struct {
	Skip  int    `json:"skip"`
	Query string `json:"query"`
	Limit int    `json:"take"`
}

const (
	// ErrCodeNotFound is thrown when an entity does not exist in storage
	ErrCodeNotFound = stacktrace.ErrorCode(1000)
)
