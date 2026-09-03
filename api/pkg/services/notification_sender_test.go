package services

import (
	"context"
	"testing"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingFCMClient struct {
	message *messaging.Message
	result  string
	err     error
	calls   int
}

func (client *recordingFCMClient) Send(_ context.Context, message *messaging.Message) (string, error) {
	client.calls++
	client.message = message
	return client.result, client.err
}

func TestFCMNotificationSenderPassesMessageUnchanged(t *testing.T) {
	ttl := 5 * time.Minute
	data := map[string]string{"KEY_MESSAGE_ID": uuid.NewString()}
	client := &recordingFCMClient{result: "projects/test/messages/1"}
	sender := NewFCMNotificationSender(client)
	message := &messaging.Message{
		Token: "fcm-token:value",
		Data:  data,
		Android: &messaging.AndroidConfig{
			Priority: "normal",
			TTL:      &ttl,
		},
	}

	result, err := sender.Send(context.Background(), message, uuid.New())

	require.NoError(t, err)
	assert.Equal(t, "projects/test/messages/1", result)
	require.Equal(t, 1, client.calls)
	assert.Same(t, message, client.message)
	assert.Equal(t, "fcm-token:value", client.message.Token)
	assert.Equal(t, map[string]string{"KEY_MESSAGE_ID": data["KEY_MESSAGE_ID"]}, client.message.Data)
	require.NotNil(t, client.message.Android)
	assert.Equal(t, "normal", client.message.Android.Priority)
	assert.Equal(t, &ttl, client.message.Android.TTL)
	assert.Equal(t, map[string]string{"KEY_MESSAGE_ID": data["KEY_MESSAGE_ID"]}, data)
}

func TestFCMNotificationSenderRejectsNilMessage(t *testing.T) {
	client := &recordingFCMClient{}
	sender := NewFCMNotificationSender(client)

	_, err := sender.Send(context.Background(), nil, uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification message is nil")
	assert.Zero(t, client.calls)
}
