package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const maxAPIErrorBodyBytes = 4 * 1024

func (instance *emulator) fetchOutstanding(
	ctx context.Context,
	registeredGateway gateway,
	messageID string,
) (map[string]any, error) {
	endpoint, err := url.Parse(instance.apiBaseURL + "/v1/messages/outstanding")
	if err != nil {
		return nil, fmt.Errorf("build outstanding message URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("message_id", messageID)
	endpoint.RawQuery = query.Encode()

	log.Printf("[ADAPTER] fetching outstanding message %s", messageID)
	var response struct {
		Data map[string]any `json:"data"`
	}
	if err := instance.doAPIRequest(ctx, registeredGateway, http.MethodGet, endpoint.String(), nil, &response); err != nil {
		return nil, fmt.Errorf("fetch outstanding message %s: %w", messageID, err)
	}
	return response.Data, nil
}

func (instance *emulator) fireMessageEvent(
	ctx context.Context,
	registeredGateway gateway,
	messageID string,
	eventName string,
) error {
	payload := map[string]any{
		"event_name": eventName,
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
	}
	endpoint := fmt.Sprintf("%s/v1/messages/%s/events", instance.apiBaseURL, url.PathEscape(messageID))
	log.Printf("[ADAPTER] posting %s for message %s", eventName, messageID)
	if err := instance.doAPIRequest(ctx, registeredGateway, http.MethodPost, endpoint, payload, nil); err != nil {
		return fmt.Errorf("post %s event for message %s: %w", eventName, messageID, err)
	}
	return nil
}

func (instance *emulator) receiveMessage(
	ctx context.Context,
	registeredGateway gateway,
	request incomingMessageRequest,
) (map[string]any, error) {
	payload := map[string]any{
		"from":      request.Contact,
		"to":        registeredGateway.PhoneNumber,
		"content":   request.Content,
		"encrypted": request.Encrypted,
		"sim":       "SIM1",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}
	log.Printf("[ADAPTER] posting incoming message for gateway phone %s", registeredGateway.PhoneNumber)

	var response struct {
		Data map[string]any `json:"data"`
	}
	if err := instance.doAPIRequest(
		ctx,
		registeredGateway,
		http.MethodPost,
		instance.apiBaseURL+"/v1/messages/receive",
		payload,
		&response,
	); err != nil {
		return nil, fmt.Errorf("post incoming message: %w", err)
	}
	return response.Data, nil
}

func (instance *emulator) storeHeartbeat(ctx context.Context, registeredGateway gateway) error {
	payload := map[string]any{
		"phone_numbers": []string{registeredGateway.PhoneNumber},
		"charging":      true,
	}
	log.Printf("[ADAPTER] posting heartbeat for %s", registeredGateway.PhoneNumber)
	if err := instance.doAPIRequest(
		ctx,
		registeredGateway,
		http.MethodPost,
		instance.apiBaseURL+"/v1/heartbeats",
		payload,
		nil,
	); err != nil {
		return fmt.Errorf("post heartbeat for %s: %w", registeredGateway.PhoneNumber, err)
	}
	return nil
}

func (instance *emulator) doAPIRequest(
	ctx context.Context,
	registeredGateway gateway,
	method string,
	endpoint string,
	payload any,
	result any,
) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("x-api-key", registeredGateway.PhoneAPIKey)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := instance.client.Do(request)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		errorBody, readErr := io.ReadAll(io.LimitReader(response.Body, maxAPIErrorBodyBytes))
		if readErr != nil {
			return fmt.Errorf("unexpected status %d and read error body: %w", response.StatusCode, readErr)
		}
		return fmt.Errorf("unexpected status %d: %s", response.StatusCode, string(errorBody))
	}
	if result == nil || response.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}
	return nil
}
