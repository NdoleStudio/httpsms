package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestMCPMiddlewareTracesSafeProtocolMetadata(t *testing.T) {
	const secret = "secret-message-body"

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previousProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	var logs bytes.Buffer
	logger := zerolog.New(&logs)
	middleware := MCPMiddleware(logger, "receive", trace.SpanKindServer)
	handler := middleware(func(context.Context, string, mcp.Request) (mcp.Result, error) {
		return &mcp.CallToolResult{}, nil
	})
	request := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "send_sms",
			Arguments: json.RawMessage(fmt.Sprintf(`{"content":%q}`, secret)),
		},
	}

	_, err := handler(context.Background(), "tools/call", request)
	require.NoError(t, err)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "mcp tools/call", span.Name)
	assert.Equal(t, trace.SpanKindServer, span.SpanKind)
	assert.Equal(t, "mcp", spanAttribute(span, "rpc.system"))
	assert.Equal(t, "tools/call", spanAttribute(span, "rpc.method"))
	assert.Equal(t, "receive", spanAttribute(span, "mcp.message.direction"))
	assert.Equal(t, "send_sms", spanAttribute(span, "mcp.tool.name"))
	assert.Equal(t, "success", spanAttribute(span, "mcp.outcome"))
	assert.NotContains(t, fmt.Sprint(span), secret)

	logOutput := logs.String()
	assert.Contains(t, logOutput, `"rpc.method":"tools/call"`)
	assert.Contains(t, logOutput, `"mcp.tool.name":"send_sms"`)
	assert.NotContains(t, logOutput, secret)
}

func TestMCPMiddlewareDoesNotRecordErrorMessages(t *testing.T) {
	const secret = "sensitive-downstream-error"

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previousProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	var logs bytes.Buffer
	middleware := MCPMiddleware(zerolog.New(&logs), "send", trace.SpanKindClient)
	handler := middleware(func(context.Context, string, mcp.Request) (mcp.Result, error) {
		return nil, errors.New(secret)
	})

	_, err := handler(context.Background(), "sampling/createMessage", &mcp.CreateMessageRequest{
		Params: &mcp.CreateMessageParams{SystemPrompt: secret},
	})
	require.EqualError(t, err, secret)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "error", spanAttribute(spans[0], "mcp.outcome"))
	assert.Equal(t, "*errors.errorString", spanAttribute(spans[0], "error.type"))
	assert.NotContains(t, fmt.Sprint(spans[0]), secret)
	assert.NotContains(t, logs.String(), secret)
}

func spanAttribute(span tracetest.SpanStub, key string) string {
	for _, attr := range span.Attributes {
		if string(attr.Key) == key {
			return attr.Value.AsString()
		}
	}
	return ""
}
