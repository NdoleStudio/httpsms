package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
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

func (repository *gormWebhookRepository) Save(ctx context.Context, webhook *entities.Webhook) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(webhook).Error; err != nil {
		msg := fmt.Sprintf("cannot update webhook with ID [%s]", webhook.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
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
		msg := fmt.Sprintf("cannot fetch webhooks for user [%s] and params [%+#v]", userID, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return webhooks, nil
}

func (repository *gormWebhookRepository) LoadByEvent(ctx context.Context, userID entities.UserID, event string) ([]*entities.Webhook, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	webhooks := make([]*entities.Webhook, 0)
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("? = ANY(events)", event).Find(webhooks).Error
	if err != nil {
		msg := fmt.Sprintf("cannot load webhooks for user with ID [%s] and event [%s]", userID, event)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return webhooks, nil
}

func (repository *gormWebhookRepository) Load(ctx context.Context, userID entities.UserID, webhookID uuid.UUID) (*entities.Webhook, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	webhook := new(entities.Webhook)
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", webhookID).First(&webhook).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("webhook with ID [%s] for user [%s] does not exist", webhookID, userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load webhook with ID [%s] for user [%s]", webhookID, userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
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
		msg := fmt.Sprintf("cannot delete webhook with ID [%s] and userID [%s]", webhookID, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
