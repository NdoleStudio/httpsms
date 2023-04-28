package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ApplicationService is the API client for the interacting with commands
type ApplicationService service

// CreateCommand creates a new guild command
//
// API Docs: https://discord.com/developers/docs/interactions/application-commands#create-guild-application-command
func (service *ApplicationService) CreateCommand(ctx context.Context, serverID string, channelID string, params *CommandCreateRequest) (*CommandCreateResponse, *Response, error) {
	url := fmt.Sprintf("/applications/%s/guilds/%s/commands", serverID, channelID)
	request, err := service.client.newRequest(ctx, http.MethodPost, url, params)
	if err != nil {
		return nil, nil, err
	}

	response, err := service.client.do(request)
	if err != nil {
		return nil, response, err
	}

	message := new(CommandCreateResponse)
	if err = json.Unmarshal(*response.Body, message); err != nil {
		return nil, response, err
	}

	return message, response, nil
}
