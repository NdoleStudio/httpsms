# Webhook Email Payload Formatting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render the webhook failure email's Event Payload as readable indented JSON with safe lightweight syntax colors in HTML and readable indentation in plain text.

**Architecture:** Add a focused formatter in `api/pkg/emails` that produces a plain string and a safely constructed `template.HTML` code block. Teach the custom Hermes HTML dictionary template to use `hermes.Entry.UnsafeValue` when explicitly supplied, then wire only the webhook email's Event Payload entry to that path.

**Tech Stack:** Go 1.25.8 module, Go standard library (`bytes`, `encoding/json`, `html`, `html/template`, `strings`), go-hermes v2.6.2, testify v1.11.1.

## Global Constraints

- Format only the webhook email's **Event Payload** value.
- Leave SMS content, failure reasons, HTTP responses, and every other email field unchanged.
- Pretty-print valid JSON with indentation.
- Render valid JSON in HTML with lightweight colors for keys, strings, numbers, booleans, and `null`.
- Render invalid JSON in the same monospace block with original whitespace and no syntax colors.
- Add no third-party dependency.
- HTML-escape every payload token before converting the completed markup to `template.HTML`.
- Invalid JSON must not prevent email generation or delivery.
- Preserve the existing plain-text email behavior except for readable payload indentation.
- Use `stacktrace.Propagate` for existing Hermes generation errors; this feature introduces no new returned error.

---

## File Structure

- Create `api/pkg/emails/event_payload_formatter.go`: JSON indentation, token highlighting, HTML escaping, and code-block construction.
- Create `api/pkg/emails/event_payload_formatter_test.go`: focused tests for valid JSON tokens, escaping, and invalid JSON fallback.
- Create `api/pkg/emails/hermes_theme_test.go`: verifies safe opt-in HTML dictionary rendering and unchanged plain-text rendering.
- Modify `api/pkg/emails/hermes_theme.go:325-334`: render `Entry.UnsafeValue` only when present.
- Create `api/pkg/emails/hermes_notification_email_factory_test.go`: end-to-end factory coverage for the webhook notification email.
- Modify `api/pkg/emails/hermes_notification_email_factory.go:78-122`: format and attach only the Event Payload dictionary entry.

### Task 1: Build the event payload formatter

**Files:**
- Create: `api/pkg/emails/event_payload_formatter.go`
- Create: `api/pkg/emails/event_payload_formatter_test.go`

**Interfaces:**
- Consumes: a raw webhook event payload `string`.
- Produces: `formatEventPayload(payload string) (string, template.HTML)`.
- Produces: plain indented JSON for `hermes.Entry.Value` and an escaped styled code block for `hermes.Entry.UnsafeValue`.

- [ ] **Step 1: Write the failing formatter tests**

Create `api/pkg/emails/event_payload_formatter_test.go`:

```go
package emails

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatEventPayloadIndentsAndHighlightsJSON(t *testing.T) {
	payload := `{"message":"hello","count":2,"ratio":1.5,"enabled":true,"disabled":false,"missing":null,"nested":{"value":"ok"}}`

	plain, rich := formatEventPayload(payload)
	html := string(rich)

	assert.Equal(t, `{
  "message": "hello",
  "count": 2,
  "ratio": 1.5,
  "enabled": true,
  "disabled": false,
  "missing": null,
  "nested": {
    "value": "ok"
  }
}`, plain)
	assert.Contains(t, html, `<span style="color:#0550AE;font-weight:600;">&#34;message&#34;</span>`)
	assert.Contains(t, html, `<span style="color:#0A3069;">&#34;hello&#34;</span>`)
	assert.Contains(t, html, `<span style="color:#953800;">2</span>`)
	assert.Contains(t, html, `<span style="color:#953800;">1.5</span>`)
	assert.Contains(t, html, `<span style="color:#8250DF;">true</span>`)
	assert.Contains(t, html, `<span style="color:#8250DF;">false</span>`)
	assert.Contains(t, html, `<span style="color:#8250DF;">null</span>`)
	assert.Contains(t, html, `white-space:pre-wrap`)
}

func TestFormatEventPayloadEscapesPayloadHTML(t *testing.T) {
	plain, rich := formatEventPayload(`{"message":"<script>alert(\"x\")</script>&"}`)
	html := string(rich)

	assert.Contains(t, plain, `<script>alert`)
	assert.NotContains(t, html, `<script>`)
	assert.Contains(t, html, `&lt;script&gt;`)
	assert.Contains(t, html, `&amp;`)
}

func TestFormatEventPayloadPreservesInvalidJSONWithoutHighlighting(t *testing.T) {
	payload := "line one\n  <strong>line two</strong>"

	plain, rich := formatEventPayload(payload)
	html := string(rich)

	assert.Equal(t, payload, plain)
	assert.Contains(t, html, "line one\n  &lt;strong&gt;line two&lt;/strong&gt;")
	assert.NotContains(t, html, `<strong>`)
	assert.NotContains(t, html, `<span style="color:`)
	assert.Equal(t, 1, strings.Count(html, `<pre style=`))
}
```

- [ ] **Step 2: Run the formatter tests to verify they fail**

Run:

```powershell
Set-Location api
go test ./pkg/emails -run '^TestFormatEventPayload' -count=1
```

Expected: build failure with `undefined: formatEventPayload`.

- [ ] **Step 3: Implement the formatter**

Create `api/pkg/emails/event_payload_formatter.go`:

```go
package emails

import (
	"bytes"
	"encoding/json"
	"html"
	"html/template"
	"strings"
)

const (
	eventPayloadCodeBlockStyle = "margin:0;padding:12px;border:1px solid #D0D7DE;border-radius:6px;background:#F6F8FA;color:#24292F;font-family:Consolas,Monaco,'Courier New',monospace;font-size:13px;line-height:1.5;white-space:pre-wrap;word-break:break-word;overflow-wrap:anywhere;"
	jsonKeyStyle               = "color:#0550AE;font-weight:600;"
	jsonStringStyle            = "color:#0A3069;"
	jsonNumberStyle            = "color:#953800;"
	jsonLiteralStyle           = "color:#8250DF;"
)

func formatEventPayload(payload string) (string, template.HTML) {
	formattedPayload, isJSON := indentEventPayloadJSON(payload)
	content := html.EscapeString(formattedPayload)
	if isJSON {
		content = highlightEventPayloadJSON(formattedPayload)
	}

	// Every payload token is escaped before this trusted wrapper is constructed.
	richPayload := template.HTML(`<pre style="` + eventPayloadCodeBlockStyle + `">` + content + `</pre>`)
	return formattedPayload, richPayload
}

func indentEventPayloadJSON(payload string) (string, bool) {
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, []byte(payload), "", "  "); err != nil {
		return payload, false
	}

	return formatted.String(), true
}

func highlightEventPayloadJSON(payload string) string {
	var highlighted strings.Builder
	highlighted.Grow(len(payload))

	for index := 0; index < len(payload); {
		switch {
		case payload[index] == '"':
			end := eventPayloadJSONStringEnd(payload, index)
			style := jsonStringStyle
			if eventPayloadNextNonSpace(payload, end) == ':' {
				style = jsonKeyStyle
			}
			writeEventPayloadToken(&highlighted, style, payload[index:end])
			index = end
		case payload[index] == '-' || isEventPayloadDigit(payload[index]):
			end := index + 1
			for end < len(payload) && isEventPayloadNumberCharacter(payload[end]) {
				end++
			}
			writeEventPayloadToken(&highlighted, jsonNumberStyle, payload[index:end])
			index = end
		case strings.HasPrefix(payload[index:], "true"):
			writeEventPayloadToken(&highlighted, jsonLiteralStyle, "true")
			index += len("true")
		case strings.HasPrefix(payload[index:], "false"):
			writeEventPayloadToken(&highlighted, jsonLiteralStyle, "false")
			index += len("false")
		case strings.HasPrefix(payload[index:], "null"):
			writeEventPayloadToken(&highlighted, jsonLiteralStyle, "null")
			index += len("null")
		default:
			highlighted.WriteString(html.EscapeString(payload[index : index+1]))
			index++
		}
	}

	return highlighted.String()
}

func eventPayloadJSONStringEnd(payload string, start int) int {
	escaped := false
	for index := start + 1; index < len(payload); index++ {
		switch {
		case escaped:
			escaped = false
		case payload[index] == '\\':
			escaped = true
		case payload[index] == '"':
			return index + 1
		}
	}

	return len(payload)
}

func eventPayloadNextNonSpace(payload string, start int) byte {
	for index := start; index < len(payload); index++ {
		switch payload[index] {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return payload[index]
		}
	}

	return 0
}

func isEventPayloadDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isEventPayloadNumberCharacter(value byte) bool {
	return isEventPayloadDigit(value) ||
		value == '-' ||
		value == '+' ||
		value == '.' ||
		value == 'e' ||
		value == 'E'
}

func writeEventPayloadToken(builder *strings.Builder, style string, token string) {
	builder.WriteString(`<span style="`)
	builder.WriteString(style)
	builder.WriteString(`">`)
	builder.WriteString(html.EscapeString(token))
	builder.WriteString(`</span>`)
}
```

- [ ] **Step 4: Format and run the formatter tests**

Run:

```powershell
pre-commit run go-fumpt --files api/pkg/emails/event_payload_formatter.go api/pkg/emails/event_payload_formatter_test.go
Set-Location api
go test ./pkg/emails -run '^TestFormatEventPayload' -count=1
```

Expected: `go-fumpt` passes (rerun it once if it initially reformats the files),
then all three formatter tests pass.

- [ ] **Step 5: Commit the formatter**

Run from the repository root:

```powershell
git add -- api\pkg\emails\event_payload_formatter.go api\pkg\emails\event_payload_formatter_test.go
git commit -m "feat(api): format webhook email payload" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>" -m "Copilot-Session: 48f6d946-ae22-4440-b7a1-44e939419b11"
```

Expected: one commit containing only the formatter and its tests.

### Task 2: Add opt-in rich dictionary rendering to the Hermes theme

**Files:**
- Create: `api/pkg/emails/hermes_theme_test.go`
- Modify: `api/pkg/emails/hermes_theme.go:325-334`

**Interfaces:**
- Consumes: `hermes.Entry{Value: plain, UnsafeValue: rich}`.
- Produces: HTML rendering that uses `UnsafeValue` only when non-empty.
- Preserves: plain-text rendering always uses `Value`.

- [ ] **Step 1: Write the failing Hermes theme test**

Create `api/pkg/emails/hermes_theme_test.go`:

```go
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
```

- [ ] **Step 2: Run the theme test to verify it fails**

Run:

```powershell
Set-Location api
go test ./pkg/emails -run '^TestHermesThemeRendersUnsafeDictionaryValueOnlyWhenProvided$' -count=1
```

Expected: failure because the generated HTML contains `plain payload` instead
of the `<pre>` block.

- [ ] **Step 3: Update the HTML dictionary template**

In `api/pkg/emails/hermes_theme.go`, replace:

```gotemplate
<dd>{{ $entry.Value }}</dd>
```

with:

```gotemplate
<dd>{{ if $entry.UnsafeValue }}{{ $entry.UnsafeValue }}{{ else }}{{ $entry.Value }}{{ end }}</dd>
```

Do not change the plain-text template at `api/pkg/emails/hermes_theme.go:523-528`;
it must continue to render:

```gotemplate
<li>{{ $entry.Key }}: {{ $entry.Value }}</li>
```

- [ ] **Step 4: Format and run the theme and formatter tests**

Run:

```powershell
pre-commit run go-fumpt --files api/pkg/emails/hermes_theme.go api/pkg/emails/hermes_theme_test.go
Set-Location api
go test ./pkg/emails -run '^(TestHermesTheme|TestFormatEventPayload)' -count=1
```

Expected: `go-fumpt` passes (rerun it once if it initially reformats the files),
then the theme test and all formatter tests pass.

- [ ] **Step 5: Commit the theme support**

Run from the repository root:

```powershell
git add -- api\pkg\emails\hermes_theme.go api\pkg\emails\hermes_theme_test.go
git commit -m "feat(api): render rich email dictionary values" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>" -m "Copilot-Session: 48f6d946-ae22-4440-b7a1-44e939419b11"
```

Expected: one commit containing only the Hermes template opt-in and its test.

### Task 3: Wire formatting into the webhook failure email

**Files:**
- Create: `api/pkg/emails/hermes_notification_email_factory_test.go`
- Modify: `api/pkg/emails/hermes_notification_email_factory.go:78-122`

**Interfaces:**
- Consumes: `formatEventPayload(payload string) (string, template.HTML)` from Task 1.
- Produces: the existing `NotificationEmailFactory.WebhookSendFailed` email with only its Event Payload entry using `UnsafeValue`.
- Preserves: factory interface, subjects, recipients, other dictionary entries, actions, and error propagation.

- [ ] **Step 1: Write the failing webhook factory tests**

Create `api/pkg/emails/hermes_notification_email_factory_test.go`:

```go
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
```

- [ ] **Step 2: Run the webhook factory tests to verify they fail**

Run:

```powershell
Set-Location api
go test ./pkg/emails -run '^TestWebhookSendFailed' -count=1
```

Expected: failure because Event Payload still renders as an ordinary dictionary
value and no `<pre>` block is present.

- [ ] **Step 3: Format Event Payload in the webhook factory**

At the start of
`hermesNotificationEmailFactory.WebhookSendFailed` in
`api/pkg/emails/hermes_notification_email_factory.go`, add:

```go
formattedPayload, formattedPayloadHTML := formatEventPayload(payload.EventPayload)
```

Replace the existing Event Payload dictionary entry:

```go
{Key: "Event Payload", Value: payload.EventPayload},
```

with:

```go
{
	Key:         "Event Payload",
	Value:       formattedPayload,
	UnsafeValue: formattedPayloadHTML,
},
```

Do not change the Error Message / HTTP Response entry or any SMS email method.

- [ ] **Step 4: Format and run all email package tests**

Run:

```powershell
pre-commit run go-fumpt --files api/pkg/emails/hermes_notification_email_factory.go api/pkg/emails/hermes_notification_email_factory_test.go
Set-Location api
go test ./pkg/emails -count=1
```

Expected: `go-fumpt` passes (rerun it once if it initially reformats the files),
then all `pkg/emails` tests pass.

- [ ] **Step 5: Run the stable API package regression suite**

Run:

```powershell
Set-Location api
$env:GOTOOLCHAIN = 'go1.25.8'
$packages = go list ./... | Where-Object { $_ -ne 'github.com/NdoleStudio/httpsms/pkg/handlers' }
go test -vet=off $packages
```

Expected: all selected API packages pass without adding or changing
dependencies.

The clean baseline cannot use plain `go test ./...`: existing stacktrace calls
fail Go vet's non-constant-format check, and
`pkg/handlers.TestPhoneAPIKeyHandler_store` requires an API server at
`localhost:8000`. Do not change those unrelated packages as part of this
feature.

- [ ] **Step 6: Review the final diff**

Run from the repository root:

```powershell
git --no-pager diff --check
git --no-pager diff -- api\pkg\emails
```

Expected:

- no whitespace errors;
- only the formatter, Hermes dictionary opt-in, webhook Event Payload wiring,
  and their tests are present;
- SMS content, error responses, and failure reasons are unchanged; and
- `api\go.mod` and `api\go.sum` are unchanged.

- [ ] **Step 7: Commit the webhook factory integration**

Run from the repository root:

```powershell
git add -- api\pkg\emails\hermes_notification_email_factory.go api\pkg\emails\hermes_notification_email_factory_test.go
git commit -m "feat(api): highlight webhook email payload" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>" -m "Copilot-Session: 48f6d946-ae22-4440-b7a1-44e939419b11"
```

Expected: one commit containing only the webhook factory integration and its
tests.
