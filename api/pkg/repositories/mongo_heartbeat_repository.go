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
	"github.com/palantir/stacktrace"
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
	client *mongo.Client,
) HeartbeatRepository {
	return &mongoHeartbeatRepository{
		logger:     logger.WithService(fmt.Sprintf("%T", &mongoHeartbeatRepository{})),
		tracer:     tracer,
		collection: client.Database(mongoDBName).Collection(collectionHeartbeats),
	}
}

type heartbeatDocument struct {
	ID        string    `bson:"_id"`
	Owner     string    `bson:"owner"`
	Version   string    `bson:"version"`
	Charging  bool      `bson:"charging"`
	UserID    string    `bson:"user_id"`
	Timestamp time.Time `bson:"timestamp"`
}

func (repository *mongoHeartbeatRepository) Store(ctx context.Context, heartbeat *entities.Heartbeat) error {
	ctx, span, _ := repository.tracer.StartWithLogger(ctx, repository.logger)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	doc := heartbeatDocument{
		ID:        heartbeat.ID.String(),
		Owner:     heartbeat.Owner,
		Version:   heartbeat.Version,
		Charging:  heartbeat.Charging,
		UserID:    string(heartbeat.UserID),
		Timestamp: heartbeat.Timestamp.UTC(),
	}

	_, err := repository.collection.InsertOne(ctx, doc)
	if err != nil {
		msg := fmt.Sprintf("cannot save heartbeat with ID [%s]", heartbeat.ID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (repository *mongoHeartbeatRepository) Index(ctx context.Context, userID entities.UserID, owner string, params IndexParams) (*[]entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{"user_id", string(userID)},
		{"owner", owner},
	}

	if len(params.Query) > 0 {
		filter = append(filter, bson.E{"version", bson.D{{"$regex", params.Query}, {"$options", "i"}}})
	}

	opts := options.Find().
		SetSort(bson.D{{"timestamp", -1}}).
		SetSkip(int64(params.Skip)).
		SetLimit(int64(params.Limit))

	cursor, err := repository.collection.Find(ctx, filter, opts)
	if err != nil {
		msg := fmt.Sprintf("cannot fetch heartbeats with owner [%s] and params [%+#v]", owner, params)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}
	defer cursor.Close(ctx)

	var docs []heartbeatDocument
	if err = cursor.All(ctx, &docs); err != nil {
		msg := fmt.Sprintf("cannot decode heartbeats for owner [%s]", owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	heartbeats := make([]entities.Heartbeat, 0, len(docs))
	for _, doc := range docs {
		hb, convertErr := docToHeartbeat(doc)
		if convertErr != nil {
			msg := fmt.Sprintf("cannot convert heartbeat document for owner [%s]", owner)
			return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(convertErr, msg))
		}
		heartbeats = append(heartbeats, *hb)
	}

	return &heartbeats, nil
}

func (repository *mongoHeartbeatRepository) Last(ctx context.Context, userID entities.UserID, owner string) (*entities.Heartbeat, error) {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	filter := bson.D{
		{"user_id", string(userID)},
		{"owner", owner},
	}

	opts := options.FindOne().SetSort(bson.D{{"timestamp", -1}})

	var doc heartbeatDocument
	err := repository.collection.FindOne(ctx, filter, opts).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		msg := fmt.Sprintf("heartbeat with userID [%s] and owner [%s] does not exist", userID, owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, ErrCodeNotFound, msg))
	}
	if err != nil {
		msg := fmt.Sprintf("cannot load heartbeat with userID [%s] and owner [%s]", userID, owner)
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	heartbeat, err := docToHeartbeat(doc)
	if err != nil {
		return nil, repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot convert heartbeat document"))
	}

	return heartbeat, nil
}

func (repository *mongoHeartbeatRepository) DeleteAllForUser(ctx context.Context, userID entities.UserID) error {
	ctx, span := repository.tracer.Start(ctx)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, dbOperationDuration)
	defer cancel()

	_, err := repository.collection.DeleteMany(ctx, bson.D{{"user_id", string(userID)}})
	if err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s]", &entities.Heartbeat{}, userID)
		return repository.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func docToHeartbeat(doc heartbeatDocument) (*entities.Heartbeat, error) {
	id, err := uuid.Parse(doc.ID)
	if err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot parse heartbeat ID [%s]", doc.ID))
	}
	return &entities.Heartbeat{
		ID:        id,
		Owner:     doc.Owner,
		Version:   doc.Version,
		Charging:  doc.Charging,
		UserID:    entities.UserID(doc.UserID),
		Timestamp: doc.Timestamp,
	}, nil
}
