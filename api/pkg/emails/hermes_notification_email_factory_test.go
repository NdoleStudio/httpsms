package emails

import (
	"strings"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNotificationEmailFactory() NotificationEmailFactory {
	return NewHermesNotificationEmailFactory(&HermesGeneratorConfig{
		AppURL:     "https://httpsms.com",
		AppName:    "httpSMS",
		AppLogoURL: "https://httpsms.com/logo.png",
	})
}

func TestWebhookSendFailedFormatsOnlyEventPayload(t *testing.T) {
	statusCode := 500
	factory := testNotificationEmailFactory()
	user := &entities.User{
		Email:    "name@email.com",
		Timezone: "UTC",
	}
	payload := &events.WebhookSendFailedPayload{
		WebhookID:              uuid.New(),
		WebhookURL:             "https://example.com/webhooks",
		Owner:                  "+237612345678",
		EventID:                "event-id",
		EventType:              "message.phone.received",
		EventPayload:           `{"message":"hello","retry":false}`,
		HTTPResponseStatusCode: &statusCode,
		ErrorMessage:           "plain failure response",
	}

	email, err := factory.WebhookSendFailed(user, payload)
	require.NoError(t, err)

	assert.Equal(t, "name@email.com", email.ToEmail)
	assert.Equal(t, "📢 We could not forward a webhook event to your server", email.Subject)
	assert.Contains(t, email.HTML, `<pre style=`)
	assert.Equal(t, 1, strings.Count(email.HTML, `<pre style=`))
	assert.Contains(t, email.HTML, `&#34;message&#34;`)
	assert.Contains(t, email.HTML, `plain failure response`)
	assert.Contains(t, email.Text, `"message": "hello"`)
	assert.Contains(t, email.Text, `"retry": false`)
	assert.NotContains(t, email.Text, "<pre")
}

func TestWebhookSendFailedPreservesNonJSONEventPayload(t *testing.T) {
	factory := testNotificationEmailFactory()
	user := &entities.User{
		Email:    "name@email.com",
		Timezone: "UTC",
	}
	payload := &events.WebhookSendFailedPayload{
		WebhookID:    uuid.New(),
		WebhookURL:   "https://example.com/webhooks",
		Owner:        "+237612345678",
		EventID:      "event-id",
		EventType:    "message.phone.received",
		EventPayload: "line one\n  line two",
		ErrorMessage: "plain failure response",
	}

	email, err := factory.WebhookSendFailed(user, payload)
	require.NoError(t, err)

	assert.Contains(t, email.HTML, "line one\n  line two")
	assert.NotContains(t, email.HTML, `<span style="color:`)
	assert.Contains(t, email.Text, "line one\n  line two")
}
