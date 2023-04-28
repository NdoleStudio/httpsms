package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GuildService is the API client for interacting with guilds
type GuildService service

// Get the guild object for the given id.
//
// API Docs: https://discord.com/developers/docs/resources/guild#get-guild
func (service *GuildService) Get(ctx context.Context, guildID string) (*map[string]any, *Response, error) {
	request, err := service.client.newRequest(ctx, http.MethodGet, fmt.Sprintf("/guilds/%s", guildID), nil)
	if err != nil {
		return nil, nil, err
	}

	response, err := service.client.do(request)
	if err != nil {
		return nil, response, err
	}

	channel := new(map[string]any)
	if err = json.Unmarshal(*response.Body, channel); err != nil {
		return nil, response, err
	}

	return channel, response, nil
}
