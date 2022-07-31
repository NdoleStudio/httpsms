package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormPhoneRepository is responsible for persisting entities.Phone
type gormPhoneRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormPhoneRepository creates the GORM version of the PhoneRepository
func NewGormPhoneRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) PhoneRepository {
	return &gormPhoneRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormPhoneRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// LoadByID loads a phone by MessageID
func (repository *gormPhoneRepository) LoadByID(ctx context.Context, phoneID uuid.UUID) (*entities.Phone, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	phone := new(entities.Phone)
	err := repository.db.WithContext(ctx).First(phone, phoneID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("phone with MessageID [%s] does not exist", phoneID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load phone with MessageID [%s]", phoneID)
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
		msg := fmt.Sprintf("cannot delete phone with MessageID [%s] and userID [%s]", phoneID, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Save a new entities.Phone
func (repository *gormPhoneRepository) Save(ctx context.Context, phone *entities.Phone) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(phone).Error; err != nil {
		msg := fmt.Sprintf("cannot save phone with MessageID [%s]", phone.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Load a phone based on entities.UserID and phoneNumber
func (repository *gormPhoneRepository) Load(ctx context.Context, userID entities.UserID, phoneNumber string) (*entities.Phone, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

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
