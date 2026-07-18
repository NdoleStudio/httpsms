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

	// #nosec G203 -- every dynamic payload token is escaped before the static wrapper is added.
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
			// json.Indent already validated this as JSON, so this continuation set only sees JSON number bytes.
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
