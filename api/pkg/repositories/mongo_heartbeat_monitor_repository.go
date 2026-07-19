package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
)

// mongoHeartbeatMonitorRepository is responsible for persisting entities.HeartbeatMonitor in MongoDB
type mongoHeartbeatMonitorRepository struct {
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	collection *mongo.Collection
}

// NewMongoHeartbeatMonitorRepository creates the MongoDB version of the HeartbeatMonitorRepository
func NewMongoHeartbeatMonitorRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *mongo.Database,
) HeartbeatMonitorRepository {
	return &mongoHeartbeatMonitorRepository{
		logger:     logger.WithService(fmt.Sprintf("%T", &mongoHeartbeatMonitorRepository{})),
		tracer:     tracer,
		collection: db.Collection(collectionHeartbeatMonitors),
	}
}

func (repository *mongoHeartbeatMonitorRepository) Store(ctx context.Context, monitor *entities.HeartbeatMonitor) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.collection.InsertOne(ctx, monitor)
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot save heartbeat monitor with ID [%s]", monitor.ID))
	}

	return nil
}

func (repository *mongoHeartbeatMonitorRepository) Load(ctx context.Context, userID entities.UserID, phoneNumber string) (*entities.HeartbeatMonitor, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{Key: "user_id", Value: string(userID)},
		{Key: "owner", Value: phoneNumber},
	}

	var monitor entities.HeartbeatMonitor
	err := repository.collection.FindOne(ctx, filter).Decode(&monitor)
	if err == mongo.ErrNoDocuments {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, "heartbeat monitor with userID [%s] and owner [%s] does not exist", userID, phoneNumber))
	}
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot load heartbeat monitor with userID [%s] and owner [%s]", userID, phoneNumber))
	}

	return &monitor, nil
}

func (repository *mongoHeartbeatMonitorRepository) Exists(ctx context.Context, userID entities.UserID, monitorID uuid.UUID) (bool, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{Key: "user_id", Value: string(userID)},
		{Key: "_id", Value: monitorID.String()},
	}

	count, err := repository.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot check if heartbeat monitor exists with userID [%s] and monitor ID [%s]", userID, monitorID))
	}

	return count > 0, nil
}

func (repository *mongoHeartbeatMonitorRepository) UpdateQueueID(ctx context.Context, monitorID uuid.UUID, queueID string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{{Key: "_id", Value: monitorID.String()}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "queue_id", Value: queueID},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}

	_, err := repository.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot update heartbeat monitor ID [%s]", monitorID))
	}

	return nil
}

func (repository *mongoHeartbeatMonitorRepository) Delete(ctx context.Context, userID entities.UserID, phoneNumber string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{Key: "user_id", Value: string(userID)},
		{Key: "owner", Value: phoneNumber},
	}

	_, err := repository.collection.DeleteMany(ctx, filter)
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete heartbeat monitor with owner [%s] and userID [%s]", phoneNumber, userID))
	}

	return nil
}

func (repository *mongoHeartbeatMonitorRepository) UpdatePhoneOnline(ctx context.Context, userID entities.UserID, monitorID uuid.UUID, online bool) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{Key: "_id", Value: monitorID.String()},
		{Key: "user_id", Value: string(userID)},
	}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "phone_online", Value: online},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}

	_, err := repository.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot update heartbeat monitor ID [%s] for user [%s]", monitorID, userID))
	}

	return nil
}

func (repository *mongoHeartbeatMonitorRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.collection.DeleteMany(ctx, bson.D{{Key: "user_id", Value: string(userID)}})
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete all [%T] for user with ID [%s]", &entities.HeartbeatMonitor{}, userID))
	}

	return nil
}
