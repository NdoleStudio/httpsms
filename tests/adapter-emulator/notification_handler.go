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

	notificationID := strings.TrimSpace(request.Header.Get("X-httpSMS-Notification-ID"))
	if notificationID == "" {
		http.Error(writer, "missing X-httpSMS-Notification-ID", http.StatusBadRequest)
		return
	}

	request.Body = http.MaxBytesReader(writer, request.Body, maxCallbackBodyBytes)
	var envelope callbackEnvelope
	if err := json.NewDecoder(request.Body).Decode(&envelope); err != nil {
		http.Error(writer, "invalid callback payload", http.StatusBadRequest)
		return
	}

	kind, messageID, validationErr := notificationKind(envelope.Message.Data)
	_, firstDelivery := instance.beginNotification(
		notificationID,
		gatewayID,
		envelope.Message.Data,
		kind,
		messageID,
	)
	log.Printf(
		"[ADAPTER] callback notification=%s gateway=%s data=%v should_process=%t",
		notificationID,
		gatewayID,
		envelope.Message.Data,
		firstDelivery,
	)
	if !firstDelivery {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	if validationErr != nil {
		instance.markNotificationFailed(notificationID, validationErr)
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
		instance.markNotificationFailed(notificationID, processingErr)
		log.Printf("[ADAPTER] notification %s failed: %v", notificationID, processingErr)
		http.Error(writer, "notification processing failed", http.StatusInternalServerError)
		return
	}

	instance.markNotificationProcessed(notificationID)
	log.Printf("[ADAPTER] notification %s processed as %s", notificationID, kind)
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
