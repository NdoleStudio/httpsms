package services

import (
	"context"
	"testing"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingNotificationSender struct {
	destination  string
	notification GatewayNotification
	result       string
	err          error
	calls        int
}

func (sender *recordingNotificationSender) Send(_ context.Context, destination string, notification GatewayNotification) (string, error) {
	sender.calls++
	sender.destination = destination
	sender.notification = notification
	return sender.result, sender.err
}

func TestPhoneNotificationDispatcherRoutesFCMToken(t *testing.T) {
	token := "fcm-token:value"
	phone := &entities.Phone{FcmToken: &token}
	fcmSender := &recordingNotificationSender{result: "projects/test/messages/1"}
	httpSender := &recordingNotificationSender{}
	dispatcher := NewPhoneNotificationDispatcher(fcmSender, httpSender)
	notification := GatewayNotification{Data: map[string]string{"KEY_MESSAGE_ID": uuid.NewString()}}

	result, err := dispatcher.Send(context.Background(), phone, notification)

	require.NoError(t, err)
	assert.Equal(t, "projects/test/messages/1", result)
	assert.Equal(t, 1, fcmSender.calls)
	assert.Zero(t, httpSender.calls)
	assert.Equal(t, token, fcmSender.destination)
}

func TestPhoneNotificationDispatcherRoutesHTTPSURL(t *testing.T) {
	endpoint := "https://adapter.example.com/notifications/gateway-1"
	phone := &entities.Phone{FcmToken: &endpoint}
	fcmSender := &recordingNotificationSender{}
	httpSender := &recordingNotificationSender{result: "accepted"}
	dispatcher := NewPhoneNotificationDispatcher(fcmSender, httpSender)
	notification := GatewayNotification{Data: map[string]string{"KEY_MESSAGE_ID": uuid.NewString()}}

	result, err := dispatcher.Send(context.Background(), phone, notification)

	require.NoError(t, err)
	assert.Equal(t, "accepted", result)
	assert.Zero(t, fcmSender.calls)
	assert.Equal(t, 1, httpSender.calls)
	assert.Equal(t, endpoint, httpSender.destination)
}

func TestPhoneNotificationDispatcherRejectsInvalidURLLikeTokenWithoutSending(t *testing.T) {
	token := "https://"
	phone := &entities.Phone{FcmToken: &token}
	fcmSender := &recordingNotificationSender{}
	httpSender := &recordingNotificationSender{}
	dispatcher := NewPhoneNotificationDispatcher(fcmSender, httpSender)

	_, err := dispatcher.Send(context.Background(), phone, GatewayNotification{})

	require.Error(t, err)
	assert.Zero(t, fcmSender.calls)
	assert.Zero(t, httpSender.calls)
}

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

func TestFCMNotificationSenderMapsGatewayNotification(t *testing.T) {
	ttl := 5 * time.Minute
	notificationID := uuid.New()
	data := map[string]string{"KEY_MESSAGE_ID": uuid.NewString()}
	client := &recordingFCMClient{result: "projects/test/messages/1"}
	sender := NewFCMNotificationSender(client)

	result, err := sender.Send(context.Background(), "fcm-token:value", GatewayNotification{
		Data:           data,
		Priority:       "normal",
		TTL:            &ttl,
		NotificationID: notificationID,
	})

	require.NoError(t, err)
	assert.Equal(t, "projects/test/messages/1", result)
	require.Equal(t, 1, client.calls)
	require.NotNil(t, client.message)
	assert.Equal(t, "fcm-token:value", client.message.Token)
	assert.Equal(t, map[string]string{"KEY_MESSAGE_ID": data["KEY_MESSAGE_ID"]}, client.message.Data)
	require.NotNil(t, client.message.Android)
	assert.Equal(t, "normal", client.message.Android.Priority)
	assert.Equal(t, &ttl, client.message.Android.TTL)
	assert.Equal(t, map[string]string{"KEY_MESSAGE_ID": data["KEY_MESSAGE_ID"]}, data)
}
