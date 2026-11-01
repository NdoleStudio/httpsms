package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

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
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot save heartbeat monitor with ID [%s]", monitor.ID))
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
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCodef(err, ErrCodeNotFound, "heartbeat monitor with userID [%s] and owner [%s] does not exist", userID, phoneNumber))
	}
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot load heartbeat monitor with userID [%s] and owner [%s]", userID, phoneNumber))
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
		return false, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot check if heartbeat monitor exists with userID [%s] and monitor ID [%s]", userID, monitorID))
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
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update heartbeat monitor ID [%s]", monitorID))
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
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete heartbeat monitor with owner [%s] and userID [%s]", phoneNumber, userID))
	}

	return nil
}

func (repository *mongoHeartbeatMonitorRepository) CompareAndSwapQueueID(ctx context.Context, monitorID uuid.UUID, expected, queueID string) (bool, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{Key: "_id", Value: monitorID.String()},
		{Key: "queue_id", Value: expected},
	}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "queue_id", Value: queueID},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}
	result, err := repository.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot claim heartbeat monitor ID [%s]", monitorID))
	}
	return result.MatchedCount == 1, nil
}

func (repository *mongoHeartbeatMonitorRepository) RecordHeartbeat(ctx context.Context, userID entities.UserID, monitorID, expected, heartbeatID uuid.UUID) (*entities.HeartbeatMonitor, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "last_heartbeat_id", Value: heartbeatID.String()},
		{Key: "phone_online", Value: true},
	}}}
	var previous entities.HeartbeatMonitor
	err := repository.collection.FindOneAndUpdate(ctx, heartbeatMonitorVersionFilter(userID, monitorID, expected), update,
		options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&previous)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot record heartbeat version for monitor [%s]", monitorID))
	}
	return &previous, nil
}

func (repository *mongoHeartbeatMonitorRepository) UpdatePhoneOfflineForHeartbeat(ctx context.Context, userID entities.UserID, monitorID, expected, heartbeatID uuid.UUID) (bool, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "last_heartbeat_id", Value: heartbeatID.String()},
		{Key: "phone_online", Value: false},
	}}}
	result, err := repository.collection.UpdateOne(ctx, heartbeatMonitorVersionFilter(userID, monitorID, expected), update)
	if err != nil {
		return false, repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot apply offline heartbeat version for monitor [%s]", monitorID))
	}
	return result.MatchedCount == 1, nil
}

func heartbeatMonitorVersionFilter(userID entities.UserID, monitorID, heartbeatID uuid.UUID) bson.D {
	filter := bson.D{{Key: "_id", Value: monitorID.String()}, {Key: "user_id", Value: string(userID)}}
	if heartbeatID == uuid.Nil {
		return append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "last_heartbeat_id", Value: uuid.Nil.String()}},
			bson.D{{Key: "last_heartbeat_id", Value: nil}},
		}})
	}
	return append(filter, bson.E{Key: "last_heartbeat_id", Value: heartbeatID.String()})
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
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot update heartbeat monitor ID [%s] for user [%s]", monitorID, userID))
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
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete all [%T] for user with ID [%s]", &entities.HeartbeatMonitor{}, userID))
	}

	return nil
}
