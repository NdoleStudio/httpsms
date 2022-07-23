package services

import (
	"context"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
)

// PushQueueTask represents a push queue task
type PushQueueTask struct {
	Method  string
	URL     string
	Body    []byte
	Headers map[string]string
}

// PushQueueConfig configurations for the push queue
type PushQueueConfig struct {
	Name             string
	UserAPIKey       string
	UserID           entities.UserID
	ConsumerEndpoint string
}

// PushQueue is a push queue
type PushQueue interface {
	// Enqueue adds a message to the push queue
	Enqueue(ctx context.Context, task *PushQueueTask, timeout time.Duration) error
}
