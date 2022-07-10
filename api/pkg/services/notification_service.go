package services

import (
	"context"
	"fmt"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// NotificationService sends out notifications to mobile phones
type NotificationService struct {
	logger          telemetry.Logger
	tracer          telemetry.Tracer
	phoneRepository repositories.PhoneRepository
	messagingClient *messaging.Client
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	messagingClient *messaging.Client,
	phoneRepository repositories.PhoneRepository,
) (s *NotificationService) {
	return &NotificationService{
		logger:          logger.WithService(fmt.Sprintf("%T", s)),
		tracer:          tracer,
		messagingClient: messagingClient,
		phoneRepository: phoneRepository,
	}
}

// NotificationMessageSentParams are parameters for sending a notification
type NotificationMessageSentParams struct {
	UserID    entities.UserID
	Owner     string
	MessageID uuid.UUID
}

// MessageSent sends a message when a message is sent
func (service *NotificationService) MessageSent(ctx context.Context, params *NotificationMessageSentParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	phone, err := service.phoneRepository.Load(ctx, params.UserID, params.Owner)
	if err != nil {
		msg := fmt.Sprintf("cannot load phone with userID [%s] and phone [%s]", params.UserID, params.Owner)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if phone.FcmToken == nil {
		msg := fmt.Sprintf("phone with id [%s] has no FCM token", phone.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.NewError(msg))
	}

	result, err := service.messagingClient.Send(ctx, &messaging.Message{
		Data: map[string]string{
			"KEY_MESSAGE_ID": params.MessageID.String(),
		},
		Token: *phone.FcmToken,
	})
	if err != nil {
		msg := fmt.Sprintf("cannot send notification for message [%s] to phone [%s]", params.MessageID, phone.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("sent notification [%s] for message [%s] to phone [%s]", result, params.MessageID, phone.ID))
	return nil
}
