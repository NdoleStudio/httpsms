package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// readLineWithin reads one line from reader, returning an error if nothing
// arrives within timeout. A wrapper that swallows Flush shows up here as a
// timeout: the bytes would sit in net/http's response buffer until the
// handler returned, and the handlers under test deliberately do not return
// until the test has already read.
//
// It returns the timeout as an error rather than failing the test itself,
// so the caller can always unblock the still-running handler before
// asserting (a t.Fatal here would leave the handler parked forever and
// deadlock httptest.Server.Close).
func readLineWithin(reader *bufio.Reader, timeout time.Duration) (string, error) {
	type readResult struct {
		line string
		err  error
	}
	lines := make(chan readResult, 1)
	go func() {
		line, err := reader.ReadString('\n')
		lines <- readResult{line: line, err: err}
	}()

	select {
	case result := <-lines:
		return result.line, result.err
	case <-time.After(timeout):
		return "", errors.New("timed out waiting for a streamed line: the response was buffered, not flushed")
	}
}

// streamingClient returns an *http.Client that bounds only the wait for
// response headers, never the body. A streaming handler that never flushes
// leaves net/http buffering headers and body together until the handler
// returns, so without this bound the request itself -- not just the body
// read -- would block until the (deliberately blocked) handler finished.
func streamingClient() *http.Client {
	return &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 3 * time.Second}}
}

// TestStreamedResponsesAreFlushedThroughTheMiddlewareChain is the
// regression test for a response writer wrapper that silently breaks
// streaming: both statusCapturingWriter (logging) and noStoreWriter
// (/mcp cache control) sit between net/http and the MCP SDK's SSE writer,
// which flushes every event through http.NewResponseController. A wrapper
// that implements neither Flush nor Unwrap makes that flush a no-op, and
// every SSE event -- progress notifications, elicitation requests,
// subscription updates -- is withheld until the whole response completes.
//
// It exercises the exact production middleware composition
// (withBaseMiddleware) around the exact /mcp cache-control wrapper
// (withNoStore).
func TestStreamedResponsesAreFlushedThroughTheMiddlewareChain(t *testing.T) {
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	flushErrors := make(chan error, 1)

	streaming := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: first\n\n"))

		// This is what mcp/event.go does for every SSE event it writes.
		flushErrors <- http.NewResponseController(w).Flush()

		<-release
		_, _ = w.Write([]byte("data: second\n\n"))
		_ = http.NewResponseController(w).Flush()
	})

	httpServer := httptest.NewServer(withBaseMiddleware(zerolog.Nop(), false, withNoStore(streaming)))
	// The handler blocks until released, and httptest.Server.Close waits
	// for in-flight requests: release first, close second.
	defer httpServer.Close()
	defer releaseOnce()

	resp, err := streamingClient().Get(httpServer.URL)
	require.NoError(t, err, "no response headers arrived: the response was buffered, not flushed")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))

	reader := bufio.NewReader(resp.Body)
	firstEvent, readErr := readLineWithin(reader, 3*time.Second)

	// Unblock the handler before asserting anything, so a failure here is
	// a failed assertion rather than a deadlocked httptest.Server.Close.
	releaseOnce()

	require.NoError(t, readErr)
	require.Equal(t, "data: first\n", firstEvent)

	// http.ResponseController.Flush must find a real flusher through the
	// wrappers, never return http.ErrNotSupported.
	require.NoError(t, <-flushErrors)

	blankLine, err := readLineWithin(reader, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "\n", blankLine)

	secondEvent, err := readLineWithin(reader, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "data: second\n", secondEvent)
}

// TestMCPSSENotificationsReachTheClientBeforeTheToolReturns is the same
// regression, driven by the real MCP SDK rather than a hand-written SSE
// handler: a tool sends a progress notification and then blocks. With
// working flush plumbing the notification reaches the client while the
// tool is still running; without it the client sees nothing until the tool
// returns.
//
// It uses the SDK's SSE response mode (JSONResponse: false) because that is
// the only mode in which the SDK streams more than one message per
// response; the assembled /mcp endpoint runs in JSON mode today, but it
// shares these very wrappers, so a wrapper regression would break the
// moment streaming is turned on.
func TestMCPSSENotificationsReachTheClientBeforeTheToolReturns(t *testing.T) {
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	defer releaseOnce()

	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, &mcp.ServerOptions{})
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "slow_stream",
		Description: "sends a progress notification, then blocks until released",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		if token := req.Params.GetProgressToken(); token != nil {
			_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
				ProgressToken: token,
				Message:       "working",
				Progress:      1,
			})
		}
		<-release
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "done"}}}, nil, nil
	})

	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return mcpServer },
		&mcp.StreamableHTTPOptions{
			Stateless:           true,
			MaxRequestBodyBytes: maxMCPRequestBodyBytes,
		},
	)

	httpServer := httptest.NewServer(withBaseMiddleware(zerolog.Nop(), false, withNoStore(mcpHandler)))
	defer httpServer.Close()
	defer releaseOnce()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{` +
		`"name":"slow_stream","arguments":{},"_meta":{"progressToken":"tok-1"}}}`

	req, err := http.NewRequest(http.MethodPost, httpServer.URL, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := streamingClient().Do(req)
	require.NoError(t, err, "no response headers arrived: the response was buffered, not flushed")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))

	reader := bufio.NewReader(resp.Body)

	var (
		payload string
		readErr error
	)
	for {
		var line string
		line, readErr = readLineWithin(reader, 5*time.Second)
		if readErr != nil {
			break
		}
		if data, found := strings.CutPrefix(strings.TrimSpace(line), "data: "); found {
			payload = data
			break
		}
	}

	// Unblock the tool before asserting, so a broken flush fails as an
	// assertion instead of deadlocking httptest.Server.Close.
	releaseOnce()
	require.NoError(t, readErr)

	var notification struct {
		Method string `json:"method"`
		Params struct {
			Message string `json:"message"`
		} `json:"params"`
	}
	require.NoError(t, json.Unmarshal([]byte(payload), &notification))
	require.Equal(t, "notifications/progress", notification.Method)
	require.Equal(t, "working", notification.Params.Message)
}
