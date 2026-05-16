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
	"github.com/palantir/stacktrace"
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
		msg := fmt.Sprintf("cannot save heartbeat monitor with ID [%s]", monitor.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *mongoHeartbeatMonitorRepository) Load(ctx context.Context, userID entities.UserID, phoneNumber string) (*entities.HeartbeatMonitor, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{"user_id", string(userID)},
		{"owner", phoneNumber},
	}

	var monitor entities.HeartbeatMonitor
	err := repository.collection.FindOne(ctx, filter).Decode(&monitor)
	if err == mongo.ErrNoDocuments {
		msg := fmt.Sprintf("heartbeat monitor with userID [%s] and owner [%s] does not exist", userID, phoneNumber)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}
	if err != nil {
		msg := fmt.Sprintf("cannot load heartbeat monitor with userID [%s] and owner [%s]", userID, phoneNumber)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return &monitor, nil
}

func (repository *mongoHeartbeatMonitorRepository) Exists(ctx context.Context, userID entities.UserID, monitorID uuid.UUID) (bool, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{"user_id", string(userID)},
		{"_id", monitorID.String()},
	}

	count, err := repository.collection.CountDocuments(ctx, filter)
	if err != nil {
		msg := fmt.Sprintf("cannot check if heartbeat monitor exists with userID [%s] and monitor ID [%s]", userID, monitorID)
		return false, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return count > 0, nil
}

func (repository *mongoHeartbeatMonitorRepository) UpdateQueueID(ctx context.Context, monitorID uuid.UUID, queueID string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{{"_id", monitorID.String()}}
	update := bson.D{{"$set", bson.D{
		{"queue_id", queueID},
		{"updated_at", time.Now().UTC()},
	}}}

	_, err := repository.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		msg := fmt.Sprintf("cannot update heartbeat monitor ID [%s]", monitorID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *mongoHeartbeatMonitorRepository) Delete(ctx context.Context, userID entities.UserID, phoneNumber string) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{"user_id", string(userID)},
		{"owner", phoneNumber},
	}

	_, err := repository.collection.DeleteMany(ctx, filter)
	if err != nil {
		msg := fmt.Sprintf("cannot delete heartbeat monitor with owner [%s] and userID [%s]", phoneNumber, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *mongoHeartbeatMonitorRepository) UpdatePhoneOnline(ctx context.Context, userID entities.UserID, monitorID uuid.UUID, online bool) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{"_id", monitorID.String()},
		{"user_id", string(userID)},
	}
	update := bson.D{{"$set", bson.D{
		{"phone_online", online},
		{"updated_at", time.Now().UTC()},
	}}}

	_, err := repository.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		msg := fmt.Sprintf("cannot update heartbeat monitor ID [%s] for user [%s]", monitorID, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *mongoHeartbeatMonitorRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.collection.DeleteMany(ctx, bson.D{{"user_id", string(userID)}})
	if err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.HeartbeatMonitor{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
