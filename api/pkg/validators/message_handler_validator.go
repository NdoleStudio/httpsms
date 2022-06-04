package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/http-sms-manager/pkg/requests"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
)

// MessageHandlerValidator validates models used in handlers.MessageHandler
type MessageHandlerValidator struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewMessageHandlerValidator creates a new handlers.MessageHandler validator
func NewMessageHandlerValidator(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (v *MessageHandlerValidator) {
	return &MessageHandlerValidator{
		logger: logger.WithService(fmt.Sprintf("%T", v)),
		tracer: tracer,
	}
}

// ValidateMessageSend validates the requests.MessageSend request
func (v MessageHandlerValidator) ValidateMessageSend(ctx context.Context, request requests.MessageSend) url.Values {
	return nil
}
