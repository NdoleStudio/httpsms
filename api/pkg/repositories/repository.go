package repositories

import (
	"time"

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
