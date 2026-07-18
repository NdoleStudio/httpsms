package emails

import (
	"html/template"
	"testing"

	"github.com/go-hermes/hermes/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHermesThemeRendersUnsafeDictionaryValueOnlyWhenProvided(t *testing.T) {
	generator := (&HermesGeneratorConfig{
		AppURL:     "https://httpsms.com",
		AppName:    "httpSMS",
		AppLogoURL: "https://httpsms.com/logo.png",
	}).Generator()

	email := hermes.Email{
		Body: hermes.Body{
			Title: "Hello",
			Dictionary: []hermes.Entry{
				{Key: "Normal", Value: "<b>escaped</b>"},
				{
					Key:         "Payload",
					Value:       "plain payload",
					UnsafeValue: template.HTML(`<pre style="white-space:pre-wrap">rich payload</pre>`),
				},
			},
		},
	}

	htmlEmail, err := generator.GenerateHTML(email)
	require.NoError(t, err)

	textEmail, err := generator.GeneratePlainText(email)
	require.NoError(t, err)

	assert.Contains(t, htmlEmail, `&lt;b&gt;escaped&lt;/b&gt;`)
	assert.NotContains(t, htmlEmail, `<b>escaped</b>`)
	assert.Contains(t, htmlEmail, `<pre style="white-space:pre-wrap">rich payload</pre>`)
	assert.NotContains(t, htmlEmail, `<dd>plain payload</dd>`)
	assert.Contains(t, textEmail, "Normal: <b>escaped</b>")
	assert.Contains(t, textEmail, "Payload: plain payload")
	assert.NotContains(t, textEmail, "<pre")
}
