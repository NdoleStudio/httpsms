package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormPhoneRepository is responsible for persisting entities.Phone
type gormPhoneRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	cache  *ristretto.Cache[string, *entities.Phone]
	db     *gorm.DB
}

// NewGormPhoneRepository creates the GORM version of the PhoneRepository
func NewGormPhoneRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
	cache *ristretto.Cache[string, *entities.Phone],
) PhoneRepository {
	return &gormPhoneRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormPhoneRepository{})),
		tracer: tracer,
		db:     db,
		cache:  cache,
	}
}

func (repository *gormPhoneRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.Phone{}).Error; err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.Phone{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	repository.cache.Clear()
	return nil
}

// LoadByID loads a phone by ID
func (repository *gormPhoneRepository) LoadByID(ctx context.Context, userID entities.UserID, phoneID uuid.UUID) (*entities.Phone, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	phone := new(entities.Phone)
	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", phoneID).
		First(&phone).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("phone with ID [%s] does not exist", phoneID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load phone with ID [%s]", phoneID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return phone, nil
}

// Delete an entities.Phone
func (repository *gormPhoneRepository) Delete(ctx context.Context, userID entities.UserID, phoneID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", phoneID).
		Delete(&entities.Phone{}).Error
	if err != nil {
		msg := fmt.Sprintf("cannot delete phone with ID [%s] and userID [%s]", phoneID, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	repository.cache.Clear()
	return nil
}

// Save a new entities.Phone
func (repository *gormPhoneRepository) Save(ctx context.Context, phone *entities.Phone) error {
	ctx, span, ctxLogger := repository.tracer.StartWithLogger(ctx, repository.logger)
	defer span.End()

	err := repository.db.WithContext(ctx).Save(phone).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		ctxLogger.Info(fmt.Sprintf("phone with user [%s] and number[%s] already exists", phone.UserID, phone.PhoneNumber))
		loadedPhone, err := repository.Load(ctx, phone.UserID, phone.PhoneNumber)
		if err != nil {
			msg := fmt.Sprintf("cannot load phone for user [%s] and number [%s]", phone.UserID, phone.PhoneNumber)
			return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}
		*phone = *loadedPhone
		return nil
	}

	if err != nil {
		msg := fmt.Sprintf("cannot save phone with ID [%s]", phone.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	repository.cache.Del(repository.getCacheKey(phone.UserID, phone.PhoneNumber))
	return nil
}

// Load a phone based on entities.UserID and phoneNumber
func (repository *gormPhoneRepository) Load(ctx context.Context, userID entities.UserID, phoneNumber string) (*entities.Phone, error) {
	ctx, span, ctxLogger := repository.tracer.StartWithLogger(ctx, repository.logger)
	defer span.End()

	if phone, found := repository.cache.Get(repository.getCacheKey(userID, phoneNumber)); found {
		ctxLogger.Info(fmt.Sprintf("cache hit for [%T] with ID [%s]", phone, userID))
		return phone, nil
	}

	phone := new(entities.Phone)
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("phone_number = ?", phoneNumber).First(phone).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("phone with userID [%s] and phoneNumber [%s] does not exist", userID, phoneNumber)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load phone phone with userID [%s] and phoneNumber [%s]", userID, phoneNumber)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if result := repository.cache.SetWithTTL(repository.getCacheKey(userID, phoneNumber), phone, 1, 30*time.Minute); !result {
		msg := fmt.Sprintf("cannot cache [%T] with ID [%s] and result [%t]", phone, phone.ID, result)
		ctxLogger.Error(repository.tracer.WrapErrorSpan(span, stacktrace.NewError(msg)))
	}

	return phone, nil
}

func (repository *gormPhoneRepository) Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.Phone, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.WithContext(ctx).Where("user_id = ?", userID)
	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query.Where("phone_number ILIKE ?", queryPattern)
	}

	phones := new([]entities.Phone)
	if err := query.Order("created_at DESC").Limit(params.Limit).Offset(params.Skip).Find(&phones).Error; err != nil {
		msg := fmt.Sprintf("cannot fetch phones with userID [%s] and params [%+#v]", userID, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return phones, nil
}

func (repository *gormPhoneRepository) getCacheKey(userID entities.UserID, phoneNumber string) string {
	return fmt.Sprintf("user:%s:phone:%s", userID, phoneNumber)
}
