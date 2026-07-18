# Webhook Email Payload Formatting

- Date: 2026-07-18
- Status: Approved (design)
- Scope: `api/` email rendering only.

## Problem

Failed-webhook notification emails currently display the event payload as an
unformatted dictionary value. Nested JSON is difficult to scan, and multiline
payloads lose the visual structure users need when diagnosing webhook failures.

## Decisions

- Format only the webhook email's **Event Payload** value.
- Leave SMS content, failure reasons, HTTP responses, and every other email
  field unchanged.
- Pretty-print valid JSON with indentation.
- Render valid JSON in the HTML email with lightweight colors for keys, strings,
  numbers, booleans, and `null`.
- Render non-JSON payloads in the same monospace block while preserving their
  original whitespace, without syntax colors.
- Use only the Go standard library and the existing Hermes email integration.
- Keep the plain-text email readable with indentation but no HTML styling.

## Design

### 1. Payload formatter

Add a focused formatter under `api/pkg/emails` that accepts the event payload
string and returns:

- a plain-text value for `hermes.Entry.Value`; and
- safely escaped highlighted markup for `hermes.Entry.UnsafeValue`.

For valid JSON, use the Go standard library to produce deterministic
indentation. A small JSON-aware tokenizer will wrap keys, strings, numbers,
booleans, and `null` in spans with inline colors. Inline styles are preferred
because many email clients strip style blocks or external styles.

All payload tokens must be HTML-escaped before they are inserted into the
trusted template value. Payload content must never be interpreted as HTML.

For invalid JSON, return the original payload unchanged as the plain-text value
and HTML-escape it into an unhighlighted code block. Invalid JSON is an expected
fallback and must not prevent email generation or delivery.

### 2. Hermes dictionary rendering

Update the custom HTML template in `api/pkg/emails/hermes_theme.go` so dictionary
entries render `UnsafeValue` when it is present and otherwise retain the current
escaped `Value` rendering.

The payload block will use email-compatible markup and inline styles:

- a neutral light background and subtle border;
- padding and a monospace font;
- `white-space: pre-wrap` to preserve indentation and line breaks; and
- safe wrapping for long values so the email remains usable on narrow clients.

The plain-text template continues to render `Entry.Value`, so it contains
readable indented JSON without HTML tags or colors.

### 3. Webhook email factory

In `hermesNotificationEmailFactory.WebhookSendFailed`, pass
`payload.EventPayload` through the formatter. Set both `Value` and
`UnsafeValue` only on the existing `Event Payload` dictionary entry.

No other dictionary entry or notification email uses the new unsafe rendering
path. The order, subject, explanatory text, action button, and delivery behavior
of the webhook failure email remain unchanged.

## Data Flow

1. `WebhookSendFailed` receives `payload.EventPayload`.
2. The formatter checks whether the string is valid JSON.
3. Valid JSON is indented and highlighted; invalid JSON preserves its original
   text and receives code-block styling only.
4. Hermes uses the plain value for the text email and the escaped styled value
   for the HTML email.
5. The existing mailer sends both representations without service or listener
   changes.

## Error Handling and Security

- Invalid JSON falls back to a whitespace-preserving code block.
- Formatting does not introduce a new email-generation failure path.
- Payload markup, quotes, ampersands, and other special characters are escaped
  before insertion.
- Highlighting is generated server-side and requires no JavaScript, remote
  assets, or external CSS.
- Existing Hermes generation errors continue to use the current stacktrace
  propagation behavior.

## Testing

Add targeted tests under `api/pkg/emails` covering:

- nested valid JSON is consistently indented;
- keys, strings, numbers, booleans, and `null` receive the expected HTML spans;
- payload HTML is escaped and cannot inject markup;
- invalid JSON preserves whitespace and is not syntax-highlighted;
- the webhook email HTML renders only `Event Payload` as a code block;
- the webhook email plain text contains readable indented JSON; and
- existing non-payload dictionary entries keep their current rendering.

Run the targeted email package tests, then `go test ./...` in `api/`.

## Out of Scope

- Formatting SMS message content or failure reasons.
- Formatting HTTP response bodies or error messages.
- Highlighting non-JSON languages.
- Adding a third-party syntax-highlighting dependency.
- Changing webhook retries, event payload generation, or email delivery logic.
