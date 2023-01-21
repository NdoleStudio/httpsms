package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"
)

// LemonsqueezyHandlerValidator validates models used in handlers.LemonsqueezyHandler
type LemonsqueezyHandlerValidator struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	client *lemonsqueezy.Client
}

// NewLemonsqueezyHandlerValidator creates a new handlers.LemonsqueezyHandler validator
func NewLemonsqueezyHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	client *lemonsqueezy.Client,
) (v *LemonsqueezyHandlerValidator) {
	return &LemonsqueezyHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
		client: client,
	}
}

// ValidateEvent checks that an event is coming from lemonsqueezy
func (validator *LemonsqueezyHandlerValidator) ValidateEvent(ctx context.Context, signature string, request []byte) url.Values {
	_, span := validator.tracer.Start(ctx)
	defer span.End()

	isValid := validator.client.Webhooks.Verify(ctx, signature, request)
	if !isValid {
		return url.Values{
			"body": []string{
				"The signature is not valid",
			},
		}
	}
	return url.Values{}
}
