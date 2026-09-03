package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotificationHTTPClientUsesOTelRoundTripperWithoutRetries(t *testing.T) {
	t.Setenv("ENV", "local")
	client := NewLiteContainer().NotificationHTTPClient()

	assert.Zero(t, client.Timeout)
	assert.Equal(t, "*otelroundtripper.otelRoundTripper", reflect.TypeOf(client.Transport).String())
	assert.Nil(t, client.CheckRedirect)
}

func TestPhoneNotificationDispatcherInjectsNotificationHTTPClient(t *testing.T) {
	t.Setenv("ENV", "local")
	t.Setenv("FCM_ENDPOINT", "http://localhost")

	dispatcher := NewLiteContainer().PhoneNotificationDispatcher()
	httpSender := reflect.ValueOf(dispatcher).Elem().FieldByName("httpSender").Elem().Elem()
	client := httpSender.FieldByName("client").Elem()
	transport := client.FieldByName("Transport").Elem()

	assert.Equal(t, "*otelroundtripper.otelRoundTripper", transport.Type().String())
	attemptRecorder := httpSender.FieldByName("attemptRecorder").Elem()
	assert.Equal(t, "*services.otelNotificationHTTPAttemptRecorder", attemptRecorder.Type().String())
}
