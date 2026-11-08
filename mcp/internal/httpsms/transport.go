package httpsms

import (
	"context"
	"net/http"
)

// rawQueryContextKey is the context key queryRedactingTransport uses to
// smuggle a request's real, unmodified RawQuery past otelhttp.Transport to
// queryRestoringTransport. It is unexported and unique to this package, so
// it can never collide with a context value set by a caller or by another
// package.
type rawQueryContextKey struct{}

// queryRedactingTransport wraps an otelhttp-instrumented transport so that
// query string values (for example the free-text "query" search filter,
// which can contain SMS content, phone numbers, or other sensitive filter
// values) are never recorded as OpenTelemetry span attributes, while the
// real, unmodified query string is still sent to the httpSMS API on the
// wire and trace-context propagation headers are still injected as usual.
//
// otelhttp.Transport.RoundTrip derives every request span attribute
// (including the full request URL, via semconv.URLFull) from the exact
// *http.Request instance it is handed, and then forwards that same
// instance (after Clone-ing it to attach the span's context) one layer
// further down to its own configured base transport. There is therefore no
// exported option to give otelhttp one URL for its attributes and a
// different one for the real network call: the only seam available is
// between "what otelhttp is handed" and "what otelhttp's own base
// transport sends", which is exactly what this pair of transports uses.
//
//   - queryRedactingTransport (this type) sits in front of otelhttp.
//     It clones the incoming request, strips RawQuery from the clone's
//     URL, stashes the real RawQuery on the clone's context, and hands
//     that sanitized clone to otelhttp. otelhttp's span attributes are
//     therefore built from a query-free URL.
//   - queryRestoringTransport sits behind otelhttp, installed as the base
//     transport passed to otelhttp.NewTransport. It reads the real
//     RawQuery back out of the request's context and restores it onto the
//     request's URL immediately before delegating to the real network
//     transport (*http.Transport), so the httpSMS API still receives the
//     original, unmodified query string.
//
// Only one otelhttp.Transport is ever involved, so exactly one span is
// created per call: queryRedactingTransport itself does not start a span.
// Neither transport mutates the *http.Request a caller passed to
// http.Client.Do: queryRedactingTransport clones before making any change,
// and queryRestoringTransport only ever sees clones (first otelhttp's own
// Clone of queryRedactingTransport's clone).
type queryRedactingTransport struct {
	next http.RoundTripper // otelhttp.NewTransport(&queryRestoringTransport{...})
}

// RoundTrip implements http.RoundTripper.
func (t *queryRedactingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL == nil || req.URL.RawQuery == "" {
		// Nothing to redact: forward unchanged so GET requests without a
		// query string (and all POST/DELETE calls) skip the clone.
		return t.next.RoundTrip(req)
	}

	ctx := context.WithValue(req.Context(), rawQueryContextKey{}, req.URL.RawQuery)
	sanitized := req.Clone(ctx)

	sanitizedURL := *req.URL
	sanitizedURL.RawQuery = ""
	sanitized.URL = &sanitizedURL

	return t.next.RoundTrip(sanitized)
}

// queryRestoringTransport restores the real query string (stashed by
// queryRedactingTransport) onto the request's URL immediately before
// handing it to the real network transport, so the httpSMS API still
// receives the original, unmodified query even though otelhttp only ever
// saw a query-free URL.
type queryRestoringTransport struct {
	base http.RoundTripper // the real network transport (*http.Transport)
}

// RoundTrip implements http.RoundTripper.
func (t *queryRestoringTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if rawQuery, ok := req.Context().Value(rawQueryContextKey{}).(string); ok {
		restoredURL := *req.URL
		restoredURL.RawQuery = rawQuery
		req.URL = &restoredURL
	}
	return t.base.RoundTrip(req)
}
