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

	axiomzerolog "github.com/axiomhq/axiom-go/adapters/zerolog"
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
	axiomTokenEnv         = "AXIOM_TOKEN"
	axiomEventsDatasetEnv = "AXIOM_DATASET_EVENTS"
	axiomOTLPEndpointEnv  = "AXIOM_OTLP_ENDPOINT"
	defaultAxiomEndpoint  = "us-east-1.aws.edge.axiom.co"
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
	logger, closeLogger, err := newLogger(serviceName, version)
	if err != nil {
		return zerolog.Nop(), noopShutdown, err
	}
	if hasPartialAxiomConfig() {
		logger.Warn().Msg("Axiom telemetry disabled because AXIOM_TOKEN and AXIOM_DATASET_EVENTS are not both configured")
	}

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
		closeLogger()
		return logger, noopShutdown, fmt.Errorf("observability: cannot build resource: %w", err)
	}

	options := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}

	traceOptions, exportTraces, err := traceExporterOptions()
	if err != nil {
		closeLogger()
		return logger, noopShutdown, err
	}
	if exportTraces {
		exporter, err := otlptracehttp.New(ctx, traceOptions...)
		if err != nil {
			closeLogger()
			return logger, noopShutdown, fmt.Errorf("observability: cannot create OTLP trace exporter: %w", err)
		}
		options = append(options, sdktrace.WithBatcher(exporter))
	}

	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)

	shutdown := func(shutdownCtx context.Context) error {
		err := provider.Shutdown(shutdownCtx)
		closeLogger()
		return err
	}

	return logger, shutdown, nil
}

// traceExporterOptions selects a standard OTLP destination when configured,
// otherwise the same Axiom OTLP endpoint and events dataset used by the main
// httpSMS API. The token is passed only as an exporter header and is never
// attached to a span or log event.
func traceExporterOptions() ([]otlptracehttp.Option, bool, error) {
	if os.Getenv(otlpEndpointEnv) != "" || os.Getenv(otlpTracesEndpointEnv) != "" {
		return nil, true, nil
	}

	token := os.Getenv(axiomTokenEnv)
	dataset := os.Getenv(axiomEventsDatasetEnv)
	if token == "" && dataset == "" {
		return nil, false, nil
	}
	if token == "" || dataset == "" {
		return nil, false, nil
	}

	endpoint := os.Getenv(axiomOTLPEndpointEnv)
	if endpoint == "" {
		endpoint = defaultAxiomEndpoint
	}

	return []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization":   "Bearer " + token,
			"X-Axiom-Dataset": dataset,
		}),
	}, true, nil
}

func hasPartialAxiomConfig() bool {
	tokenConfigured := os.Getenv(axiomTokenEnv) != ""
	datasetConfigured := os.Getenv(axiomEventsDatasetEnv) != ""
	return tokenConfigured != datasetConfigured
}

// newLogger builds a JSON logger enriched with service/version fields. It
// always writes to stdout for Cloud Run logging and, when Axiom is configured,
// also sends the same redacted structured events directly to Axiom.
func newLogger(serviceName string, version string) (zerolog.Logger, func(), error) {
	output := zerolog.LevelWriter(zerolog.MultiLevelWriter(os.Stdout))
	closeLogger := func() {}

	token := os.Getenv(axiomTokenEnv)
	dataset := os.Getenv(axiomEventsDatasetEnv)
	if token != "" && dataset != "" {
		writer, err := axiomzerolog.New(
			axiomzerolog.SetDataset(dataset),
			axiomzerolog.SetLevels([]zerolog.Level{
				zerolog.DebugLevel,
				zerolog.InfoLevel,
				zerolog.WarnLevel,
				zerolog.ErrorLevel,
				zerolog.FatalLevel,
				zerolog.PanicLevel,
			}),
		)
		if err != nil {
			return zerolog.Nop(), closeLogger, fmt.Errorf("observability: cannot create Axiom log writer: %w", err)
		}
		output = zerolog.MultiLevelWriter(os.Stdout, writer)
		closeLogger = writer.Close
	}

	logger := zerolog.New(output).With().
		Timestamp().
		Str("service", serviceName).
		Str("version", version).
		Logger()

	return logger, closeLogger, nil
}

// noopShutdown is returned alongside a non-nil error from New, so callers can
// always defer the returned shutdown function unconditionally.
func noopShutdown(context.Context) error {
	return nil
}
