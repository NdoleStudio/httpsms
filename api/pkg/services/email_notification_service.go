package services

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/cache"
	"github.com/NdoleStudio/httpsms/pkg/emails"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// EmailNotificationService is responsible for handling email notifications about messages
type EmailNotificationService struct {
	service
	logger         telemetry.Logger
	tracer         telemetry.Tracer
	userRepository repositories.UserRepository
	factory        emails.NotificationEmailFactory
	mailer         emails.Mailer
	cache          cache.Cache
}

// NewEmailNotificationService creates a new EmailNotificationService
func NewEmailNotificationService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	userRepository repositories.UserRepository,
	factory emails.NotificationEmailFactory,
	mailer emails.Mailer,
	cache cache.Cache,
) *EmailNotificationService {
	return &EmailNotificationService{
		logger:         logger.WithService(fmt.Sprintf("%T", &EmailNotificationService{})),
		tracer:         tracer,
		userRepository: userRepository,
		factory:        factory,
		mailer:         mailer,
		cache:          cache,
	}
}

// NotifyMessageExpired sends an email to the user about an expired message
func (service *EmailNotificationService) NotifyMessageExpired(ctx context.Context, payload *events.MessageSendExpiredPayload) error {
	return nil
}

// NotifyMessageFailed sends an email to the user about a failed message
func (service *EmailNotificationService) NotifyMessageFailed(ctx context.Context, payload *events.MessageSendFailedPayload) error {
	return nil
}
