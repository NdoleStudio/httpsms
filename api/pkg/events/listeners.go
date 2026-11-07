package events

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// ExtensionQueueID identifies the queue task delivering an event.
const ExtensionQueueID = "queueid"

// EventListener is the type for processing events
type EventListener func(ctx context.Context, event cloudevents.Event) error
