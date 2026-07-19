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

// gormContactRepository is responsible for persisting entities.Contact.
type gormContactRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormContactRepository creates the GORM version of the ContactRepository.
func NewGormContactRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) ContactRepository {
	return &gormContactRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormContactRepository{})),
		tracer: tracer,
		db:     db,
	}
}

func (repository *gormContactRepository) Store(ctx context.Context, contacts []*entities.Contact) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if len(contacts) == 0 {
		return nil
	}

	if err := repository.db.WithContext(ctx).Create(&contacts).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot store [%d] contacts", len(contacts)))
	}

	return nil
}

func (repository *gormContactRepository) Update(ctx context.Context, contact *entities.Contact) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(contact).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update contact with ID [%s]", contact.ID))
	}

	return nil
}

func (repository *gormContactRepository) Load(ctx context.Context, userID entities.UserID, contactID uuid.UUID) (*entities.Contact, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	contact := new(entities.Contact)
	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", contactID).
		First(contact).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "contact with ID [%s] for user [%s] does not exist", contactID, userID))
	}

	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load contact with ID [%s] for user [%s]", contactID, userID))
	}

	return contact, nil
}

func (repository *gormContactRepository) Index(ctx context.Context, userID entities.UserID, params IndexParams) (*[]entities.Contact, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	query := repository.db.WithContext(ctx).Where("user_id = ?", userID)
	if len(params.Query) > 0 {
		queryPattern := "%" + params.Query + "%"
		query = query.Where(
			repository.db.WithContext(ctx).Where("name ILIKE ?", queryPattern).
				Or("array_to_string(emails, ',') ILIKE ?", queryPattern).
				Or("array_to_string(phone_numbers, ',') ILIKE ?", queryPattern),
		)
	}

	contacts := new([]entities.Contact)
	if err := query.Order("updated_at DESC").Limit(params.Limit).Offset(params.Skip).Find(contacts).Error; err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot index contacts for user [%s] with params [%+#v]", userID, params))
	}

	return contacts, nil
}

func (repository *gormContactRepository) FetchAll(ctx context.Context, userID entities.UserID) (*[]entities.Contact, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	contacts := new([]entities.Contact)
	if err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at ASC").
		Find(contacts).Error; err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot fetch all contacts for user [%s]", userID))
	}

	return contacts, nil
}

func (repository *gormContactRepository) Delete(ctx context.Context, userID entities.UserID, contactID uuid.UUID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	err := repository.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("id = ?", contactID).
		Delete(&entities.Contact{}).Error
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete contact with ID [%s] for user [%s]", contactID, userID))
	}

	return nil
}

func (repository *gormContactRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.Contact{}).Error; err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete all contacts for user [%s]", userID))
	}

	return nil
}
