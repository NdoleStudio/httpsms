package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ChannelService is the API client for interacting with channels
type ChannelService service

// CreateMessage sends a message to a guild text or DM channel.
//
// API Docs: https://discord.com/developers/docs/resources/channel#create-message
func (service *ChannelService) CreateMessage(ctx context.Context, channelID string, payload map[string]any) (*map[string]any, *Response, error) {
	request, err := service.client.newRequest(ctx, http.MethodPost, fmt.Sprintf("/channels/%s/messages", channelID), payload)
	if err != nil {
		return nil, nil, err
	}

	response, err := service.client.do(request)
	if err != nil {
		return nil, response, err
	}

	message := new(map[string]any)
	if err = json.Unmarshal(*response.Body, message); err != nil {
		return nil, response, err
	}

	return message, response, nil
}

// Get a channel by ID
//
// API Docs: https://discord.com/developers/docs/resources/channel#get-channel
func (service *ChannelService) Get(ctx context.Context, channelID string) (*map[string]any, *Response, error) {
	request, err := service.client.newRequest(ctx, http.MethodGet, fmt.Sprintf("/channels/%s", channelID), nil)
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
