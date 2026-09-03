package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxControlBodyBytes = 1024 * 1024

type gatewayRegistration struct {
	PhoneNumber string `json:"phone_number"`
	PhoneAPIKey string `json:"phone_api_key"`
}

type incomingMessageRequest struct {
	Contact   string `json:"contact"`
	Content   string `json:"content"`
	Encrypted bool   `json:"encrypted"`
}

func (instance *emulator) controlHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /test/gateways/{gatewayID}", instance.handleGatewayRegistration)
	mux.HandleFunc("POST /test/gateways/{gatewayID}/incoming", instance.handleIncomingMessage)
	mux.HandleFunc("GET /test/gateways/{gatewayID}/notifications", instance.handleNotificationRecords)
	mux.HandleFunc("GET /health", instance.handleHealth)
	return mux
}

func (instance *emulator) handleGatewayRegistration(writer http.ResponseWriter, request *http.Request) {
	var registration gatewayRegistration
	if err := decodeControlJSON(writer, request, &registration); err != nil {
		writeControlError(writer, http.StatusBadRequest, err)
		return
	}
	registration.PhoneNumber = strings.TrimSpace(registration.PhoneNumber)
	registration.PhoneAPIKey = strings.TrimSpace(registration.PhoneAPIKey)
	if registration.PhoneNumber == "" || registration.PhoneAPIKey == "" {
		writeControlError(writer, http.StatusBadRequest, errors.New("phone_number and phone_api_key are required"))
		return
	}

	instance.registerGateway(request.PathValue("gatewayID"), registration)
	writer.WriteHeader(http.StatusNoContent)
}

func (instance *emulator) handleIncomingMessage(writer http.ResponseWriter, request *http.Request) {
	registeredGateway, ok := instance.loadGateway(request.PathValue("gatewayID"))
	if !ok {
		writeControlError(writer, http.StatusNotFound, errors.New("unknown gateway"))
		return
	}

	var incoming incomingMessageRequest
	if err := decodeControlJSON(writer, request, &incoming); err != nil {
		writeControlError(writer, http.StatusBadRequest, err)
		return
	}
	incoming.Contact = strings.TrimSpace(incoming.Contact)
	if incoming.Contact == "" {
		writeControlError(writer, http.StatusBadRequest, errors.New("contact is required"))
		return
	}

	message, err := instance.receiveMessage(request.Context(), registeredGateway, incoming)
	if err != nil {
		writeControlError(writer, http.StatusBadGateway, err)
		return
	}
	writeControlJSON(writer, http.StatusOK, map[string]any{"data": message})
}

func (instance *emulator) handleNotificationRecords(writer http.ResponseWriter, request *http.Request) {
	gatewayID := request.PathValue("gatewayID")
	if _, ok := instance.loadGateway(gatewayID); !ok {
		writeControlError(writer, http.StatusNotFound, errors.New("unknown gateway"))
		return
	}
	records := instance.listGatewayRecords(gatewayID)
	if messageID := strings.TrimSpace(request.URL.Query().Get("message_id")); messageID != "" {
		filtered := make([]notificationRecord, 0, len(records))
		for _, record := range records {
			if record.MessageID == messageID {
				filtered = append(filtered, record)
			}
		}
		records = filtered
	}
	writeControlJSON(writer, http.StatusOK, map[string]any{
		"data": records,
	})
}

func (*emulator) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("ok\n"))
}

func decodeControlJSON(writer http.ResponseWriter, request *http.Request, result any) error {
	request.Body = http.MaxBytesReader(writer, request.Body, maxControlBodyBytes)
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func writeControlError(writer http.ResponseWriter, status int, err error) {
	writeControlJSON(writer, status, map[string]any{"error": err.Error()})
}

func writeControlJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
