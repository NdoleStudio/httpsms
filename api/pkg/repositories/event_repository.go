package repositories

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// EventRepository is responsible for persisting cloudevents.Event
type EventRepository interface {
	// Save a new entities.Message
	Save(ctx context.Context, event cloudevents.Event) error
}
