package services

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
)

// MessageService is handles message requests
type MessageService struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
}

// NewMessageService creates a new MessageService
func NewMessageService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
) (s *MessageService) {
	return &MessageService{
		logger: logger.WithService(fmt.Sprintf("%T", s)),
		tracer: tracer,
	}
}

// Send a new message
func (service *MessageService) Send(ctx context.Context, params MessageSendParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()
	return nil, nil
}
