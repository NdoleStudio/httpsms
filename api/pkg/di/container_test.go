package di

import (
	"crypto/tls"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationDispatcherUsesSecuredTransportAndSafeTelemetry(t *testing.T) {
	t.Setenv("ENV", "local")
	t.Setenv("FCM_ENDPOINT", "http://localhost")

	dispatcher := NewLiteContainer().NotificationDispatcher()
	httpSender := reflect.ValueOf(dispatcher).Elem().FieldByName("httpSender").Elem().Elem()
	client := httpSender.FieldByName("client").Elem()
	trustedTransport := client.FieldByName("Transport").Elem()

	require.Equal(t, "*services.notificationHTTPTransport", trustedTransport.Type().String())

	parent := trustedTransport.Elem().FieldByName("secured")
	require.Equal(t, "*http.Transport", parent.Type().String())

	transport := parent.Elem()
	assert.True(t, transport.FieldByName("Proxy").IsNil())
	assert.False(t, transport.FieldByName("DialContext").IsNil())
	assert.True(t, transport.FieldByName("DialTLS").IsNil())
	assert.True(t, transport.FieldByName("DialTLSContext").IsNil())

	tlsConfig := transport.FieldByName("TLSClientConfig").Elem()
	assert.False(t, tlsConfig.FieldByName("InsecureSkipVerify").Bool())
	assert.Empty(t, tlsConfig.FieldByName("ServerName").String())
	assert.Equal(t, uint64(tls.VersionTLS12), tlsConfig.FieldByName("MinVersion").Uint())

	attemptRecorder := httpSender.FieldByName("attemptRecorder").Elem()
	assert.Equal(t, "*services.otelNotificationHTTPAttemptRecorder", attemptRecorder.Type().String())
}
