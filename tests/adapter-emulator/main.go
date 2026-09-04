package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	callbackAddress = ":9091"
	controlAddress  = ":9092"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	apiBaseURL := environmentOrDefault("API_BASE_URL", "http://api:8000")
	tlsCertificate := environmentOrDefault("ADAPTER_TLS_CERT", "/certs/server.pem")
	tlsKey := environmentOrDefault("ADAPTER_TLS_KEY", "/certs/server-key.pem")

	instance := newEmulator(apiBaseURL, &http.Client{Timeout: 15 * time.Second})
	callbackServer := newHTTPServer(callbackAddress, instance.notificationHandler())
	callbackServer.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	controlServer := newHTTPServer(controlAddress, instance.controlHandler())

	serverErrors := make(chan error, 2)
	go func() {
		log.Printf("[ADAPTER] HTTPS callback server listening on %s", callbackAddress)
		serverErrors <- callbackServer.ListenAndServeTLS(tlsCertificate, tlsKey)
	}()
	go func() {
		log.Printf("[ADAPTER] HTTP control server listening on %s", controlAddress)
		serverErrors <- controlServer.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	var serveErr error
	select {
	case <-signals:
		log.Printf("[ADAPTER] shutdown signal received")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			serveErr = err
		}
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	shutdownErr := errors.Join(
		callbackServer.Shutdown(shutdownContext),
		controlServer.Shutdown(shutdownContext),
	)
	return errors.Join(serveErr, shutdownErr)
}

func newHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func environmentOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
