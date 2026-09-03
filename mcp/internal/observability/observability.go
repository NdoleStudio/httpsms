// Package observability bootstraps the httpSMS MCP service's structured
// logging and distributed tracing, following the same conventions as the
// httpSMS API: JSON logs enriched with service/version fields, W3C trace
// context propagation, and an OpenTelemetry tracer provider that exports to
// whichever backend is configured through the environment (or exports
// nowhere, in local development, when none is configured).
package observability

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Environment variables that select an OTLP exporter destination for traces.
// These are the standard OpenTelemetry SDK variable names; otlptracehttp
// reads OTEL_EXPORTER_OTLP_ENDPOINT/OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
// itself, but New checks for their presence up front so it can fall back to
// a no-exporter local mode instead of constructing an exporter that would
// otherwise silently point nowhere.
const (
	otlpEndpointEnv       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	otlpTracesEndpointEnv = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
)

// New configures JSON logging and OpenTelemetry tracing for serviceName at
// version. It registers a global W3C (tracecontext + baggage) propagator and
// a global TracerProvider, then returns a logger, a shutdown function that
// flushes and stops the tracer provider, and any setup error.
//
// When neither OTEL_EXPORTER_OTLP_ENDPOINT nor OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
// is set, New registers a TracerProvider with no span processor: spans are
// still created (so context propagation and span-derived log fields keep
// working) but nothing is exported over the network. This is the local
// development / test mode.
func New(ctx context.Context, serviceName string, version string) (zerolog.Logger, func(context.Context) error, error) {
	logger := newLogger(serviceName, version)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		return logger, noopShutdown, fmt.Errorf("observability: cannot build resource: %w", err)
	}

	options := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}

	if hasOTLPExporterConfig() {
		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return logger, noopShutdown, fmt.Errorf("observability: cannot create OTLP trace exporter: %w", err)
		}
		options = append(options, sdktrace.WithBatcher(exporter))
	}

	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)

	return logger, provider.Shutdown, nil
}

// hasOTLPExporterConfig reports whether an OTLP trace exporter destination is
// configured through the environment. It never inspects endpoint values (no
// secrets are logged), only whether they are present.
func hasOTLPExporterConfig() bool {
	return os.Getenv(otlpEndpointEnv) != "" || os.Getenv(otlpTracesEndpointEnv) != ""
}

// newLogger builds a JSON zerolog.Logger writing to stdout, enriched with
// timestamp, service, and version fields.
func newLogger(serviceName string, version string) zerolog.Logger {
	return zerolog.New(os.Stdout).With().
		Timestamp().
		Str("service", serviceName).
		Str("version", version).
		Logger()
}

// noopShutdown is returned alongside a non-nil error from New, so callers can
// always defer the returned shutdown function unconditionally.
func noopShutdown(context.Context) error {
	return nil
}
