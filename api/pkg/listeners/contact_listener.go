package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// ContactDeletionService removes contacts belonging to a deleted user.
type ContactDeletionService interface {
	DeleteAllForUser(ctx context.Context, userID entities.UserID) error
}

// ContactListener handles contact-related cloud events.
type ContactListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service ContactDeletionService
}

// NewContactListener creates a new ContactListener.
func NewContactListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service ContactDeletionService,
) (l *ContactListener, routes map[string]events.EventListener) {
	l = &ContactListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.UserAccountDeleted: l.onUserAccountDeleted,
	}
}

func (listener *ContactListener) onUserAccountDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserAccountDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	if err := listener.service.DeleteAllForUser(ctx, payload.UserID); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete [entities.Contact] for user [%s] on [%s] event with ID [%s]", payload.UserID, event.Type(), event.ID()))
	}

	return nil
}
