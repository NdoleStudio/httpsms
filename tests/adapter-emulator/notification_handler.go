package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

const maxCallbackBodyBytes = 1024 * 1024

type callbackEnvelope struct {
	Message struct {
		Token string            `json:"token"`
		Data  map[string]string `json:"data"`
	} `json:"message"`
}

func (instance *emulator) notificationHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notifications/{gatewayID}", instance.handleNotification)
	return mux
}

func (instance *emulator) handleNotification(writer http.ResponseWriter, request *http.Request) {
	gatewayID := request.PathValue("gatewayID")
	registeredGateway, ok := instance.loadGateway(gatewayID)
	if !ok {
		http.Error(writer, "unknown gateway", http.StatusNotFound)
		return
	}

	request.Body = http.MaxBytesReader(writer, request.Body, maxCallbackBodyBytes)
	var envelope callbackEnvelope
	if err := json.NewDecoder(request.Body).Decode(&envelope); err != nil {
		http.Error(writer, "invalid callback payload", http.StatusBadRequest)
		return
	}

	kind, messageID, validationErr := notificationKind(envelope.Message.Data)
	record := instance.recordNotification(
		gatewayID,
		envelope.Message.Data,
		kind,
		messageID,
	)
	log.Printf(
		"[ADAPTER] callback gateway=%s data=%v",
		gatewayID,
		envelope.Message.Data,
	)
	if validationErr != nil {
		instance.markNotificationFailed(record, validationErr)
		http.Error(writer, validationErr.Error(), http.StatusBadRequest)
		return
	}

	var processingErr error
	switch kind {
	case "message":
		_, processingErr = instance.fetchOutstanding(request.Context(), registeredGateway, messageID)
		if processingErr == nil {
			processingErr = instance.fireMessageEvent(request.Context(), registeredGateway, messageID, "SENT")
		}
		if processingErr == nil {
			processingErr = instance.fireMessageEvent(request.Context(), registeredGateway, messageID, "DELIVERED")
		}
	case "heartbeat":
		processingErr = instance.storeHeartbeat(request.Context(), registeredGateway)
	}
	if processingErr != nil {
		instance.markNotificationFailed(record, processingErr)
		log.Printf("[ADAPTER] notification failed: %v", processingErr)
		http.Error(writer, "notification processing failed", http.StatusInternalServerError)
		return
	}

	instance.markNotificationProcessed(record)
	log.Printf("[ADAPTER] notification processed as %s", kind)
	writer.WriteHeader(http.StatusNoContent)
}

func notificationKind(data map[string]string) (kind string, messageID string, err error) {
	messageID = strings.TrimSpace(data["KEY_MESSAGE_ID"])
	heartbeatID := strings.TrimSpace(data["KEY_HEARTBEAT_ID"])

	switch {
	case messageID != "" && heartbeatID == "":
		return "message", messageID, nil
	case heartbeatID != "" && messageID == "":
		return "heartbeat", "", nil
	default:
		return "", "", fmt.Errorf("unsupported notification data")
	}
}
