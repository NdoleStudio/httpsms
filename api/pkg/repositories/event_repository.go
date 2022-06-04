package repositories

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// EventRepository is responsible for persisting events
type EventRepository interface {
	Save(ctx context.Context, event cloudevents.Event) error
}
