package services

import (
	"context"
	"testing"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingNotificationSender struct {
	destination    string
	message        *messaging.Message
	notificationID uuid.UUID
	result         string
	err            error
	calls          int
}

func (sender *recordingNotificationSender) Send(
	_ context.Context,
	message *messaging.Message,
	notificationID uuid.UUID,
) (string, error) {
	sender.calls++
	sender.message = message
	sender.notificationID = notificationID
	if message != nil {
		sender.destination = message.Token
	}
	return sender.result, sender.err
}

func TestPhoneNotificationDispatcherRoutesFCMToken(t *testing.T) {
	token := "  fcm-token:value  "
	phone := &entities.Phone{FcmToken: &token}
	fcmSender := &recordingNotificationSender{result: "projects/test/messages/1"}
	httpSender := &recordingNotificationSender{}
	dispatcher := NewPhoneNotificationDispatcher(fcmSender, httpSender)
	message := &messaging.Message{Data: map[string]string{"KEY_MESSAGE_ID": uuid.NewString()}}
	notificationID := uuid.New()

	result, err := dispatcher.Send(context.Background(), phone, message, notificationID)

	require.NoError(t, err)
	assert.Equal(t, "projects/test/messages/1", result)
	assert.Equal(t, 1, fcmSender.calls)
	assert.Zero(t, httpSender.calls)
	assert.Equal(t, "fcm-token:value", fcmSender.destination)
	assert.Equal(t, "fcm-token:value", message.Token)
	assert.Same(t, message, fcmSender.message)
	assert.Equal(t, notificationID, fcmSender.notificationID)
}

func TestPhoneNotificationDispatcherRoutesHTTPSURL(t *testing.T) {
	endpoint := "https://adapter.example.com/notifications/gateway-1"
	phone := &entities.Phone{FcmToken: &endpoint}
	fcmSender := &recordingNotificationSender{}
	httpSender := &recordingNotificationSender{result: "accepted"}
	dispatcher := NewPhoneNotificationDispatcher(fcmSender, httpSender)
	message := &messaging.Message{Data: map[string]string{"KEY_MESSAGE_ID": uuid.NewString()}}
	notificationID := uuid.New()

	result, err := dispatcher.Send(context.Background(), phone, message, notificationID)

	require.NoError(t, err)
	assert.Equal(t, "accepted", result)
	assert.Zero(t, fcmSender.calls)
	assert.Equal(t, 1, httpSender.calls)
	assert.Equal(t, endpoint, httpSender.destination)
	assert.Same(t, message, httpSender.message)
	assert.Equal(t, notificationID, httpSender.notificationID)
}

func TestPhoneNotificationDispatcherRejectsInvalidURLLikeTokenWithoutSending(t *testing.T) {
	token := "https://"
	phone := &entities.Phone{FcmToken: &token}
	fcmSender := &recordingNotificationSender{}
	httpSender := &recordingNotificationSender{}
	dispatcher := NewPhoneNotificationDispatcher(fcmSender, httpSender)

	_, err := dispatcher.Send(context.Background(), phone, &messaging.Message{}, uuid.New())

	require.Error(t, err)
	assert.Zero(t, fcmSender.calls)
	assert.Zero(t, httpSender.calls)
}

func TestPhoneNotificationDispatcherRejectsNilMessage(t *testing.T) {
	token := "fcm-token:value"
	phone := &entities.Phone{ID: uuid.New(), FcmToken: &token}
	fcmSender := &recordingNotificationSender{}
	httpSender := &recordingNotificationSender{}
	dispatcher := NewPhoneNotificationDispatcher(fcmSender, httpSender)

	_, err := dispatcher.Send(context.Background(), phone, nil, uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification message is nil")
	assert.Zero(t, fcmSender.calls)
	assert.Zero(t, httpSender.calls)
}
