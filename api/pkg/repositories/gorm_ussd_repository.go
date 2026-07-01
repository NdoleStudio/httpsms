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

// gormUSSDRepository is responsible for persisting entities.USSD
type gormUSSDRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormUSSDRepository creates the GORM version of the USSDRepository
func NewGormUSSDRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) USSDRepository {
	return &gormUSSDRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormUSSDRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// Store saves a new USSD session
func (repository *gormUSSDRepository) Store(ctx context.Context, ussd *entities.USSD) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Create(ussd).Error; err != nil {
		msg := fmt.Sprintf("cannot save USSD session with ID [%s]", ussd.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Update updates an existing USSD session
func (repository *gormUSSDRepository) Update(ctx context.Context, ussd *entities.USSD) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(ussd).Error; err != nil {
		msg := fmt.Sprintf("cannot update USSD session with ID [%s]", ussd.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// Load loads a USSD session by ID and user ID
func (repository *gormUSSDRepository) Load(ctx context.Context, userID entities.UserID, ussdID uuid.UUID) (*entities.USSD, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ussd := new(entities.USSD)
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", ussdID).First(ussd).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("USSD session with ID [%s] and userID [%s] does not exist", ussdID, userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load USSD session with ID [%s]", ussdID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return ussd, nil
}

// LoadBySessionID loads a USSD session by session ID and user ID
func (repository *gormUSSDRepository) LoadBySessionID(ctx context.Context, userID entities.UserID, sessionID string) (*entities.USSD, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ussd := new(entities.USSD)
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("session_id = ?", sessionID).Order("created_at DESC").First(ussd).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := fmt.Sprintf("USSD session with sessionID [%s] and userID [%s] does not exist", sessionID, userID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}

	if err != nil {
		msg := fmt.Sprintf("cannot load USSD session with sessionID [%s]", sessionID)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return ussd, nil
}

// Index fetches paginated USSD sessions for a user
func (repository *gormUSSDRepository) Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.USSD, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.
		WithContext(ctx).
		Where("user_id = ?", userID)

	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		subQuery := repository.db.Where("content ILIKE ?", queryPattern).
			Or("session_id ILIKE ?", queryPattern).
			Or("owner ILIKE ?", queryPattern)

		if _, err := uuid.Parse(params.Query); err == nil {
			subQuery = subQuery.Or("id = ?", params.Query)
		}

		query = query.Where(subQuery)
	}

	ussds := new([]entities.USSD)
	if err := query.Order("created_at DESC").Limit(params.Limit).Offset(params.Skip).Find(&ussds).Error; err != nil {
		msg := fmt.Sprintf("cannot fetch USSD sessions for user [%s] with params [%+#v]", userID, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return ussds, nil
}

// IndexByPhoneID fetches paginated USSD sessions for a phone
func (repository *gormUSSDRepository) IndexByPhoneID(ctx context.Context, userID entities.UserID, phoneID uuid.UUID, params IndexParams) (*[]entities.USSD, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Where("phone_id = ?", phoneID)

	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		subQuery := repository.db.Where("content ILIKE ?", queryPattern).
			Or("session_id ILIKE ?", queryPattern)

		query = query.Where(subQuery)
	}

	ussds := new([]entities.USSD)
	if err := query.Order("created_at DESC").Limit(params.Limit).Offset(params.Skip).Find(&ussds).Error; err != nil {
		msg := fmt.Sprintf("cannot fetch USSD sessions for phone [%s] with params [%+#v]", phoneID, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return ussds, nil
}

// Delete deletes a USSD session by ID and user ID
func (repository *gormUSSDRepository) Delete(ctx context.Context, userID entities.UserID, ussdID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Where("id = ?", ussdID).Delete(&entities.USSD{}).Error
	if err != nil {
		msg := fmt.Sprintf("cannot delete USSD session with ID [%s] for user with ID [%s]", ussdID, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// DeleteAllForPhone deletes all USSD sessions for a phone
func (repository *gormUSSDRepository) DeleteAllForPhone(ctx context.Context, phoneID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("phone_id = ?", phoneID).Delete(&entities.USSD{}).Error; err != nil {
		msg := fmt.Sprintf("cannot delete all USSD sessions for phone with ID [%s]", phoneID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}