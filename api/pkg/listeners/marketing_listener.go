package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"
)

// MarketingListener handled marketing events
type MarketingListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.MarketingService
}

// NewMarketingListener creates a new instance of MarketingListener
func NewMarketingListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.MarketingService,
) (l *MarketingListener, routes map[string]events.EventListener) {
	l = &MarketingListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.UserAccountDeleted: l.onUserAccountDeleted,
	}
}

func (listener *MarketingListener) onUserAccountDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserAccountDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.DeleteUser(ctx, payload.UserID); err != nil {
		msg := fmt.Sprintf("cannot delete [sendgrid contact] for user [%s] on [%s] event with ID [%s]", payload.UserID, event.Type(), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
