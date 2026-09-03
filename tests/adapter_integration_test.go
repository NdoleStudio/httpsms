package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	httpsms "github.com/NdoleStudio/httpsms-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapterGatewayOutgoingMessage(t *testing.T) {
	ctx := context.Background()
	phone := setupAdapterPhone(ctx, t, 60)
	contact := randomPhoneNumber()
	content := "Adapter outgoing " + randomEncryptionKey()

	response, httpResponse, err := newAPIClient().Messages.Send(ctx, &httpsms.MessageSendParams{
		From:    phone.PhoneNumber,
		To:      contact,
		Content: content,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, httpResponse.HTTPResponse.StatusCode)

	messageID := response.Data.ID.String()
	message := pollMessageStatus(ctx, t, messageID, "delivered", 30*time.Second)

	assert.Equal(t, phone.PhoneNumber, message.Owner)
	assert.Equal(t, contact, message.Contact)
	assert.Equal(t, content, message.Content)
	records := waitForAdapterMessageRecords(t, phone.GatewayID, messageID, 30*time.Second)
	require.Len(t, records, 1)
	assert.Equal(t, "message", records[0].Kind)
	assert.True(t, records[0].Processed)
	assert.Equal(t, messageID, records[0].Data["KEY_MESSAGE_ID"])
	assert.NotEmpty(t, records[0].NotificationID)
}

func TestAdapterGatewayIncomingMessage(t *testing.T) {
	ctx := context.Background()
	phone := setupAdapterPhone(ctx, t, 60)
	contact := randomPhoneNumber()
	content := "Adapter incoming " + randomEncryptionKey()

	messageID := triggerAdapterIncoming(ctx, t, phone, contact, content)
	message := pollMessageStatus(ctx, t, messageID, "received", 15*time.Second)

	assert.Equal(t, phone.PhoneNumber, message.Owner)
	assert.Equal(t, contact, message.Contact)
	assert.Equal(t, content, message.Content)
	assert.Equal(t, "received", message.Status)
}

func TestAdapterGatewayHeartbeatWakeUp(t *testing.T) {
	ctx := context.Background()
	phone := setupAdapterPhone(ctx, t, 60)
	monitorID := uuid.NewString()

	dispatchInternalEvent(ctx, t, map[string]any{
		"specversion":     "1.0",
		"id":              uuid.NewString(),
		"source":          "/tests/adapter-emulator",
		"type":            "phone.heartbeat.missed",
		"time":            time.Now().UTC().Format(time.RFC3339),
		"datacontenttype": "application/json",
		"data": map[string]any{
			"phone_id":                 phone.PhoneID,
			"user_id":                  "test-user-id",
			"last_heartbeat_timestamp": time.Now().UTC().Add(-20 * time.Minute).Format(time.RFC3339),
			"timestamp":                time.Now().UTC().Format(time.RFC3339),
			"monitor_id":               monitorID,
			"owner":                    phone.PhoneNumber,
		},
	})

	record := waitForAdapterHeartbeatRecord(t, phone.GatewayID, 30*time.Second)
	assert.Equal(t, "heartbeat", record.Kind)
	assert.NotEmpty(t, record.Data["KEY_HEARTBEAT_ID"])

	heartbeats, response, err := newAPIClient().Heartbeats.Index(ctx, &httpsms.HeartbeatIndexParams{
		Owner: phone.PhoneNumber,
		Limit: 1,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	require.NotEmpty(t, heartbeats.Data)
	assert.Equal(t, phone.PhoneNumber, heartbeats.Data[0].Owner)
}
