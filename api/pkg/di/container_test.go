package di

import (
	"reflect"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationHTTPClientUsesOTelRoundTripperWithoutRetries(t *testing.T) {
	t.Setenv("ENV", "local")
	client := NewLiteContainer().NotificationHTTPClient()

	assert.Zero(t, client.Timeout)
	assert.Equal(t, "*otelroundtripper.otelRoundTripper", reflect.TypeOf(client.Transport).String())
	assert.Nil(t, client.CheckRedirect)
}

func TestPhoneNotificationClientsMapsConfiguredTransports(t *testing.T) {
	t.Setenv("ENV", "local")
	t.Setenv("FCM_ENDPOINT", "http://localhost")

	clients := NewLiteContainer().PhoneNotificationClients()

	require.Len(t, clients, 2)
	assert.IsType(t, &services.EmulatorFCMClient{}, clients[entities.NotificationTransportFCM])
	httpSender, ok := clients[entities.NotificationTransportHTTP].(*services.HTTPNotificationSender)
	require.True(t, ok)
	client := reflect.ValueOf(httpSender).Elem().FieldByName("client").Elem()
	transport := client.FieldByName("Transport").Elem()

	assert.Equal(t, "*otelroundtripper.otelRoundTripper", transport.Type().String())
}
