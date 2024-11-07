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

// gormDiscordRepository is responsible for persisting entities.Discord
type gormDiscordRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormDiscordRepository creates the GORM version of the DiscordRepository
func NewGormDiscordRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) DiscordRepository {
	return &gormDiscordRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormDiscordRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *gormDiscordRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.Discord{}).Error; err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.Discord{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *gormDiscordRepository) Save(ctx context.Context, Discord *entities.Discord) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(Discord).Error; err != nil {
		msg := fmt.Sprintf("cannot update discord integration with ID [%s]", Discord.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Index entities.Message between 2 parties
func (repository *gormDiscordRepository) Index(ctx context.Context, userID entities.UserID, params IndexParams) ([]*entities.Discord, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.WithContext(ctx).Where("user_id = ?", userID)
	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query.Where(repository.db.Where("url ILIKE ?", queryPattern))
	}

	discords := make([]*entities.Discord, 0)
	if err := query.Order("created_at DESC").Limit(params.Limit).Offset(params.Skip).Find(&discords).Error; err != nil {
		msg := fmt.Sprintf("cannot fetch discord integrations for user [%s] and params [%+#v]", userID, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return discords, nil
}

func (repository *gormDiscordRepository) FetchHavingIncomingChannel(ctx context.Context, userID entities.UserID) ([]*entities.Discord, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	discords := make([]*entities.Discord, 0)
	err := repository.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Where("incoming_channel_id IS NOT NULL").
		Where("incoming_channel_id != ?", "").
		Find(&discords).Error
	if err != nil {
		msg := fmt.Sprintf("cannot load discord integrations for user with ID [%s] having a valid [incoming_channel_id]", userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return discords, nil
}

func (repository *gormDiscordRepository) Load(ctx context.Context, userID entities.UserID, discordID uuid.UUID) (*entities.Discord, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	discord := new(entities.Discord)
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", discordID).First(&discord).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("discord integration with ID [%s] for user [%s] does not exist", discordID, userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load discord integration with ID [%s] for user [%s]", discordID, userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return discord, nil
}

func (repository *gormDiscordRepository) FindByServerID(ctx context.Context, serverID string) (*entities.Discord, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	discord := new(entities.Discord)
	err := repository.db.WithContext(ctx).Where("server_id = ?", serverID).First(&discord).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("discord integration with server ID [%s] does not exist", serverID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load discord integration with serverID [%s]", serverID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return discord, nil
}

func (repository *gormDiscordRepository) Delete(ctx context.Context, userID entities.UserID, discordID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", discordID).
		Delete(&entities.Discord{}).Error
	if err != nil {
		msg := fmt.Sprintf("cannot delete discord integration with ID [%s] and userID [%s]", discordID, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
