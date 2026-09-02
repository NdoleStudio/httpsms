package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stringPointer(value string) *string {
	return &value
}

func TestPhoneNotificationTransport(t *testing.T) {
	tests := []struct {
		name      string
		token     *string
		transport NotificationTransport
		hasError  bool
	}{
		{name: "firebase token", token: stringPointer("fcm-token:value"), transport: NotificationTransportFCM},
		{name: "public https url", token: stringPointer("https://adapter.example.com/notify"), transport: NotificationTransportHTTP},
		{name: "missing token", token: nil, hasError: true},
		{name: "empty token", token: stringPointer("  "), hasError: true},
		{name: "http url", token: stringPointer("http://adapter.example.com/notify"), hasError: true},
		{name: "ftp url", token: stringPointer("ftp://adapter.example.com/notify"), hasError: true},
		{name: "missing host", token: stringPointer("https:///notify"), hasError: true},
		{name: "embedded credentials", token: stringPointer("******adapter.example.com/notify"), hasError: true},
		{name: "malformed url", token: stringPointer("https://[::1"), hasError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			phone := &Phone{FcmToken: test.token}

			transport, err := phone.NotificationTransport()

			if test.hasError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.transport, transport)
		})
	}
}

func TestPhoneNotificationURL(t *testing.T) {
	phone := &Phone{FcmToken: stringPointer("https://adapter.example.com/notify?tenant=42")}

	endpoint, err := phone.NotificationURL()

	require.NoError(t, err)
	assert.Equal(t, "https", endpoint.Scheme)
	assert.Equal(t, "adapter.example.com", endpoint.Hostname())
	assert.Equal(t, "/notify", endpoint.Path)
	assert.Equal(t, "tenant=42", endpoint.RawQuery)
}

func TestPhoneNotificationURLRejectsFCMToken(t *testing.T) {
	phone := &Phone{FcmToken: stringPointer("fcm-token:value")}

	_, err := phone.NotificationURL()

	require.Error(t, err)
}
