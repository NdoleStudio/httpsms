package events

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// EventListener is the type for processing events
type EventListener func(ctx context.Context, event cloudevents.Event) error

// EventFilter decides whether an event may reach any subscribers.
type EventFilter func(ctx context.Context, event cloudevents.Event) (bool, error)
