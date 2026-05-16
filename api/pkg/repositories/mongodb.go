package repositories

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/palantir/stacktrace"
)

const (
	mongoDBName                 = "httpsms"
	collectionHeartbeats        = "heartbeats"
	collectionHeartbeatMonitors = "heartbeat_monitors"
)

// NewMongoDB creates a new *mongo.Client connection to MongoDB Atlas and ensures indexes
func NewMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, stacktrace.Propagate(err, "cannot connect to MongoDB Atlas")
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, stacktrace.Propagate(err, fmt.Sprintf("cannot ping MongoDB with URI [%s]", uri))
	}

	if err = createMongoIndexes(ctx, client); err != nil {
		return nil, stacktrace.Propagate(err, "cannot create MongoDB indexes")
	}

	return client, nil
}

func createMongoIndexes(ctx context.Context, client *mongo.Client) error {
	db := client.Database(mongoDBName)

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
