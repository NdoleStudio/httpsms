package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// GormEvent is a serialized version of cloudevents.Event
type GormEvent struct {
	ID     uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;"`
	Time   time.Time
	Source string
	Type   string
	Data   datatypes.JSON
}

// TableName overrides the table name used by GormEvent to `events`
func (GormEvent) TableName() string {
	return "events"
}

type gormEventRepository struct {
	logger telemetry.Logger
	tracer telemetry.Tracer
	db     *gorm.DB
}

// NewGormEventRepository creates the GORM version of the EventRepository
func NewGormEventRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *gorm.DB,
) EventRepository {
	return &gormEventRepository{
		logger: logger.WithService(fmt.Sprintf("%T", &gormEventRepository{})),
		tracer: tracer,
		db:     db,
	}
}

// FetchAll returns all cloudevents.Event ordered by time in ascending order
func (repository *gormEventRepository) FetchAll(ctx context.Context) (*[]cloudevents.Event, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	var events []GormEvent
	if err := repository.db.WithContext(ctx).Order("time ASC").Find(&events).Error; err != nil {
		msg := fmt.Sprintf("cannot fetch all cloudevents")
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	results := make([]cloudevents.Event, 0, len(events))
	for _, event := range events {
		var cloudevent cloudevents.Event
		if err := json.Unmarshal(event.Data, &cloudevent); err != nil {
			msg := fmt.Sprintf("cannot unmarshal [%s] into [%T]", event.Data, cloudevent)
			return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
		}
		results = append(results, cloudevent)
	}
	return &results, nil
}

// Save saves a new cloudevents.Event
func (repository *gormEventRepository) Save(ctx context.Context, event cloudevents.Event) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	data, err := event.MarshalJSON()
	if err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot marshall event [%s]  and type [%s] into JSON", event.ID(), event.Type()))
	}

	gormEvent := GormEvent{
		ID:     uuid.MustParse(event.ID()),
		Time:   event.Time(),
		Source: event.Source(),
		Type:   event.Type(),
		Data:   datatypes.JSON(data),
	}

	if err = repository.db.WithContext(ctx).Create(gormEvent).Error; err != nil {
		return stacktrace.Propagate(err, fmt.Sprintf("cannot save event [%s] and type [%s]", event.ID(), event.Type()))
	}

	return nil
}
