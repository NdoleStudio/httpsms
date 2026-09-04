package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

type phoneNotificationPhoneRepository struct {
	repositories.PhoneRepository
	phone *entities.Phone
	err   error
}

func (repository *phoneNotificationPhoneRepository) LoadByID(
	_ context.Context,
	_ entities.UserID,
	_ uuid.UUID,
) (*entities.Phone, error) {
	return repository.phone, repository.err
}

type phoneNotificationRepository struct {
	repositories.PhoneNotificationRepository
	notificationID uuid.UUID
	status         entities.PhoneNotificationStatus
}

func (repository *phoneNotificationRepository) UpdateStatus(
	_ context.Context,
	notificationID uuid.UUID,
	status entities.PhoneNotificationStatus,
) error {
	repository.notificationID = notificationID
	repository.status = status
	return nil
}

type phoneNotificationEventQueue struct {
	events []cloudevents.Event
}

func (queue *phoneNotificationEventQueue) Enqueue(
	_ context.Context,
	task *PushQueueTask,
	_ time.Duration,
) (string, error) {
	var event cloudevents.Event
	if err := json.Unmarshal(task.Body, &event); err != nil {
		return "", err
	}
	queue.events = append(queue.events, event)
	return "", nil
}

type phoneNotificationLogger struct {
	warnings []string
}

var _ telemetry.Logger = (*phoneNotificationLogger)(nil)

func (logger *phoneNotificationLogger) Error(error)                         {}
func (logger *phoneNotificationLogger) WithService(string) telemetry.Logger { return logger }
func (logger *phoneNotificationLogger) WithString(string, string) telemetry.Logger {
	return logger
}

func (logger *phoneNotificationLogger) WithSpan(trace.SpanContext) telemetry.Logger { return logger }
func (logger *phoneNotificationLogger) Trace(string)                                {}
func (logger *phoneNotificationLogger) Info(string)                                 {}
func (logger *phoneNotificationLogger) Warn(err error) {
	logger.warnings = append(logger.warnings, err.Error())
}
func (logger *phoneNotificationLogger) Debug(string)                  {}
func (logger *phoneNotificationLogger) Fatal(error)                   {}
func (logger *phoneNotificationLogger) Printf(string, ...interface{}) {}

type recordingPhoneNotificationClient struct {
	message *messaging.Message
	result  string
	err     error
	calls   int
}

func (client *recordingPhoneNotificationClient) Send(
	_ context.Context,
	message *messaging.Message,
) (string, error) {
	client.calls++
	client.message = message
	return client.result, client.err
}

func TestPhoneNotificationServiceSendPhoneNotificationUsesMappedClient(t *testing.T) {
	endpoint := "  https://adapter.example.com/notify  "
	phone := &entities.Phone{ID: uuid.New(), FcmToken: &endpoint}
	httpClient := &recordingPhoneNotificationClient{result: "accepted"}
	service := &PhoneNotificationService{
		phoneNotificationClients: map[entities.NotificationTransport]FCMClient{
			entities.NotificationTransportHTTP: httpClient,
		},
	}
	message := &messaging.Message{Data: map[string]string{"KEY_MESSAGE_ID": uuid.NewString()}}

	result, transport, err := service.sendPhoneNotification(context.Background(), phone, message)

	require.NoError(t, err)
	assert.Equal(t, "accepted", result)
	assert.Equal(t, entities.NotificationTransportHTTP, transport)
	assert.Equal(t, "https://adapter.example.com/notify", message.Token)
	assert.Same(t, message, httpClient.message)
	assert.Equal(t, 1, httpClient.calls)
}

func TestPhoneNotificationServiceSendPhoneNotificationRejectsMissingClient(t *testing.T) {
	endpoint := "https://adapter.example.com/notify"
	phone := &entities.Phone{ID: uuid.New(), FcmToken: &endpoint}
	service := &PhoneNotificationService{
		phoneNotificationClients: map[entities.NotificationTransport]FCMClient{},
	}

	_, transport, err := service.sendPhoneNotification(
		context.Background(),
		phone,
		&messaging.Message{},
	)

	require.Error(t, err)
	assert.Equal(t, entities.NotificationTransportHTTP, transport)
	assert.Contains(t, err.Error(), "notification client is not configured for transport [http]")
}

func TestPhoneNotificationServiceSendPhoneNotificationRejectsNilMessage(t *testing.T) {
	token := "fcm-token"
	phone := &entities.Phone{ID: uuid.New(), FcmToken: &token}
	fcmClient := &recordingPhoneNotificationClient{}
	service := &PhoneNotificationService{
		phoneNotificationClients: map[entities.NotificationTransport]FCMClient{
			entities.NotificationTransportFCM: fcmClient,
		},
	}

	_, transport, err := service.sendPhoneNotification(context.Background(), phone, nil)

	require.Error(t, err)
	assert.Empty(t, transport)
	assert.Contains(t, err.Error(), "notification message is nil")
	assert.Zero(t, fcmClient.calls)
}

func TestPhoneNotificationServiceSendUsesHTTPSMessage(t *testing.T) {
	endpoint := "https://adapter.example.com/notify"
	phone := &entities.Phone{
		ID:                       uuid.New(),
		UserID:                   "user-1",
		FcmToken:                 &endpoint,
		PhoneNumber:              "+18005550199",
		MessageExpirationSeconds: 90,
	}
	httpClient := &recordingPhoneNotificationClient{result: "http/success"}
	eventQueue := &phoneNotificationEventQueue{}
	notificationRepository := &phoneNotificationRepository{}
	service := newPhoneNotificationServiceForTest(
		phone,
		notificationRepository,
		eventQueue,
		map[entities.NotificationTransport]FCMClient{
			entities.NotificationTransportFCM:  &recordingPhoneNotificationClient{},
			entities.NotificationTransportHTTP: httpClient,
		},
	)
	params := &PhoneNotificationSendParams{
		UserID:              phone.UserID,
		PhoneID:             phone.ID,
		PhoneNotificationID: uuid.New(),
		Source:              "test",
		ScheduledAt:         time.Now().UTC(),
		MessageID:           uuid.New(),
	}

	require.NoError(t, service.Send(context.Background(), params))

	require.NotNil(t, httpClient.message)
	assert.Equal(t, endpoint, httpClient.message.Token)
	assert.Equal(t, params.MessageID.String(), httpClient.message.Data["KEY_MESSAGE_ID"])
	require.NotNil(t, httpClient.message.Android)
	assert.Equal(t, "normal", httpClient.message.Android.Priority)
	require.NotNil(t, httpClient.message.Android.TTL)
	assert.Equal(t, phone.MessageExpirationDuration(), *httpClient.message.Android.TTL)
	require.Len(t, eventQueue.events, 1)
	assert.Equal(t, events.EventTypeMessageNotificationSent, eventQueue.events[0].Type())
	assert.Equal(t, params.PhoneNotificationID, notificationRepository.notificationID)
	assert.Equal(t, entities.PhoneNotificationStatus(entities.PhoneNotificationStatusSent), notificationRepository.status)
}

func TestPhoneNotificationServiceSendHTTPFailureUsesAdapterGuidance(t *testing.T) {
	endpoint := "https://adapter.example.com/notify"
	phone := &entities.Phone{ID: uuid.New(), UserID: "user-1", FcmToken: &endpoint, PhoneNumber: "+18005550199"}
	eventQueue := &phoneNotificationEventQueue{}
	notificationRepository := &phoneNotificationRepository{}
	service := newPhoneNotificationServiceForTest(
		phone,
		notificationRepository,
		eventQueue,
		map[entities.NotificationTransport]FCMClient{
			entities.NotificationTransportFCM: &recordingPhoneNotificationClient{},
			entities.NotificationTransportHTTP: &recordingPhoneNotificationClient{
				err: errors.New("adapter unavailable"),
			},
		},
	)
	params := &PhoneNotificationSendParams{
		UserID:              phone.UserID,
		PhoneID:             phone.ID,
		PhoneNotificationID: uuid.New(),
		Source:              "test",
		MessageID:           uuid.New(),
	}

	require.NoError(t, service.Send(context.Background(), params))

	require.Len(t, eventQueue.events, 1)
	assert.Equal(t, events.EventTypeMessageNotificationFailed, eventQueue.events[0].Type())
	var payload events.MessageNotificationFailedPayload
	require.NoError(t, eventQueue.events[0].DataAs(&payload))
	assert.Equal(t, "cannot notify the configured adapter for phone [+18005550199]. Check the adapter URL and availability.", payload.ErrorMessage)
	assert.NotContains(t, payload.ErrorMessage, "Reinstall the httpSMS app")
	assert.Equal(t, entities.PhoneNotificationStatus(entities.PhoneNotificationStatusFailed), notificationRepository.status)
}

func TestPhoneNotificationServiceSendFCMFailurePreservesAndroidGuidance(t *testing.T) {
	token := "fcm-token"
	phone := &entities.Phone{ID: uuid.New(), UserID: "user-1", FcmToken: &token, PhoneNumber: "+18005550199"}
	eventQueue := &phoneNotificationEventQueue{}
	fcmClient := &recordingPhoneNotificationClient{err: errors.New("firebase unavailable")}
	service := newPhoneNotificationServiceForTest(
		phone,
		&phoneNotificationRepository{},
		eventQueue,
		map[entities.NotificationTransport]FCMClient{
			entities.NotificationTransportFCM:  fcmClient,
			entities.NotificationTransportHTTP: &recordingPhoneNotificationClient{},
		},
	)
	params := &PhoneNotificationSendParams{
		UserID:              phone.UserID,
		PhoneID:             phone.ID,
		PhoneNotificationID: uuid.New(),
		Source:              "test",
		MessageID:           uuid.New(),
	}

	require.NoError(t, service.Send(context.Background(), params))

	require.Len(t, eventQueue.events, 1)
	require.NotNil(t, fcmClient.message)
	assert.Equal(t, token, fcmClient.message.Token)
	assert.Equal(t, 1, fcmClient.calls)
	var payload events.MessageNotificationFailedPayload
	require.NoError(t, eventQueue.events[0].DataAs(&payload))
	assert.Equal(t, "cannot send notification to your phone [+18005550199]. Reinstall the httpSMS app on your Android phone.", payload.ErrorMessage)
}

func TestPhoneNotificationServiceSendHeartbeatFCMUsesHTTPSMessage(t *testing.T) {
	endpoint := "https://adapter.example.com/notify"
	phone := &entities.Phone{ID: uuid.New(), UserID: "user-1", FcmToken: &endpoint}
	httpClient := &recordingPhoneNotificationClient{err: errors.New("adapter unavailable")}
	service := newPhoneNotificationServiceForTest(
		phone,
		&phoneNotificationRepository{},
		&phoneNotificationEventQueue{},
		map[entities.NotificationTransport]FCMClient{
			entities.NotificationTransportFCM:  &recordingPhoneNotificationClient{},
			entities.NotificationTransportHTTP: httpClient,
		},
	)

	err := service.SendHeartbeatFCM(context.Background(), &events.PhoneHeartbeatMissedPayload{
		UserID:    phone.UserID,
		PhoneID:   phone.ID,
		MonitorID: uuid.New(),
	})

	require.NoError(t, err)
	require.NotNil(t, httpClient.message)
	assert.Equal(t, endpoint, httpClient.message.Token)
	heartbeatID := httpClient.message.Data["KEY_HEARTBEAT_ID"]
	_, err = time.Parse(time.RFC3339, heartbeatID)
	require.NoError(t, err)
	require.NotNil(t, httpClient.message.Android)
	assert.Equal(t, "high", httpClient.message.Android.Priority)
	assert.Nil(t, httpClient.message.Android.TTL)
}

func newPhoneNotificationServiceForTest(
	phone *entities.Phone,
	notificationRepository repositories.PhoneNotificationRepository,
	eventQueue *phoneNotificationEventQueue,
	clients map[entities.NotificationTransport]FCMClient,
) *PhoneNotificationService {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewNotificationService(
		logger,
		tracer,
		clients,
		&phoneNotificationPhoneRepository{phone: phone},
		notificationRepository,
		nil,
		NewEventDispatcher(logger, tracer, nil, eventQueue, PushQueueConfig{}),
	)
}
