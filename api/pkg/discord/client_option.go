package discord

import (
	"net/http"
	"strings"
)

// Option is options for constructing a client
type Option interface {
	apply(config *clientConfig)
}

type clientOptionFunc func(config *clientConfig)

func (fn clientOptionFunc) apply(config *clientConfig) {
	fn(config)
}

// WithHTTPClient sets the underlying HTTP client used for API requests.
// By default, http.DefaultClient is used.
func WithHTTPClient(httpClient *http.Client) Option {
	return clientOptionFunc(func(config *clientConfig) {
		if httpClient != nil {
			config.httpClient = httpClient
		}
	})
}

// WithBaseURL set's the base url for the discord API
func WithBaseURL(baseURL string) Option {
	return clientOptionFunc(func(config *clientConfig) {
		if baseURL != "" {
			config.baseURL = strings.TrimRight(baseURL, "/")
		}
	})
}

// WithApplicationID sets the discord bot application ID
func WithApplicationID(applicationID string) Option {
	return clientOptionFunc(func(config *clientConfig) {
		config.applicationID = applicationID
	})
}

// WithBotToken sets the discord bot token
func WithBotToken(botToken string) Option {
	return clientOptionFunc(func(config *clientConfig) {
		config.botToken = botToken
	})
}
