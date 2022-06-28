package repositories

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormUserRepository is responsible for persisting entities.User
type gormUserRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormUserRepository creates the GORM version of the UserRepository
func NewGormUserRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) UserRepository {
	return &gormUserRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormUserRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *gormUserRepository) Store(ctx context.Context, user *entities.User) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.Create(user).Error; err != nil {
		msg := fmt.Sprintf("cannot save user with ID [%s]", user.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *gormUserRepository) Update(ctx context.Context, user *entities.User) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.Save(user).Error; err != nil {
		msg := fmt.Sprintf("cannot update user with ID [%s]", user.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *gormUserRepository) LoadAuthUser(ctx context.Context, apiKey string) (*entities.AuthUser, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	user := new(entities.User)
	err := repository.db.Where("api_key = ?", apiKey).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("user with api key [%s] does not exist", apiKey)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load user with api key [%s]", apiKey)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return &entities.AuthUser{
		ID:    user.ID,
		Email: user.Email,
	}, nil
}

func (repository *gormUserRepository) Load(ctx context.Context, userID entities.UserID) (*entities.User, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	message := new(entities.User)
	err := repository.db.First(message, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("user with ID [%s] does not exist", message.ID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load user with ID [%s]", userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return message, nil
}

func (repository *gormUserRepository) LoadOrStore(ctx context.Context, authUser entities.AuthUser) (*entities.User, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	user, err := repository.Load(ctx, authUser.ID)
	if err == nil {
		return user, nil
	}

	apiKey, err := repository.generateAPIKey(64)
	if err != nil {
		return nil, stacktrace.Propagate(err, "cannot generate apiKey")
	}

	user = &entities.User{
		ID:            authUser.ID,
		Email:         authUser.Email,
		APIKey:        apiKey,
		ActivePhoneID: nil,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	err = crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error {
		return tx.Where(entities.User{ID: user.ID}).FirstOrCreate(user).Error
	})

	if err != nil {
		msg := fmt.Sprintf("cannot create user from auth user [%+#v]", authUser)
		return user, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return user, nil
}

// generateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func (repository *gormUserRepository) generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot generate [%d] random bytes", n))
	}

	return b, nil
}

// generateAPIKey returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func (repository *gormUserRepository) generateAPIKey(n int) (string, error) {
	b, err := repository.generateRandomBytes(n)
	return base64.URLEncoding.EncodeToString(b)[0:n], stacktrace.Propagate(err, "cannot generate random bytes")
}
