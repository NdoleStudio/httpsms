package di

import (
	"crypto/tls"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationDispatcherRetainsTelemetryWrappedSecureHTTPTransport(t *testing.T) {
	t.Setenv("ENV", "local")
	t.Setenv("FCM_ENDPOINT", "http://localhost")

	dispatcher := NewLiteContainer().NotificationDispatcher()
	httpSender := reflect.ValueOf(dispatcher).Elem().FieldByName("httpSender").Elem().Elem()
	client := httpSender.FieldByName("client").Elem()
	roundTripper := client.FieldByName("Transport").Elem()

	require.Equal(t, "*otelroundtripper.otelRoundTripper", roundTripper.Type().String())

	parent := roundTripper.Elem().FieldByName("parent").Elem()
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
}
