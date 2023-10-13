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

const (
	fifteenMinuteTimeout = 15 * time.Minute
	oneHourTimeout       = 1 * time.Hour
)

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

	if !user.NotificationMessageStatusEnabled {
		ctxLogger.Info(fmt.Sprintf("[%s] email notifications disabled for user [%s] with owner [%s]", events.EventTypeMessageSendExpired, payload.UserID, payload.Owner))
		return nil
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

	service.addToCache(ctx, oneHourTimeout, events.EventTypeMessageSendExpired, payload.Owner)
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

	if !user.NotificationMessageStatusEnabled {
		ctxLogger.Info(fmt.Sprintf("[%s] email notifications disabled for user [%s] with owner [%s]", events.EventTypeMessageSendFailed, payload.UserID, payload.Owner))
		return nil
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

	service.addToCache(ctx, fifteenMinuteTimeout, events.EventTypeMessageSendFailed, payload.Owner)
	return nil
}

// NotifyWebhookSendFailed sends an email to the user about a failed webhook
func (service *EmailNotificationService) NotifyWebhookSendFailed(ctx context.Context, payload *events.WebhookSendFailedPayload) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if !service.canSendEmail(ctx, events.EventTypeWebhookSendFailed, payload.Owner) {
		ctxLogger.Info(fmt.Sprintf("[%s] email already sent to user [%s] with owner [%s]", events.EventTypeWebhookSendFailed, payload.UserID, payload.Owner))
		return nil
	}

	user, err := service.userRepository.Load(ctx, payload.UserID)
	if err != nil {
		msg := fmt.Sprintf("cannot load user with ID [%s] for [%s] event with ID [%s]", payload.UserID, events.EventTypeWebhookSendFailed, payload.EventID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	if !user.NotificationWebhookEnabled {
		ctxLogger.Info(fmt.Sprintf("[%s] email notifications disabled for user [%s] with owner [%s]", events.EventTypeWebhookSendFailed, payload.UserID, payload.Owner))
		return nil
	}

	email, err := service.factory.WebhookSendFailed(user, payload)
	if err != nil {
		msg := fmt.Sprintf("cannot create [%s] email for user with ID [%s] for [%s] event with ID [%s]", events.EventTypeWebhookSendFailed, payload.UserID, payload.EventType, payload.EventID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.mailer.Send(ctx, email); err != nil {
		msg := fmt.Sprintf("cannot send [%s] email for user with ID [%s] for [%s] event with ID [%s]", events.EventTypeWebhookSendFailed, payload.UserID, payload.EventType, payload.EventID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("[%s] email sent to [%s] for [%s] event with ID [%s]", events.EventTypeWebhookSendFailed, user.ID, payload.EventType, payload.EventID))

	service.addToCache(ctx, oneHourTimeout, events.EventTypeWebhookSendFailed, payload.Owner)
	return nil
}

// NotifyDiscordSendFailed sends an email to the user about a failed discord webhook event
func (service *EmailNotificationService) NotifyDiscordSendFailed(ctx context.Context, payload *events.DiscordSendFailedPayload) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if !service.canSendEmail(ctx, events.EventTypeDiscordSendFailed, payload.Owner) {
		ctxLogger.Info(fmt.Sprintf("[%s] email already sent to user [%s] with owner [%s]", events.EventTypeWebhookSendFailed, payload.UserID, payload.Owner))
		return nil
	}

	user, err := service.userRepository.Load(ctx, payload.UserID)
	if err != nil {
		msg := fmt.Sprintf("cannot load user with ID [%s] for [%s] event for message with ID [%s]", payload.UserID, payload.EventType, payload.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	if !user.NotificationWebhookEnabled {
		ctxLogger.Info(fmt.Sprintf("[%s] email notifications disabled for user [%s] with owner [%s]", events.EventTypeDiscordSendFailed, payload.UserID, payload.Owner))
		return nil
	}

	email, err := service.factory.DiscordSendFailed(user, payload)
	if err != nil {
		msg := fmt.Sprintf("cannot create email for user with ID [%s] for [%s] event and message with ID  [%s]", payload.UserID, payload.EventType, payload.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.mailer.Send(ctx, email); err != nil {
		msg := fmt.Sprintf("cannot send email for user with ID [%s] for [%s] message with ID [%s]", payload.UserID, events.EventTypeMessageSendFailed, payload.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("[%s] email sent to [%s] for [%s] event with ID [%s]", payload.EventType, user.ID, payload.EventType, payload.MessageID))

	service.addToCache(ctx, oneHourTimeout, events.EventTypeDiscordSendFailed, payload.Owner)
	return nil
}

func (service *EmailNotificationService) getCacheKey(event string, owner string) string {
	return fmt.Sprintf("email.%s.%s", event, owner)
}

func (service *EmailNotificationService) canSendEmail(ctx context.Context, event string, owner string) bool {
	_, err := service.cache.Get(ctx, service.getCacheKey(event, owner))
	return err != nil
}

func (service *EmailNotificationService) addToCache(ctx context.Context, timeout time.Duration, event string, owner string) {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	cacheKey := service.getCacheKey(event, owner)
	if err := service.cache.Set(ctx, cacheKey, "", timeout); err != nil {
		ctxLogger.Error(stacktrace.Propagate(err, fmt.Sprintf("cannot set item in redis with key [%s] for owner [%s]", cacheKey, owner)))
	}
}
