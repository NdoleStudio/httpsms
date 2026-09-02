package services

import (
	"context"
	"errors"
	"strings"
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

type phoneNotificationEventDispatcher struct {
	events []cloudevents.Event
}

func (dispatcher *phoneNotificationEventDispatcher) Dispatch(_ context.Context, event cloudevents.Event) error {
	dispatcher.events = append(dispatcher.events, event)
	return nil
}

func (dispatcher *phoneNotificationEventDispatcher) DispatchWithTimeout(
	_ context.Context,
	event cloudevents.Event,
	_ time.Duration,
) (string, error) {
	dispatcher.events = append(dispatcher.events, event)
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

func TestPhoneNotificationServiceSendUsesHTTPSGatewayNotification(t *testing.T) {
	endpoint := "https://adapter.example.com/notify"
	phone := &entities.Phone{
		ID:                       uuid.New(),
		UserID:                   "user-1",
		FcmToken:                 &endpoint,
		PhoneNumber:              "+18005550199",
		MessageExpirationSeconds: 90,
	}
	httpSender := &recordingNotificationSender{result: "http/notification-1"}
	eventDispatcher := &phoneNotificationEventDispatcher{}
	notificationRepository := &phoneNotificationRepository{}
	service := newPhoneNotificationServiceForTest(phone, notificationRepository, eventDispatcher, &recordingNotificationSender{}, httpSender)
	params := &PhoneNotificationSendParams{
		UserID:              phone.UserID,
		PhoneID:             phone.ID,
		PhoneNotificationID: uuid.New(),
		Source:              "test",
		ScheduledAt:         time.Now().UTC(),
		MessageID:           uuid.New(),
	}

	require.NoError(t, service.Send(context.Background(), params))

	assert.Equal(t, params.MessageID.String(), httpSender.notification.Data["KEY_MESSAGE_ID"])
	assert.Equal(t, "normal", httpSender.notification.Priority)
	require.NotNil(t, httpSender.notification.TTL)
	assert.Equal(t, phone.MessageExpirationDuration(), *httpSender.notification.TTL)
	assert.Equal(t, params.PhoneNotificationID, httpSender.notification.NotificationID)
	require.Len(t, eventDispatcher.events, 1)
	assert.Equal(t, events.EventTypeMessageNotificationSent, eventDispatcher.events[0].Type())
	assert.Equal(t, params.PhoneNotificationID, notificationRepository.notificationID)
	assert.Equal(t, entities.PhoneNotificationStatus(entities.PhoneNotificationStatusSent), notificationRepository.status)
}

func TestPhoneNotificationServiceSendHTTPFailureUsesAdapterGuidance(t *testing.T) {
	endpoint := "https://adapter.example.com/notify"
	phone := &entities.Phone{ID: uuid.New(), UserID: "user-1", FcmToken: &endpoint, PhoneNumber: "+18005550199"}
	eventDispatcher := &phoneNotificationEventDispatcher{}
	notificationRepository := &phoneNotificationRepository{}
	service := newPhoneNotificationServiceForTest(
		phone,
		notificationRepository,
		eventDispatcher,
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

	require.Len(t, eventDispatcher.events, 1)
	assert.Equal(t, events.EventTypeMessageNotificationFailed, eventDispatcher.events[0].Type())
	var payload events.MessageNotificationFailedPayload
	require.NoError(t, eventDispatcher.events[0].DataAs(&payload))
	assert.Equal(t, "cannot notify the configured adapter for phone [+18005550199]. Check the adapter URL and availability.", payload.ErrorMessage)
	assert.NotContains(t, payload.ErrorMessage, "Reinstall the httpSMS app")
	assert.Equal(t, entities.PhoneNotificationStatus(entities.PhoneNotificationStatusFailed), notificationRepository.status)
}

func TestPhoneNotificationServiceSendFCMFailurePreservesAndroidGuidance(t *testing.T) {
	token := "fcm-token"
	phone := &entities.Phone{ID: uuid.New(), UserID: "user-1", FcmToken: &token, PhoneNumber: "+18005550199"}
	eventDispatcher := &phoneNotificationEventDispatcher{}
	service := newPhoneNotificationServiceForTest(
		phone,
		&phoneNotificationRepository{},
		eventDispatcher,
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

	require.Len(t, eventDispatcher.events, 1)
	var payload events.MessageNotificationFailedPayload
	require.NoError(t, eventDispatcher.events[0].DataAs(&payload))
	assert.Equal(t, "cannot send notification to your phone [+18005550199]. Reinstall the httpSMS app on your Android phone.", payload.ErrorMessage)
}

func TestPhoneNotificationServiceSendDoesNotLogNotificationToken(t *testing.T) {
	endpoint := "https://adapter.example.com/private-token"
	phone := &entities.Phone{ID: uuid.New(), UserID: "user-1", FcmToken: &endpoint, PhoneNumber: "+18005550199"}
	logger := &phoneNotificationLogger{}
	service := NewNotificationService(
		logger,
		telemetry.NewOtelLogger("test", logger),
		NewNotificationDispatcher(
			&recordingNotificationSender{},
			&recordingNotificationSender{err: errors.New("POST " + endpoint + " failed")},
		),
		&phoneNotificationPhoneRepository{phone: phone},
		&phoneNotificationRepository{},
		nil,
		&phoneNotificationEventDispatcher{},
	)

	require.NoError(t, service.Send(context.Background(), &PhoneNotificationSendParams{
		UserID:              phone.UserID,
		PhoneID:             phone.ID,
		PhoneNotificationID: uuid.New(),
		Source:              "test",
		MessageID:           uuid.New(),
	}))

	for _, warning := range logger.warnings {
		assert.False(t, strings.Contains(warning, endpoint), "warning exposes notification token: %s", warning)
	}
}

func TestPhoneNotificationServiceSendHeartbeatFCMUsesHTTPSGatewayNotification(t *testing.T) {
	endpoint := "https://adapter.example.com/notify"
	phone := &entities.Phone{ID: uuid.New(), UserID: "user-1", FcmToken: &endpoint}
	httpSender := &recordingNotificationSender{err: errors.New("adapter unavailable")}
	service := newPhoneNotificationServiceForTest(
		phone,
		&phoneNotificationRepository{},
		&phoneNotificationEventDispatcher{},
		&recordingNotificationSender{},
		httpSender,
	)

	err := service.SendHeartbeatFCM(context.Background(), &events.PhoneHeartbeatMissedPayload{
		UserID:    phone.UserID,
		PhoneID:   phone.ID,
		MonitorID: uuid.New(),
	})

	require.NoError(t, err)
	heartbeatID := httpSender.notification.Data["KEY_HEARTBEAT_ID"]
	_, err = time.Parse(time.RFC3339, heartbeatID)
	require.NoError(t, err)
	assert.Equal(t, "high", httpSender.notification.Priority)
	assert.Nil(t, httpSender.notification.TTL)
	assert.NotEqual(t, uuid.Nil, httpSender.notification.NotificationID)
}

func newPhoneNotificationServiceForTest(
	phone *entities.Phone,
	notificationRepository repositories.PhoneNotificationRepository,
	eventDispatcher NotificationEventDispatcher,
	fcmSender NotificationSender,
	httpSender NotificationSender,
) *PhoneNotificationService {
	logger := &phoneNotificationLogger{}
	return NewNotificationService(
		logger,
		telemetry.NewOtelLogger("test", logger),
		NewNotificationDispatcher(fcmSender, httpSender),
		&phoneNotificationPhoneRepository{phone: phone},
		notificationRepository,
		nil,
		eventDispatcher,
	)
}
