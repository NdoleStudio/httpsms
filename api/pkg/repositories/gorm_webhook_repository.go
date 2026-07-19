package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormWebhookRepository is responsible for persisting entities.Webhook
type gormWebhookRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormWebhookRepository creates the GORM version of the WebhookRepository
func NewGormWebhookRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) WebhookRepository {
	return &gormWebhookRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormWebhookRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *gormWebhookRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.Webhook{}).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete all [%T] for user with ID [%s]", &entities.Webhook{}, userID))
	}

	return nil
}

func (repository *gormWebhookRepository) Save(ctx context.Context, webhook *entities.Webhook) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(webhook).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update webhook with ID [%s]", webhook.ID))
	}

	return nil
}

// Index entities.Message between 2 parties
func (repository *gormWebhookRepository) Index(ctx context.Context, userID entities.UserID, params IndexParams) ([]*entities.Webhook, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.WithContext(ctx).Where("user_id = ?", userID)
	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query.Where(repository.db.Where("url ILIKE ?", queryPattern))
	}

	webhooks := make([]*entities.Webhook, 0)
	if err := query.Order("created_at DESC").Limit(params.Limit).Offset(params.Skip).Find(&webhooks).Error; err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch webhooks for user [%s] and params [%+#v]", userID, params))
	}

	return webhooks, nil
}

func (repository *gormWebhookRepository) LoadByEvent(ctx context.Context, userID entities.UserID, event string, phoneNumber string) ([]*entities.Webhook, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	webhooks := make([]*entities.Webhook, 0)
	err := repository.db.
		Raw("SELECT * FROM webhooks WHERE user_id = ? AND CAST(? as TEXT) = ANY(events) AND CAST(? as TEXT) = ANY(phone_numbers)", userID, event, phoneNumber).
		Scan(&webhooks).
		Error
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load webhooks for user with ID [%s] and event [%s]", userID, event))
	}

	return webhooks, nil
}

func (repository *gormWebhookRepository) Load(ctx context.Context, userID entities.UserID, webhookID uuid.UUID) (*entities.Webhook, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	webhook := new(entities.Webhook)
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", webhookID).First(&webhook).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "webhook with ID [%s] for user [%s] does not exist", webhookID, userID))
	}

	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load webhook with ID [%s] for user [%s]", webhookID, userID))
	}

	return webhook, nil
}

func (repository *gormWebhookRepository) Delete(ctx context.Context, userID entities.UserID, webhookID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", webhookID).
		Delete(&entities.Webhook{}).Error
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete webhook with ID [%s] and userID [%s]", webhookID, userID))
	}

	return nil
}
