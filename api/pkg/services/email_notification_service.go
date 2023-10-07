package services

import (
	"context"
	"fmt"
	"time"

	"github.com/palantir/stacktrace"

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
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if !payload.IsFinal {
		ctxLogger.Info(fmt.Sprintf("[%s] event is not final, send attempt count = [%d]", events.EventTypeMessageSendExpired, payload.SendAttemptCount))
		return nil
	}

	if !service.canSendEmail(ctx, events.EventTypeMessageSendExpired, payload.Owner) {
		ctxLogger.Info(fmt.Sprintf("[%s] email already sent to user [%s] with owner [%s]", events.EventTypeMessageSendExpired, payload.UserID, payload.Owner))
		return nil
	}

	user, err := service.userRepository.Load(ctx, payload.UserID)
	if err != nil {
		msg := fmt.Sprintf("cannot load user with ID [%s] and for expired message with ID [%s]", payload.UserID, payload.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	email, err := service.factory.MessageExpired(user, payload.MessageID, payload.Owner, payload.Contact, payload.Content)
	if err != nil {
		msg := fmt.Sprintf("cannot create email for user with ID [%s] and for expired message with ID [%s]", payload.UserID, payload.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.mailer.Send(ctx, email); err != nil {
		msg := fmt.Sprintf("cannot send email for user with ID [%s] and for expired message with ID [%s]", payload.UserID, payload.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("[%s] email sent to [%s] for message with ID [%s]", events.EventTypeMessageSendExpired, user.ID, payload.MessageID))

	service.addToCache(ctx, events.EventTypeMessageSendExpired, payload.Owner)
	return nil
}

// NotifyMessageFailed sends an email to the user about a failed message
func (service *EmailNotificationService) NotifyMessageFailed(ctx context.Context, payload *events.MessageSendFailedPayload) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if !service.canSendEmail(ctx, events.EventTypeMessageSendFailed, payload.Owner) {
		ctxLogger.Info(fmt.Sprintf("[%s] email already sent to user [%s] with owner [%s]", events.EventTypeMessageSendFailed, payload.UserID, payload.Owner))
		return nil
	}

	user, err := service.userRepository.Load(ctx, payload.UserID)
	if err != nil {
		msg := fmt.Sprintf("cannot load user with ID [%s] for [%s] message with ID [%s]", payload.UserID, payload.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	email, err := service.factory.MessageFailed(user, payload.ID, payload.Owner, payload.Contact, payload.Content, payload.ErrorMessage)
	if err != nil {
		msg := fmt.Sprintf("cannot create email for user with ID [%s] for [%s] message with ID [%s]", payload.UserID, payload.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.mailer.Send(ctx, email); err != nil {
		msg := fmt.Sprintf("cannot send email for user with ID [%s] for [%s] message with ID [%s]", payload.UserID, events.EventTypeMessageSendFailed, payload.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("[%s] email sent to [%s] for message with ID [%s]", events.EventTypeMessageSendFailed, user.ID, payload.ID))

	service.addToCache(ctx, events.EventTypeMessageSendFailed, payload.Owner)
	return nil
}

func (service *EmailNotificationService) getCacheKey(event string, owner string) string {
	return fmt.Sprintf("email.%s.%s", event, owner)
}

func (service *EmailNotificationService) canSendEmail(ctx context.Context, event string, owner string) bool {
	_, err := service.cache.Get(ctx, service.getCacheKey(event, owner))
	return err != nil
}

func (service *EmailNotificationService) addToCache(ctx context.Context, event string, owner string) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	cacheKey := service.getCacheKey(event, owner)
	if err := service.cache.Set(ctx, cacheKey, "", time.Minute*15); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot set item in redis with key [%s] for owner [%s]", cacheKey, owner)))
	}
}
