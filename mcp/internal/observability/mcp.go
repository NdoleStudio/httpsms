package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const mcpInstrumentationName = "github.com/NdoleStudio/httpsms/mcp"

// MCPMiddleware traces and logs one MCP protocol method in direction. It
// deliberately records only bounded protocol metadata: method, direction,
// outcome, and tool/prompt name. Arguments, prompt text, sampled messages,
// resource URIs, response content, tokens, and request metadata are never
// recorded.
func MCPMiddleware(logger zerolog.Logger, direction string, kind trace.SpanKind) mcp.Middleware {
	tracer := otel.Tracer(mcpInstrumentationName)

	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			attributes := []attribute.KeyValue{
				attribute.String("rpc.system", "mcp"),
				attribute.String("rpc.method", method),
				attribute.String("mcp.message.direction", direction),
			}
			attributes = append(attributes, safeRequestAttributes(req)...)

			ctx, span := tracer.Start(
				ctx,
				"mcp "+method,
				trace.WithSpanKind(kind),
				trace.WithAttributes(attributes...),
			)
			defer span.End()

			start := time.Now()
			span.AddEvent("mcp.request", trace.WithAttributes(
				attribute.String("mcp.message.direction", direction),
			))

			result, err = next(ctx, method, req)

			outcome := "success"
			if err != nil {
				outcome = "error"
				errorType := fmt.Sprintf("%T", err)
				span.SetAttributes(attribute.String("error.type", errorType))
				span.SetStatus(codes.Error, "MCP method failed")
			} else {
				span.SetStatus(codes.Ok, "")
			}
			span.SetAttributes(attribute.String("mcp.outcome", outcome))
			span.AddEvent("mcp.response", trace.WithAttributes(
				attribute.String("mcp.outcome", outcome),
			))

			event := logger.Info()
			if err != nil {
				event = logger.Error().Str("error.type", fmt.Sprintf("%T", err))
			}
			spanContext := span.SpanContext()
			if spanContext.IsValid() {
				event = event.
					Str("trace_id", spanContext.TraceID().String()).
					Str("span_id", spanContext.SpanID().String())
			}
			for _, attr := range attributes {
				event = addLogAttribute(event, attr)
			}
			event.
				Str("outcome", outcome).
				Dur("duration", time.Since(start)).
				Msg("mcp request")

			return result, err
		}
	}
}

func safeRequestAttributes(req mcp.Request) []attribute.KeyValue {
	if req == nil || req.GetParams() == nil {
		return nil
	}

	switch params := req.GetParams().(type) {
	case *mcp.CallToolParamsRaw:
		return []attribute.KeyValue{attribute.String("mcp.tool.name", params.Name)}
	case *mcp.GetPromptParams:
		return []attribute.KeyValue{attribute.String("mcp.prompt.name", params.Name)}
	default:
		return nil
	}
}

func addLogAttribute(event *zerolog.Event, attr attribute.KeyValue) *zerolog.Event {
	switch attr.Value.Type() {
	case attribute.STRING:
		return event.Str(string(attr.Key), attr.Value.AsString())
	case attribute.BOOL:
		return event.Bool(string(attr.Key), attr.Value.AsBool())
	case attribute.INT64:
		return event.Int64(string(attr.Key), attr.Value.AsInt64())
	case attribute.FLOAT64:
		return event.Float64(string(attr.Key), attr.Value.AsFloat64())
	default:
		return event
	}
}
