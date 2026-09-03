package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

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

func TestPhoneNotificationServiceSendUsesHTTPSMessage(t *testing.T) {
	endpoint := "https://adapter.example.com/notify"
	phone := &entities.Phone{
		ID:                       uuid.New(),
		UserID:                   "user-1",
		FcmToken:                 &endpoint,
		PhoneNumber:              "+18005550199",
		MessageExpirationSeconds: 90,
	}
	httpSender := &recordingNotificationSender{result: "http/notification-1"}
	eventQueue := &phoneNotificationEventQueue{}
	notificationRepository := &phoneNotificationRepository{}
	service := newPhoneNotificationServiceForTest(phone, notificationRepository, eventQueue, &recordingNotificationSender{}, httpSender)
	params := &PhoneNotificationSendParams{
		UserID:              phone.UserID,
		PhoneID:             phone.ID,
		PhoneNotificationID: uuid.New(),
		Source:              "test",
		ScheduledAt:         time.Now().UTC(),
		MessageID:           uuid.New(),
	}

	require.NoError(t, service.Send(context.Background(), params))

	require.NotNil(t, httpSender.message)
	assert.Equal(t, endpoint, httpSender.message.Token)
	assert.Equal(t, params.MessageID.String(), httpSender.message.Data["KEY_MESSAGE_ID"])
	require.NotNil(t, httpSender.message.Android)
	assert.Equal(t, "normal", httpSender.message.Android.Priority)
	require.NotNil(t, httpSender.message.Android.TTL)
	assert.Equal(t, phone.MessageExpirationDuration(), *httpSender.message.Android.TTL)
	assert.Equal(t, params.PhoneNotificationID, httpSender.notificationID)
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
		&recordingNotificationSender{},
		&recordingNotificationSender{err: errors.New("adapter unavailable")},
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
	service := newPhoneNotificationServiceForTest(
		phone,
		&phoneNotificationRepository{},
		eventQueue,
		&recordingNotificationSender{err: errors.New("firebase unavailable")},
		&recordingNotificationSender{},
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
	var payload events.MessageNotificationFailedPayload
	require.NoError(t, eventQueue.events[0].DataAs(&payload))
	assert.Equal(t, "cannot send notification to your phone [+18005550199]. Reinstall the httpSMS app on your Android phone.", payload.ErrorMessage)
}

func TestPhoneNotificationServiceSendHeartbeatFCMUsesHTTPSMessage(t *testing.T) {
	endpoint := "https://adapter.example.com/notify"
	phone := &entities.Phone{ID: uuid.New(), UserID: "user-1", FcmToken: &endpoint}
	httpSender := &recordingNotificationSender{err: errors.New("adapter unavailable")}
	service := newPhoneNotificationServiceForTest(
		phone,
		&phoneNotificationRepository{},
		&phoneNotificationEventQueue{},
		&recordingNotificationSender{},
		httpSender,
	)

	err := service.SendHeartbeatFCM(context.Background(), &events.PhoneHeartbeatMissedPayload{
		UserID:    phone.UserID,
		PhoneID:   phone.ID,
		MonitorID: uuid.New(),
	})

	require.NoError(t, err)
	require.NotNil(t, httpSender.message)
	assert.Equal(t, endpoint, httpSender.message.Token)
	heartbeatID := httpSender.message.Data["KEY_HEARTBEAT_ID"]
	_, err = time.Parse(time.RFC3339, heartbeatID)
	require.NoError(t, err)
	require.NotNil(t, httpSender.message.Android)
	assert.Equal(t, "high", httpSender.message.Android.Priority)
	assert.Nil(t, httpSender.message.Android.TTL)
	assert.NotEqual(t, uuid.Nil, httpSender.notificationID)
}

func newPhoneNotificationServiceForTest(
	phone *entities.Phone,
	notificationRepository repositories.PhoneNotificationRepository,
	eventQueue *phoneNotificationEventQueue,
	fcmSender NotificationSender,
	httpSender NotificationSender,
) *PhoneNotificationService {
	logger := &phoneNotificationLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewNotificationService(
		logger,
		tracer,
		NewPhoneNotificationDispatcher(fcmSender, httpSender),
		&phoneNotificationPhoneRepository{phone: phone},
		notificationRepository,
		nil,
		NewEventDispatcher(logger, tracer, nil, eventQueue, PushQueueConfig{}),
	)
}
