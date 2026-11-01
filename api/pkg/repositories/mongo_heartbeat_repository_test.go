package repositories

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMongoHeartbeatLast_StableIdentityForEqualTimestamps(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		t.Run(fmt.Sprint(reversed), func(t *testing.T) {
			timestamp := time.Date(2026, 1, 1, 12, 0, 0, 123000000, time.UTC)
			low := bson.M{"_id": "00000000-0000-4000-8000-000000000001", "timestamp": timestamp, "user_id": "user-id", "owner": "+18005550199"}
			high := bson.M{"_id": "00000000-0000-4000-8000-000000000002", "timestamp": timestamp, "user_id": "user-id", "owner": "+18005550199"}
			higher := bson.M{"_id": "00000000-0000-4000-8000-000000000003", "timestamp": timestamp, "user_id": "user-id", "owner": "+18005550199"}
			older := bson.M{"_id": "00000000-0000-4000-8000-000000000000", "timestamp": timestamp.Add(-time.Second), "user_id": "user-id", "owner": "+18005550199"}
			documents := []bson.M{older, high, higher, low}
			if reversed {
				documents = []bson.M{low, higher, high, older}
			}
			repository := heartbeatMongoRepositoryForTest(t, documents)
			for range 2 {
				heartbeat, err := repository.Last(context.Background(), "user-id", "+18005550199")
				require.NoError(t, err)
				require.Equal(t, uuid.MustParse("00000000-0000-4000-8000-000000000001"), heartbeat.ID)
			}
		})
	}
}

func TestMongoHeartbeatLast_MissingAndSingleHistory(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprint(missing), func(t *testing.T) {
			id := uuid.New()
			documents := []bson.M{{"_id": id.String(), "timestamp": time.Now().UTC(), "user_id": "user-id", "owner": "+18005550199"}}
			if missing {
				documents = []bson.M{}
			}
			heartbeat, err := heartbeatMongoRepositoryForTest(t, documents).Last(context.Background(), "user-id", "+18005550199")
			if missing {
				require.Equal(t, ErrCodeNotFound, stacktrace.GetCode(err))
				require.Nil(t, heartbeat)
				return
			}
			require.NoError(t, err)
			require.Equal(t, id, heartbeat.ID)
		})
	}
}

func heartbeatMongoRepositoryForTest(t *testing.T, documents []bson.M) HeartbeatRepository {
	t.Helper()
	client, err := mongo.Connect(options.Client().SetHosts([]string{"memory:27017"}).
		SetDirect(true).SetRegistry(newMongoRegistry()).SetDialer(&heartbeatMongoTestDialer{documents: documents}).
		SetServerSelectionTimeout(time.Second))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Disconnect(context.Background())) })
	logger := &messageThreadTestLogger{}
	return NewMongoHeartbeatRepository(logger, telemetry.NewOtelLogger("test", logger), client.Database("test"))
}

// The driver exchanges real MongoDB commands over net.Pipe; no database or network socket is needed.
type heartbeatMongoTestDialer struct{ documents []bson.M }

func (dialer *heartbeatMongoTestDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	client, server := net.Pipe()
	go func() {
		defer func() { _ = server.Close() }()
		for {
			command, requestID, opcode, err := readHeartbeatMongoCommand(server)
			if err != nil {
				return
			}
			response := dialer.response(command)
			if err := writeHeartbeatMongoResponse(server, response, requestID, opcode); err != nil {
				return
			}
		}
	}()
	return client, nil
}

func (dialer *heartbeatMongoTestDialer) response(command bson.Raw) bson.D {
	if _, find := command.Lookup("find").StringValueOK(); !find {
		return bson.D{
			{Key: "ok", Value: 1},
			{Key: "isWritablePrimary", Value: true},
			{Key: "maxWireVersion", Value: 25},
			{Key: "maxBsonObjectSize", Value: 16777216},
			{Key: "maxMessageSizeBytes", Value: 48000000},
			{Key: "maxWriteBatchSize", Value: 100000},
		}
	}
	var order bson.D
	_ = bson.Unmarshal(command.Lookup("sort").Document(), &order)
	documents := []bson.M{}
	timestamp, bounded := command.Lookup("filter").Document().Lookup("timestamp").DateTimeOK()
	for _, document := range dialer.documents {
		if !bounded || document["timestamp"].(time.Time).UnixMilli() == timestamp {
			documents = append(documents, document)
		}
	}
	sort.SliceStable(documents, func(i, j int) bool {
		for _, key := range order {
			comparison := 0
			switch key.Key {
			case "timestamp":
				comparison = documents[i]["timestamp"].(time.Time).Compare(documents[j]["timestamp"].(time.Time))
			case "_id":
				comparison = strings.Compare(documents[i]["_id"].(string), documents[j]["_id"].(string))
			}
			if comparison != 0 {
				return comparison*int(key.Value.(int32)) < 0
			}
		}
		return false
	})
	limit := int(command.Lookup("limit").AsInt64())
	if limit > 0 && limit < len(documents) {
		documents = documents[:limit]
	}
	return bson.D{{Key: "ok", Value: 1}, {Key: "cursor", Value: bson.D{
		{Key: "id", Value: int64(0)},
		{Key: "ns", Value: "test.heartbeats"},
		{Key: "firstBatch", Value: documents},
	}}}
}

func readHeartbeatMongoCommand(connection net.Conn) (bson.Raw, int32, int32, error) {
	header := make([]byte, 16)
	if _, err := io.ReadFull(connection, header); err != nil {
		return nil, 0, 0, err
	}
	body := make([]byte, int(binary.LittleEndian.Uint32(header))-16)
	if _, err := io.ReadFull(connection, body); err != nil {
		return nil, 0, 0, err
	}
	opcode := int32(binary.LittleEndian.Uint32(header[12:]))
	offset := 5
	if opcode == 2004 {
		offset = 4 + bytes.IndexByte(body[4:], 0) + 1 + 8
	}
	return bson.Raw(body[offset:]), int32(binary.LittleEndian.Uint32(header[4:])), opcode, nil
}

func writeHeartbeatMongoResponse(connection net.Conn, response bson.D, requestID, opcode int32) error {
	document, err := bson.Marshal(response)
	if err != nil {
		return err
	}
	header := make([]byte, 16)
	body := make([]byte, 5)
	responseOpcode := uint32(2013)
	if opcode == 2004 {
		responseOpcode = 1
		body = make([]byte, 20)
		binary.LittleEndian.PutUint32(body[16:], 1)
	}
	body = append(body, document...)
	binary.LittleEndian.PutUint32(header, uint32(len(header)+len(body)))
	binary.LittleEndian.PutUint32(header[8:], uint32(requestID))
	binary.LittleEndian.PutUint32(header[12:], responseOpcode)
	_, err = connection.Write(append(header, body...))
	return err
}
