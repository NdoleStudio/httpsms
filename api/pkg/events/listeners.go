package events

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type contextKey string

// ExtensionQueueID identifies the queue task delivering an event in its context.
const ExtensionQueueID contextKey = "queue.task.id"

// EventListener is the type for processing events
type EventListener func(ctx context.Context, event cloudevents.Event) error
