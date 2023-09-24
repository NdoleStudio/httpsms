package repositories

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/palantir/stacktrace"
	"gorm.io/gorm"
)

// gormIntegration3CxRepository is responsible for persisting entities.Integration3CX
type gormIntegration3CxRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormIntegration3CXRepository creates the GORM version of the Integration3CxRepository
func NewGormIntegration3CXRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) Integration3CxRepository {
	return &gormIntegration3CxRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormIntegration3CxRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// Save an entities.Integration3CX
func (repository *gormIntegration3CxRepository) Save(ctx context.Context, integration *entities.Integration3CX) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	if err := repository.db.WithContext(ctx).Save(integration).Error; err != nil {
		msg := fmt.Sprintf("cannot save [%T] with ID [%s]", integration, integration.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
