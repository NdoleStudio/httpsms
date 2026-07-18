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
