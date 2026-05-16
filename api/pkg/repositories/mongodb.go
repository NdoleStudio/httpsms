package repositories

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/palantir/stacktrace"
)

const (
	collectionHeartbeats        = "heartbeats"
	collectionHeartbeatMonitors = "heartbeat_monitors"
)

// uuidEncodeValue encodes uuid.UUID as a BSON string
func uuidEncodeValue(_ bson.EncodeContext, vw bson.ValueWriter, val reflect.Value) error {
	u := val.Interface().(uuid.UUID)
	return vw.WriteString(u.String())
}

// uuidDecodeValue decodes a BSON string into uuid.UUID
func uuidDecodeValue(_ bson.DecodeContext, vr bson.ValueReader, val reflect.Value) error {
	str, err := vr.ReadString()
	if err != nil {
		return err
	}
	parsed, err := uuid.Parse(str)
	if err != nil {
		return err
	}
	val.Set(reflect.ValueOf(parsed))
	return nil
}

// newMongoRegistry creates a BSON registry that encodes uuid.UUID as strings
func newMongoRegistry() *bson.Registry {
	rb := bson.NewRegistry()
	rb.RegisterTypeEncoder(reflect.TypeOf(uuid.UUID{}), bson.ValueEncoderFunc(uuidEncodeValue))
	rb.RegisterTypeDecoder(reflect.TypeOf(uuid.UUID{}), bson.ValueDecoderFunc(uuidDecodeValue))
	return rb
}

// NewMongoDB creates a new *mongo.Database connection to MongoDB Atlas and ensures indexes.
// The database name is derived from the appName query parameter in the URI.
func NewMongoDB(uri string) (*mongo.Database, error) {
	dbName, err := parseMongoDBName(uri)
	if err != nil {
		return nil, stacktrace.Propagate(err, "cannot parse database name from MongoDB URI")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	registry := newMongoRegistry()
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI).SetRegistry(registry)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, stacktrace.Propagate(err, "cannot connect to MongoDB Atlas")
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot ping MongoDB with URI [%s]", uri))
	}

	db := client.Database(dbName)

	if err = createMongoIndexes(ctx, db); err != nil {
		return nil, stacktrace.Propagate(err, "cannot create MongoDB indexes")
	}

	return db, nil
}

// parseMongoDBName extracts the appName query parameter from the MongoDB URI to use as the database name
func parseMongoDBName(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", stacktrace.Propagate(err, fmt.Sprintf("cannot parse MongoDB URI [%s]", uri))
	}

	appName := parsed.Query().Get("appName")
	if appName == "" {
		return "", stacktrace.NewError("MongoDB URI is missing the 'appName' query parameter which is used as the database name")
	}

	return appName, nil
}

func createMongoIndexes(ctx context.Context, db *mongo.Database) error {
	// Heartbeats indexes
	heartbeatsCol := db.Collection(collectionHeartbeats)
	_, err := heartbeatsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{"owner", 1}, {"timestamp", -1}}},
		{Keys: bson.D{{"user_id", 1}}},
	})
	if err != nil {
		return stacktrace.Propagate(err, "cannot create indexes on heartbeats collection")
	}

	// Heartbeat monitors indexes
	monitorsCol := db.Collection(collectionHeartbeatMonitors)
	_, err = monitorsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{"user_id", 1}, {"owner", 1}}},
	})
	if err != nil {
		return stacktrace.Propagate(err, "cannot create indexes on heartbeat_monitors collection")
	}

	return nil
}
