package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const maxCallbackBodyBytes = 1024 * 1024

// notificationJWTIssuer must match the issuer the httpSMS API signs adapter notification
// tokens with (see api/pkg/services/http_notification_sender.go).
const notificationJWTIssuer = "api.httpsms.com"

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

	authorization := request.Header.Get("Authorization")
	if err := verifyNotificationAuth(authorization, registeredGateway.PhoneID); err != nil {
		log.Printf("[ADAPTER] rejected notification for gateway=%s: %v", gatewayID, err)
		http.Error(writer, fmt.Sprintf("invalid notification token: %v", err), http.StatusUnauthorized)
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
		authorization,
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

// verifyNotificationAuth validates the JWT the httpSMS API signs notification requests with,
// using the gateway's phone ID as the HMAC-SHA256 secret (see
// api/pkg/services/http_notification_sender.go getAuthToken).
func verifyNotificationAuth(authorization string, phoneID string) error {
	tokenString, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok || strings.TrimSpace(tokenString) == "" {
		return fmt.Errorf("missing bearer token")
	}

	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(phoneID), nil
	})
	if err != nil {
		return fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return fmt.Errorf("token is not valid")
	}
	if claims.Subject != phoneID {
		return fmt.Errorf("subject mismatch")
	}
	if claims.Issuer != notificationJWTIssuer {
		return fmt.Errorf("issuer mismatch")
	}
	return nil
}
