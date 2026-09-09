// Command mcp-server runs the httpSMS MCP service: it loads configuration,
// assembles every dependency (signing keys, Firebase identity
// verification, Redis-backed OAuth/rate-limit state, the typed httpSMS API
// client, and the MCP tool catalog), builds the HTTP surface (see the
// server package), and serves it until asked to shut down.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
	"github.com/NdoleStudio/httpsms/mcp/internal/config"
	"github.com/NdoleStudio/httpsms/mcp/internal/httpsms"
	"github.com/NdoleStudio/httpsms/mcp/internal/oauth"
	"github.com/NdoleStudio/httpsms/mcp/internal/observability"
	"github.com/NdoleStudio/httpsms/mcp/internal/server"
)

// serviceName identifies this service in structured logs and traces.
const serviceName = "httpsms-mcp-server"

// Version is this service's build version, overridden at build time with
// -ldflags "-X main.Version=...". It is published in the MCP
// Implementation and as the observability service.version.
var Version = "dev"

// shutdownTimeout bounds how long graceful shutdown (draining in-flight
// HTTP requests, then closing Redis and telemetry) may take before this
// process exits regardless.
const shutdownTimeout = 10 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, Version); err != nil {
		log.Error().Err(err).Msg("httpSMS MCP server exited with an error")
		stop()
		os.Exit(1)
	}
}

// run loads configuration, assembles every dependency, serves HTTP until
// ctx is cancelled or the listener fails, and then shuts everything down.
//
// It returns an error instead of exiting the process itself, so every
// failure path -- including a listener that never starts (a port already in
// use, an invalid PORT value, a missing bind permission), which must be a
// non-zero process exit rather than a log line followed by a "clean"
// shutdown -- is reachable from a test.
func run(ctx context.Context, version string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	handler, shutdown, err := build(ctx, cfg, version)
	if err != nil {
		return fmt.Errorf("build MCP server: %w", err)
	}

	return serve(ctx, ":"+cfg.Port, handler, shutdown)
}

// serve runs handler on addr until ctx is cancelled or ListenAndServe
// fails, then drains in-flight requests and calls shutdown.
//
// It returns a non-nil error when the listener failed for any reason other
// than a graceful http.ErrServerClosed, or when draining/shutdown itself
// failed; a shutdown triggered by ctx being cancelled (SIGINT/SIGTERM, the
// normal Cloud Run path) returns nil.
func serve(ctx context.Context, addr string, handler http.Handler, shutdown func(context.Context) error) error {
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	var errs []error
	select {
	case <-ctx.Done():
		log.Info().Msg("received shutdown signal")
	case err := <-serveErr:
		if err != nil {
			errs = append(errs, fmt.Errorf("serve HTTP on %q: %w", addr, err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		errs = append(errs, fmt.Errorf("shut down HTTP server: %w", err))
	}
	if err := shutdown(shutdownCtx); err != nil {
		errs = append(errs, fmt.Errorf("shut down MCP server dependencies: %w", err))
	}

	return errors.Join(errs...)
}

// build loads and wires every dependency the httpSMS MCP service needs and
// returns the assembled HTTP handler, a shutdown function that releases
// every resource build itself opened (the Redis client and the
// observability tracer provider), and any assembly error.
//
// build never partially starts serving traffic: it either returns a fully
// wired handler and a working shutdown func, or a non-nil error and a nil
// handler. Callers must still call the returned shutdown func exactly when
// build itself returns a non-nil error only if shutdown is non-nil; on
// error, build closes anything it already opened itself and returns a nil
// shutdown func.
func build(ctx context.Context, cfg config.Config, version string) (http.Handler, func(context.Context) error, error) {
	logger, shutdownObservability, err := observability.New(ctx, serviceName, version)
	if err != nil {
		return nil, nil, fmt.Errorf("build observability: %w", err)
	}

	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		_ = shutdownObservability(ctx)
		return nil, nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	// redis.NewClient always returns a standalone client, never a cluster
	// or ring client: RedisStore's cross-slot refresh-token rotation
	// script and this service's rate limiter both depend on that (see
	// oauth.NewRedisStore's doc comment).
	redisClient := redis.NewClient(redisOptions)

	issuer := strings.TrimRight(cfg.BaseURL.String(), "/")

	keys, err := auth.NewKeySet(cfg.SigningPrivateKeyPEM, cfg.SigningKeyID)
	if err != nil {
		_ = redisClient.Close()
		_ = shutdownObservability(ctx)
		return nil, nil, fmt.Errorf("build signing key set: %w", err)
	}
	if err := keys.Configure(issuer, cfg.MCPAudience, cfg.APIAudience); err != nil {
		_ = redisClient.Close()
		_ = shutdownObservability(ctx)
		return nil, nil, fmt.Errorf("configure signing key set: %w", err)
	}

	firebaseVerifier, err := auth.NewFirebaseVerifier(cfg.FirebaseProjectID, cfg.FirebaseCertsURL.String(), nil, 0, 0)
	if err != nil {
		_ = redisClient.Close()
		_ = shutdownObservability(ctx)
		return nil, nil, fmt.Errorf("build Firebase verifier: %w", err)
	}

	store := oauth.NewRedisStore(redisClient)

	// The Client ID Metadata Document (CIMD) fetch transport must never
	// route through a configured HTTP(S)_PROXY: this service pins the
	// fetch to a validated public IP address (see oauth.ClientResolver)
	// specifically to defeat DNS-rebinding SSRF, and a proxy would
	// reintroduce a second, unvalidated hop between that validation and
	// the actual connection.
	cimdTransport := &http.Transport{Proxy: nil}
	cimdHTTPClient := &http.Client{Timeout: cfg.HTTPTimeout, Transport: cimdTransport}
	resolver := oauth.NewClientResolver(cimdHTTPClient, store)

	oauthServerConfig := oauth.ServerConfig{
		Issuer:               issuer,
		Resource:             cfg.MCPAudience,
		FirebaseAPIKey:       cfg.FirebaseAPIKey,
		FirebaseAuthDomain:   cfg.FirebaseAuthDomain,
		AuthorizationCodeTTL: cfg.AuthorizationCodeTTL,
		AccessTokenTTL:       cfg.AccessTokenTTL,
		RefreshTokenTTL:      cfg.RefreshTokenTTL,
	}

	oauthServer, err := oauth.NewServer(store, resolver, keys, firebaseVerifier, oauthServerConfig)
	if err != nil {
		_ = redisClient.Close()
		_ = shutdownObservability(ctx)
		return nil, nil, fmt.Errorf("build OAuth server: %w", err)
	}

	// cfg.HTTPTimeout bounds every outbound call this service makes,
	// including calls to the httpSMS API: without passing it here the API
	// client would silently keep its own built-in default and the
	// configured HTTP_TIMEOUT would apply only to CIMD fetches.
	apiClient := httpsms.NewClient(cfg.APIURL.String(), httpsms.WithTimeout(cfg.HTTPTimeout))

	handler, err := server.New(cfg, server.Dependencies{
		Logger:                logger,
		Keys:                  keys,
		OAuthServer:           oauthServer,
		OAuthServerConfig:     oauthServerConfig,
		OAuthStore:            store,
		APIClient:             apiClient,
		RedisClient:           redisClient,
		APIDelegationTokenTTL: cfg.APIDelegationTokenTTL,
		ConfirmationTTL:       cfg.ConfirmationTTL,
		RateLimits: server.Limits{
			ReadPerMinute:       cfg.ReadToolsPerMinute,
			SendPerMinute:       cfg.SendToolsPerMinute,
			KeyCreatesPerHour:   cfg.KeyCreatesPerHour,
			KeyRotationsPerHour: cfg.KeyRotationsPerHour,
		},
		Version: version,
	})
	if err != nil {
		_ = redisClient.Close()
		_ = shutdownObservability(ctx)
		return nil, nil, fmt.Errorf("assemble HTTP server: %w", err)
	}

	shutdown := func(shutdownCtx context.Context) error {
		var errs []error
		if err := redisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close Redis client: %w", err))
		}
		if err := shutdownObservability(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("shut down observability: %w", err))
		}
		return errors.Join(errs...)
	}

	return handler, shutdown, nil
}
