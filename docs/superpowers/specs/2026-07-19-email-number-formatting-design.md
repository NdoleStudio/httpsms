# Email Number Formatting Design

## Goal

Format integer quantities in generated emails with comma thousands separators. For example, `20000` becomes `20,000`.

## Scope

Apply formatting to message counts and subscription limits in usage-limit emails:

- sent messages
- received messages
- total messages
- plan message limits

Do not change dates, percentages, HTTP status codes, phone numbers, years, or numeric event payload values.

## Design

Add a private integer-formatting helper to the shared email `factory`. The helper will produce deterministic comma-grouped decimal output without adding a dependency.

Use the helper in `hermes_user_email_factory.go` anywhere a message count or subscription limit is rendered. Because Hermes generates both HTML and plain text from the same content model, both email formats will receive identical formatting.

## Error Handling

Formatting is a pure operation and introduces no new error path. Existing email generation errors continue to propagate unchanged.

## Testing

Add focused tests for integer grouping, including values below and above one thousand. Update usage-limit email tests to assert formatted quantities in both generated plain text and HTML.
