package repositories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestHeartbeatMonitorVersionFilter_FencesUserAndHeartbeatIncludingLegacyRows(t *testing.T) {
	monitorID, heartbeatID := uuid.New(), uuid.New()
	for _, expected := range []uuid.UUID{uuid.Nil, heartbeatID} {
		data, err := bson.Marshal(heartbeatMonitorVersionFilter("user-id", monitorID, expected))
		require.NoError(t, err)
		var filter bson.M
		require.NoError(t, bson.Unmarshal(data, &filter))
		require.Equal(t, monitorID.String(), filter["_id"])
		require.Equal(t, "user-id", filter["user_id"])
		if expected != uuid.Nil {
			require.Equal(t, heartbeatID.String(), filter["last_heartbeat_id"])
		} else {
			require.ElementsMatch(t, bson.A{
				bson.D{{Key: "last_heartbeat_id", Value: uuid.Nil.String()}},
				bson.D{{Key: "last_heartbeat_id", Value: nil}},
			}, filter["$or"], "legacy rows may encode zero UUID or have a missing/null version")
		}
	}
}
