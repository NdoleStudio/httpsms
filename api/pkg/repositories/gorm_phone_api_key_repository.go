package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/dgraph-io/ristretto"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormPhoneAPIKeyRepository is responsible for persisting entities.PhoneAPIKey
type gormPhoneAPIKeyRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	cache  *ristretto.Cache[string, entities.AuthContext]
	db     *gorm.DB
}

// NewGormPhoneAPIKeyRepository creates the GORM version of the PhoneAPIKeyRepository
func NewGormPhoneAPIKeyRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
	cache *ristretto.Cache[string, entities.AuthContext],
) PhoneAPIKeyRepository {
	return &gormPhoneAPIKeyRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormPhoneAPIKeyRepository{})),
		tracer: tracer,
		cache:  cache,
		db:     db,
	}
}

func (repository *gormPhoneAPIKeyRepository) RemovePhoneByID(ctx context.Context, userID entities.UserID, phoneID uuid.UUID, phoneNumber string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := `
UPDATE ?
SET phone_ids = array_remove(phone_ids, ?),
    phone_numbers = array_remove(phone_numbers, ?)
WHERE user_id = ? AND array_position(phone_ids, ?) IS NOT NULL;
`
	err := repository.db.WithContext(ctx).
		Raw(query, (entities.PhoneAPIKey{}).TableName(), phoneID, phoneNumber, userID, phoneID).
		Error
	if err != nil {
		msg := fmt.Sprintf("cannot remove phone with ID [%s] and number [%s] for user with ID [%s] ", phoneID, phoneNumber, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	repository.cache.Clear()
	return nil
}

// Load an entities.PhoneAPIKey based on the entities.UserID
func (repository *gormPhoneAPIKeyRepository) Load(ctx context.Context, userID entities.UserID, phoneAPIKeyID uuid.UUID) (*entities.PhoneAPIKey, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	phoneAPIKey := new(entities.PhoneAPIKey)
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", phoneAPIKeyID).First(&phoneAPIKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("[%T] with ID [%s] for user with ID [%s] does not exist", phoneAPIKey, phoneAPIKeyID, userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load [%T] with ID [%s] for user with ID [%s]", phoneAPIKey, phoneAPIKeyID, userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return phoneAPIKey, nil
}

func (repository *gormPhoneAPIKeyRepository) Create(ctx context.Context, phoneAPIKey *entities.PhoneAPIKey) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Create(phoneAPIKey).Error; err != nil {
		msg := fmt.Sprintf("cannot save phone API key with ID [%s] for user with ID [%s]", phoneAPIKey.ID, phoneAPIKey.UserID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *gormPhoneAPIKeyRepository) LoadAuthContext(ctx context.Context, apiKey string) (entities.AuthContext, error) {
	ctx, span, ctxLogger := repository.tracer.StartWithLogger(ctx, repository.logger)
	defer span.End()

	if authContext, found := repository.cache.Get(apiKey); found {
		ctxLogger.Info(fmt.Sprintf("cache hit for user with ID [%s] and phone API Key ID [%s]", authContext.ID, *authContext.PhoneAPIKeyID))
		return authContext, nil
	}

	phoneAPIKey := new(entities.PhoneAPIKey)
	err := repository.db.WithContext(ctx).Where("api_key = ?", phoneAPIKey).First(apiKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("phone api key [%s] does not exist", apiKey)
		return entities.AuthContext{}, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load phone api key [%s]", apiKey)
		return entities.AuthContext{}, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	authUser := entities.AuthContext{
		ID:            phoneAPIKey.UserID,
		Email:         phoneAPIKey.UserEmail,
		PhoneAPIKeyID: &phoneAPIKey.ID,
		PhoneNumbers:  phoneAPIKey.PhoneNumbers,
	}

	if result := repository.cache.SetWithTTL(apiKey, authUser, 1, 1*time.Hour); !result {
		msg := fmt.Sprintf("cannot cache [%T] with ID [%s] and result [%t]", authUser, phoneAPIKey.ID, result)
		ctxLogger.Error(repository.tracer.WrapErrorSpan(span, stacktrace.NewError(msg)))
	}

	return authUser, nil
}

func (repository *gormPhoneAPIKeyRepository) Index(ctx context.Context, userID entities.UserID, params IndexParams) ([]*entities.PhoneAPIKey, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.WithContext(ctx).Where("user_id = ?", userID)
	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query.Where("name ILIKE ?", queryPattern)
	}

	phoneAPIKeys := new([]*entities.PhoneAPIKey)
	if err := query.Order("created_at DESC").Limit(params.Limit).Offset(params.Skip).Find(phoneAPIKeys).Error; err != nil {
		msg := fmt.Sprintf("cannot fetch phone API Keys with userID [%s] and params [%+#v]", userID, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return *phoneAPIKeys, nil
}

func (repository *gormPhoneAPIKeyRepository) Delete(ctx context.Context, phoneAPIKey *entities.PhoneAPIKey) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Delete(phoneAPIKey).Error
	if err != nil {
		msg := fmt.Sprintf("cannot delete phone API key with ID [%s] and userID [%s]", phoneAPIKey.ID, phoneAPIKey.UserID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	repository.cache.Del(phoneAPIKey.APIKey)

	return nil
}

func (repository *gormPhoneAPIKeyRepository) AddPhone(ctx context.Context, phoneAPIKey *entities.PhoneAPIKey, phone *entities.Phone) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error {
		query := `
UPDATE ?
SET phone_ids = array_remove(phone_ids, ?),
    phone_numbers = array_remove(phone_numbers, ?)
WHERE  user_id  = ?;
`
		err := tx.WithContext(ctx).
			Raw(query, phoneAPIKey.TableName(), phone.ID, phone.PhoneNumber, phone.UserID).
			Error
		if err != nil {
			msg := fmt.Sprintf("cannot remove phone with ID [%s] from API Key with ID [%s] for user with ID [%s]", phone.ID, phoneAPIKey.ID, phone.UserID)
			return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}

		query = `
UPDATE ?
SET phone_ids = array_append(phone_ids, ?),
    phone_numbers = array_append(phone_numbers, ?)
WHERE array_position(phone_ids, ?) IS NULL AND id = ?;
`
		err = repository.db.WithContext(ctx).
			Raw(query, phoneAPIKey.TableName(), phone.ID, phone.PhoneNumber, phoneAPIKey.ID).
			Error
		if err != nil {
			msg := fmt.Sprintf("cannot add [%T] with ID [%s] from API Key with ID [%s] for user with ID [%s]", phone, phone.ID, phoneAPIKey.ID, phone.UserID)
			return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}
		return nil
	})
	if err != nil {
		msg := fmt.Sprintf("cannot add [%T] with ID [%s] from API Key with ID [%s] for user with ID [%s]", phone, phone.ID, phoneAPIKey.ID, phone.UserID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	repository.cache.Clear()
	return nil
}

func (repository *gormPhoneAPIKeyRepository) RemovePhone(ctx context.Context, phoneAPIKey *entities.PhoneAPIKey, phone *entities.Phone) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := `
UPDATE ?
SET phone_ids = array_remove(phone_ids, ?),
    phone_numbers = array_remove(phone_numbers, ?)
WHERE id = ?;
`
	err := repository.db.WithContext(ctx).
		Raw(query, phoneAPIKey.TableName(), phone.ID, phone.PhoneNumber, phoneAPIKey.ID).
		Error
	if err != nil {
		msg := fmt.Sprintf("cannot remove phone with ID [%s] from phone API key with ID [%s]", phone.ID, phoneAPIKey.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	repository.cache.Del(phoneAPIKey.APIKey)

	return nil
}

func (repository *gormPhoneAPIKeyRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.PhoneAPIKey{}).Error; err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.PhoneAPIKey{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
