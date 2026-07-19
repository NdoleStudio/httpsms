package repositories

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
)

// mongoHeartbeatRepository is responsible for persisting entities.Heartbeat in MongoDB
type mongoHeartbeatRepository struct {
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	collection *mongo.Collection
}

// NewMongoHeartbeatRepository creates the MongoDB version of the HeartbeatRepository
func NewMongoHeartbeatRepository(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	db *mongo.Database,
) HeartbeatRepository {
	return &mongoHeartbeatRepository{
		logger:     logger.WithService(fmt.Sprintf("%T", &mongoHeartbeatRepository{})),
		tracer:     tracer,
		collection: db.Collection(collectionHeartbeats),
	}
}

func (repository *mongoHeartbeatRepository) Store(ctx context.Context, heartbeat *entities.Heartbeat) error {
	ctx, span, _ := repository.tracer.StartWithLogger(ctx, repository.logger)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.collection.InsertOne(ctx, heartbeat)
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot save heartbeat with ID [%s]", heartbeat.ID))
	}

	return nil
}

func (repository *mongoHeartbeatRepository) Index(ctx context.Context, userID entities.UserID, owner string, params IndexParams) (*[]entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{Key: "user_id", Value: string(userID)},
		{Key: "owner", Value: owner},
	}

	if len(params.Query) > 0 {
		filter = append(filter, bson.E{Key: "version", Value: bson.D{{Key: "$regex", Value: params.Query}, {Key: "$options", Value: "i"}}})
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetSkip(int64(params.Skip)).
		SetLimit(int64(params.Limit))

	cursor, err := repository.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot fetch heartbeats with owner [%s] and params [%+#v]", owner, params))
	}
	defer cursor.Close(ctx)

	var heartbeats []entities.Heartbeat
	if err = cursor.All(ctx, &heartbeats); err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot decode heartbeats for owner [%s]", owner))
	}

	if heartbeats == nil {
		heartbeats = make([]entities.Heartbeat, 0)
	}

	return &heartbeats, nil
}

func (repository *mongoHeartbeatRepository) Last(ctx context.Context, userID entities.UserID, owner string) (*entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{Key: "user_id", Value: string(userID)},
		{Key: "owner", Value: owner},
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})

	var heartbeat entities.Heartbeat
	err := repository.collection.FindOne(ctx, filter, opts).Decode(&heartbeat)
	if err == mongo.ErrNoDocuments {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, "heartbeat with userID [%s] and owner [%s] does not exist", userID, owner))
	}
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot load heartbeat with userID [%s] and owner [%s]", userID, owner))
	}

	return &heartbeat, nil
}

func (repository *mongoHeartbeatRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.collection.DeleteMany(ctx, bson.D{{Key: "user_id", Value: string(userID)}})
	if err != nil {
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete all [%T] for user with ID [%s]", &entities.Heartbeat{}, userID))
	}

	return nil
}
