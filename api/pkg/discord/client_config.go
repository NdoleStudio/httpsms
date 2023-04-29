package discord

import "net/http"

type clientConfig struct {
	httpClient    *http.Client
	botToken      string
	applicationID string
	baseURL       string
}

func defaultClientConfig() *clientConfig {
	return &clientConfig{
		httpClient:    http.DefaultClient,
		botToken:      "",
		applicationID: "",
		baseURL:       "https://discord.com/api",
	}
}
