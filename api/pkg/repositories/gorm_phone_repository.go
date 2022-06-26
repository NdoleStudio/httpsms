package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbgorm"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormPhoneRepository is responsible for persisting entities.Phone
type gormPhoneRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

func (repository *gormPhoneRepository) Upsert(ctx context.Context, phone *entities.Phone) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := crdbgorm.ExecuteTx(ctx, repository.db, nil, func(tx *gorm.DB) error {
		existingPhone := new(entities.Phone)

		err := tx.Model(&phone).
			Where("user_id = ?", phone.UserID).
			Where("phone_number = ?", phone.PhoneNumber).
			First(existingPhone).
			Error

		if err == nil {
			existingPhone.FcmToken = phone.FcmToken
			if err = tx.Save(existingPhone).Error; err != nil {
				return stacktrace.Propagate(err, fmt.Sprintf("cannot update exiting phone [%s]", existingPhone.ID))
			}
			*phone = *existingPhone
			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			msg := fmt.Sprintf("cannot find phone with user_Id [%s] and phone_number [%s]", phone.UserID, phone.PhoneNumber)
			return stacktrace.Propagate(err, msg)
		}

		return tx.Create(phone).Error
	})
	if err != nil {
		msg := fmt.Sprintf("cannot upsert phone with params [%+#v]", err)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *gormPhoneRepository) Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.Phone, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.Where("user_id = ?", userID)
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
