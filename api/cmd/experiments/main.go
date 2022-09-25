package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel/metric/instrument"

	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/global"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()

	// Configure OpenTelemetry with sensible defaults.
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(os.Getenv("UPTRACE_DSN")),
		uptrace.WithServiceName(os.Getenv("httpsms")),
		uptrace.WithServiceVersion("1.0.0"),
	)
	// Send buffered spans and free resources.
	defer uptrace.Shutdown(ctx)

	// Create a tracer. Usually, tracer is a global variable.
	tracer := otel.Tracer("app_or_package_name")

	// Create a root span (a trace) to measure some operation.
	ctx, main := tracer.Start(ctx, "main-operation")
	// End the span when the operation we are measuring is done.
	defer main.End()

	// We pass the main (parent) span in the ctx so we can reconstruct a trace later.
	_, child1 := tracer.Start(ctx, "child1-of-main")
	child1.SetAttributes(attribute.String("key1", "value1"))
	child1.RecordError(errors.New("error1"))
	child1.End()

	_, child2 := tracer.Start(ctx, "child2-of-main")
	child2.SetAttributes(attribute.Int("key2", 42), attribute.Float64("key3", 123.456))
	child2.End()

	fmt.Printf("trace: %s\n", uptrace.TraceURL(main))

	meter := global.MeterProvider().Meter("httpsms")
	counter, err := meter.SyncInt64().Counter(
		"test.my_counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("Just a test counter"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Increment the counter.
	counter.Add(ctx, 1, attribute.String("foo", "bar"))
	counter.Add(ctx, 10, attribute.String("hello", "world"))
}
