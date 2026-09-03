package telemetry

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactJSONFieldsRemovesSensitiveValues(t *testing.T) {
	body := []byte(`{"phone_number":"+18005550199","fcm_token":"https://adapter.example.com/secret?token=customer-secret","nested":{"fcm_token":"nested-secret"}}`)

	redacted := RedactJSONFields(body, "fcm_token")

	assert.Contains(t, redacted, `"phone_number":"+18005550199"`)
	assert.Equal(t, 2, strings.Count(redacted, "[redacted]"))
	assert.NotContains(t, redacted, "adapter.example.com")
	assert.NotContains(t, redacted, "customer-secret")
	assert.NotContains(t, redacted, "nested-secret")
}

func TestRedactJSONFieldsFailsClosedForMalformedSensitiveJSON(t *testing.T) {
	body := []byte(`{"fcm_token":"customer-secret"`)

	redacted := RedactJSONFields(body, "fcm_token")

	assert.Equal(t, "[request body omitted]", redacted)
	assert.NotContains(t, redacted, "customer-secret")
}

func TestRedactJSONFieldsFailsClosedForMalformedNonSensitiveBody(t *testing.T) {
	body := []byte(`not-json`)

	assert.Equal(t, "[request body omitted]", RedactJSONFields(body, "fcm_token"))
}
