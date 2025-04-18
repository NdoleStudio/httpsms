package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// PhoneAPIKeyService is responsible for managing entities.PhoneAPIKey
type PhoneAPIKeyService struct {
	service
	logger          telemetry.Logger
	tracer          telemetry.Tracer
	phoneRepository repositories.PhoneRepository
	repository      repositories.PhoneAPIKeyRepository
}

// NewPhoneAPIKeyService creates a new PhoneAPIKeyService
func NewPhoneAPIKeyService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	phoneRepository repositories.PhoneRepository,
	repository repositories.PhoneAPIKeyRepository,
) *PhoneAPIKeyService {
	return &PhoneAPIKeyService{
		logger:          logger.WithService(fmt.Sprintf("%T", &PhoneAPIKeyService{})),
		tracer:          tracer,
		phoneRepository: phoneRepository,
		repository:      repository,
	}
}

// Create a new entities.PhoneAPIKey
func (service *PhoneAPIKeyService) Create(ctx context.Context, authContext entities.AuthContext, name string) (*entities.PhoneAPIKey, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	apiKey, err := service.generateAPIKey(64)
	if err != nil {
		return nil, stacktrace.Propagate(err, "cannot generate API key")
	}

	phoneAPIKey := &entities.PhoneAPIKey{
		ID:           uuid.New(),
		Name:         name,
		UserID:       authContext.ID,
		UserEmail:    authContext.Email,
		PhoneNumbers: nil,
		PhoneIDs:     nil,
		APIKey:       apiKey,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err = service.repository.Create(ctx, phoneAPIKey); err != nil {
		msg := fmt.Sprintf("cannot create PhoneAPIKey for user [%s]", authContext.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return phoneAPIKey, nil
}

// Delete an entities.PhoneAPIKey
func (service *PhoneAPIKeyService) Delete(ctx context.Context, userID entities.UserID, phoneAPIKeyID uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	phoneAPIKey, err := service.repository.Load(ctx, userID, phoneAPIKeyID)
	if err != nil {
		msg := fmt.Sprintf("cannot load [%T] with ID [%s] for user [%s]", &entities.PhoneAPIKey{}, phoneAPIKeyID, userID.String())
		return stacktrace.Propagate(err, msg)
	}

	if err = service.repository.Delete(ctx, phoneAPIKey); err != nil {
		msg := fmt.Sprintf("cannot delete [%T] with ID [%s] for user [%s]", phoneAPIKey, phoneAPIKey.ID, phoneAPIKey.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// RemovePhone removes the phone from the phone API key
func (service *PhoneAPIKeyService) RemovePhone(ctx context.Context, userID entities.UserID, phoneAPIKeyID uuid.UUID, phoneID uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	phone, err := service.phoneRepository.LoadByID(ctx, userID, phoneID)
	if err != nil {
		msg := fmt.Sprintf("cannot load [%T] with ID [%s] for user [%s]", &entities.Phone{}, phoneID, userID.String())
		return stacktrace.Propagate(err, msg)
	}

	phoneAPIKey, err := service.repository.Load(ctx, userID, phoneAPIKeyID)
	if err != nil {
		msg := fmt.Sprintf("cannot load [%T] with ID [%s] for user [%s]", &entities.PhoneAPIKey{}, phoneAPIKeyID, userID.String())
		return stacktrace.Propagate(err, msg)
	}

	if err = service.repository.RemovePhone(ctx, phoneAPIKey, phone); err != nil {
		msg := fmt.Sprintf("cannot remove [%T] with ID [%s] from phone API key with ID [%s] for user [%s]", phone, phone.ID, phoneAPIKey.ID, userID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (service *PhoneAPIKeyService) generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	// Note that err == nil only if we read len(b) bytes.
	if _, err := rand.Read(b); err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot generate [%d] random bytes", n))
	}

	return b, nil
}

func (service *PhoneAPIKeyService) generateAPIKey(n int) (string, error) {
	b, err := service.generateRandomBytes(n)
	return base64.URLEncoding.EncodeToString(b)[0:n], stacktrace.Propagate(err, fmt.Sprintf("cannot generate [%s] random bytes", n))
}
