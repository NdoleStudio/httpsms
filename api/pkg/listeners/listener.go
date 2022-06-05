package listeners

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

type listener struct {
	repository repositories.EventListenerLogRepository
}

func (listener listener) handlerSignature(handler any, event cloudevents.Event) string {
	return fmt.Sprintf("%s.%T", event.Type(), handler)
}

func (listener listener) storeEventListenerLog(ctx context.Context, handler string, event cloudevents.Event) error {
	return listener.repository.Store(ctx, &entities.EventListenerLog{
		ID:        uuid.New(),
		EventID:   event.ID(),
		EventType: event.Type(),
		Handler:   handler,
		Duration:  time.Now().Sub(event.Time()),
		HandledAt: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
	})
}
