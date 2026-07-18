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

func TestFormatEventPayloadHandlesTopLevelPayloadShapes(t *testing.T) {
	tests := []struct {
		name                string
		payload             string
		wantPlain           string
		wantHTMLContains    []string
		wantHTMLNotContains []string
	}{
		{
			name:                "empty payload falls back to unhighlighted block",
			payload:             "",
			wantPlain:           "",
			wantHTMLContains:    []string{`<pre style="`, `</pre>`},
			wantHTMLNotContains: []string{`<span style="color:`},
		},
		{
			name:                "top-level number stays valid JSON",
			payload:             "42",
			wantPlain:           "42",
			wantHTMLContains:    []string{`<span style="color:#953800;">42</span>`},
			wantHTMLNotContains: []string{`color:#0550AE`},
		},
		{
			name:      "json array stays readable and escaped",
			payload:   `[{"message":"<b>safe</b>"},true,null,3]`,
			wantPlain: "[\n  {\n    \"message\": \"<b>safe</b>\"\n  },\n  true,\n  null,\n  3\n]",
			wantHTMLContains: []string{
				"[\n  {",
				`&#34;message&#34;`,
				`&lt;b&gt;safe&lt;/b&gt;`,
				`<span style="color:#953800;">3</span>`,
			},
			wantHTMLNotContains: []string{`<b>safe</b>`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plain, rich := formatEventPayload(tt.payload)
			html := string(rich)

			assert.Equal(t, tt.wantPlain, plain)
			assert.Equal(t, 1, strings.Count(html, `<pre style=`))

			for _, want := range tt.wantHTMLContains {
				assert.Contains(t, html, want)
			}

			for _, unwanted := range tt.wantHTMLNotContains {
				assert.NotContains(t, html, unwanted)
			}
		})
	}
}
