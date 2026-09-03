package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactedPhoneRequestBodyDoesNotExposeFCMToken(t *testing.T) {
	body := []byte(`{"phone_number":"+18005550199","fcm_token":"https://adapter.example.com/secret?token=customer-secret"}`)

	redacted := redactedPhoneRequestBody(body)

	assert.Contains(t, redacted, "[redacted]")
	assert.NotContains(t, redacted, "adapter.example.com")
	assert.NotContains(t, redacted, "customer-secret")
}
