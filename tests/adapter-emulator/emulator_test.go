package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestBeginNotificationDeduplicatesAndCopiesRecords(t *testing.T) {
	t.Parallel()

	instance := newEmulator("http://api.example", http.DefaultClient)
	instance.registerGateway("gateway-1", gatewayRegistration{
		PhoneNumber: "+18005550199",
		PhoneAPIKey: "phone-key",
	})

	record, firstDelivery := instance.beginNotification(
		"notification-1",
		"gateway-1",
		map[string]string{"KEY_MESSAGE_ID": "message-1"},
		"message",
		"message-1",
	)
	if !firstDelivery {
		t.Fatal("first delivery was treated as a duplicate")
	}
	if record.Attempts != 1 {
		t.Fatalf("first delivery attempts = %d, want 1", record.Attempts)
	}

	instance.markNotificationProcessed("notification-1")
	_, firstDelivery = instance.beginNotification(
		"notification-1",
		"gateway-1",
		map[string]string{"KEY_MESSAGE_ID": "message-1"},
		"message",
		"message-1",
	)
	if firstDelivery {
		t.Fatal("duplicate delivery was treated as the first delivery")
	}

	records := instance.listGatewayRecords("gateway-1")
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	if records[0].Attempts != 2 {
		t.Fatalf("duplicate attempts = %d, want 2", records[0].Attempts)
	}
	if !records[0].Processed {
		t.Fatal("processed state was not retained")
	}

	records[0].Data["KEY_MESSAGE_ID"] = "mutated"
	records[0].Attempts = 99
	fresh := instance.listGatewayRecords("gateway-1")
	if fresh[0].Data["KEY_MESSAGE_ID"] != "message-1" || fresh[0].Attempts != 2 {
		t.Fatalf("record list returned mutable state: %#v", fresh[0])
	}
}

func TestNotificationHandlerProcessesMessageOnce(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var outstandingCalls int
	var events []string
	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("x-api-key") != "phone-key" {
			t.Errorf("x-api-key = %q, want phone-key", request.Header.Get("x-api-key"))
		}

		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/v1/messages/outstanding":
			mu.Lock()
			outstandingCalls++
			mu.Unlock()
			if request.URL.Query().Get("message_id") != "message-1" {
				t.Errorf("message_id = %q, want message-1", request.URL.Query().Get("message_id"))
			}
			writeJSON(writer, http.StatusOK, map[string]any{
				"data": map[string]any{"id": "message-1"},
			})
		case request.Method == http.MethodPost && request.URL.Path == "/v1/messages/message-1/events":
			var payload struct {
				EventName string `json:"event_name"`
				Timestamp string `json:"timestamp"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode event: %v", err)
			}
			if payload.Timestamp == "" {
				t.Error("event timestamp is empty")
			}
			mu.Lock()
			events = append(events, payload.EventName)
			mu.Unlock()
			writeJSON(writer, http.StatusOK, map[string]any{"data": map[string]any{"id": "message-1"}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer api.Close()

	instance := newEmulator(api.URL, api.Client())
	instance.registerGateway("gateway-1", gatewayRegistration{
		PhoneNumber: "+18005550199",
		PhoneAPIKey: "phone-key",
	})

	body := callbackBody(t, map[string]string{"KEY_MESSAGE_ID": "message-1"})
	for range 2 {
		request := httptest.NewRequest(
			http.MethodPost,
			"/notifications/gateway-1",
			bytes.NewReader(body),
		)
		request.Header.Set("X-httpSMS-Notification-ID", "notification-1")
		response := httptest.NewRecorder()
		instance.notificationHandler().ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("callback status = %d, want 204: %s", response.Code, response.Body.String())
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if outstandingCalls != 1 {
		t.Fatalf("outstanding calls = %d, want 1", outstandingCalls)
	}
	if !reflect.DeepEqual(events, []string{"SENT", "DELIVERED"}) {
		t.Fatalf("events = %#v, want SENT then DELIVERED", events)
	}

	records := instance.listGatewayRecords("gateway-1")
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	record := records[0]
	if record.Kind != "message" || record.MessageID != "message-1" || !record.Processed || record.Attempts != 2 {
		t.Fatalf("unexpected message record: %#v", record)
	}
}

func TestNotificationHandlerStoresHeartbeat(t *testing.T) {
	t.Parallel()

	var heartbeatPayload map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/heartbeats" {
			http.NotFound(writer, request)
			return
		}
		if request.Header.Get("x-api-key") != "phone-key" {
			t.Errorf("x-api-key = %q, want phone-key", request.Header.Get("x-api-key"))
		}
		if err := json.NewDecoder(request.Body).Decode(&heartbeatPayload); err != nil {
			t.Errorf("decode heartbeat: %v", err)
		}
		writeJSON(writer, http.StatusCreated, map[string]any{"data": []any{}})
	}))
	defer api.Close()

	instance := newEmulator(api.URL, api.Client())
	instance.registerGateway("gateway-1", gatewayRegistration{
		PhoneNumber: "+18005550199",
		PhoneAPIKey: "phone-key",
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/notifications/gateway-1",
		bytes.NewReader(callbackBody(t, map[string]string{"KEY_HEARTBEAT_ID": "heartbeat-1"})),
	)
	request.Header.Set("X-httpSMS-Notification-ID", "notification-1")
	response := httptest.NewRecorder()
	instance.notificationHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("callback status = %d, want 204: %s", response.Code, response.Body.String())
	}

	if !reflect.DeepEqual(heartbeatPayload["phone_numbers"], []any{"+18005550199"}) {
		t.Fatalf("phone_numbers = %#v, want gateway phone", heartbeatPayload["phone_numbers"])
	}
	if heartbeatPayload["charging"] != true {
		t.Fatalf("charging = %#v, want true", heartbeatPayload["charging"])
	}

	records := instance.listGatewayRecords("gateway-1")
	if len(records) != 1 || records[0].Kind != "heartbeat" || !records[0].Processed {
		t.Fatalf("unexpected heartbeat records: %#v", records)
	}
}

func TestNotificationHandlerRetainsProcessingFailure(t *testing.T) {
	t.Parallel()

	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "outstanding unavailable", http.StatusServiceUnavailable)
	}))
	defer api.Close()

	instance := newEmulator(api.URL, api.Client())
	instance.registerGateway("gateway-1", gatewayRegistration{
		PhoneNumber: "+18005550199",
		PhoneAPIKey: "phone-key",
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/notifications/gateway-1",
		bytes.NewReader(callbackBody(t, map[string]string{"KEY_MESSAGE_ID": "message-1"})),
	)
	request.Header.Set("X-httpSMS-Notification-ID", "notification-1")
	response := httptest.NewRecorder()
	instance.notificationHandler().ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("callback status = %d, want 500: %s", response.Code, response.Body.String())
	}

	records := instance.listGatewayRecords("gateway-1")
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	if records[0].Processed {
		t.Fatal("failed notification was marked processed")
	}
	if !strings.Contains(records[0].Error, "fetch outstanding message") {
		t.Fatalf("record error = %q, want fetch context", records[0].Error)
	}
}

func TestNotificationHandlerRetriesFailedDeliveryWithSameID(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var outstandingCalls int
	var events []string
	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/v1/messages/outstanding":
			mu.Lock()
			outstandingCalls++
			firstCall := outstandingCalls == 1
			mu.Unlock()
			if firstCall {
				http.Error(writer, "outstanding unavailable", http.StatusServiceUnavailable)
				return
			}
			writeJSON(writer, http.StatusOK, map[string]any{
				"data": map[string]any{"id": "message-1"},
			})
		case request.Method == http.MethodPost && request.URL.Path == "/v1/messages/message-1/events":
			var payload struct {
				EventName string `json:"event_name"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode event: %v", err)
			}
			mu.Lock()
			events = append(events, payload.EventName)
			mu.Unlock()
			writeJSON(writer, http.StatusOK, map[string]any{"data": map[string]any{"id": "message-1"}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer api.Close()

	instance := newEmulator(api.URL, api.Client())
	instance.registerGateway("gateway-1", gatewayRegistration{
		PhoneNumber: "+18005550199",
		PhoneAPIKey: "phone-key",
	})
	handler := instance.notificationHandler()
	body := callbackBody(t, map[string]string{"KEY_MESSAGE_ID": "message-1"})

	firstRequest := httptest.NewRequest(http.MethodPost, "/notifications/gateway-1", bytes.NewReader(body))
	firstRequest.Header.Set("X-httpSMS-Notification-ID", "notification-1")
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, firstRequest)
	if firstResponse.Code != http.StatusInternalServerError {
		t.Fatalf("first callback status = %d, want 500: %s", firstResponse.Code, firstResponse.Body.String())
	}

	secondRequest := httptest.NewRequest(http.MethodPost, "/notifications/gateway-1", bytes.NewReader(body))
	secondRequest.Header.Set("X-httpSMS-Notification-ID", "notification-1")
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, secondRequest)
	if secondResponse.Code != http.StatusNoContent {
		t.Fatalf("retry callback status = %d, want 204: %s", secondResponse.Code, secondResponse.Body.String())
	}

	mu.Lock()
	defer mu.Unlock()
	if outstandingCalls != 2 {
		t.Fatalf("outstanding calls = %d, want 2", outstandingCalls)
	}
	if !reflect.DeepEqual(events, []string{"SENT", "DELIVERED"}) {
		t.Fatalf("events = %#v, want SENT then DELIVERED", events)
	}

	records := instance.listGatewayRecords("gateway-1")
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	if records[0].Attempts != 2 || !records[0].Processed || records[0].Error != "" {
		t.Fatalf("unexpected retried record: %#v", records[0])
	}
}

func TestControlHandlerRegistersGatewayAndReceivesIncomingMessage(t *testing.T) {
	t.Parallel()

	var receivePayload map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/messages/receive" {
			http.NotFound(writer, request)
			return
		}
		if request.Header.Get("x-api-key") != "phone-key" {
			t.Errorf("x-api-key = %q, want phone-key", request.Header.Get("x-api-key"))
		}
		if err := json.NewDecoder(request.Body).Decode(&receivePayload); err != nil {
			t.Errorf("decode receive payload: %v", err)
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"data": map[string]any{"id": "message-1"},
		})
	}))
	defer api.Close()

	instance := newEmulator(api.URL, api.Client())
	handler := instance.controlHandler()

	registration := performJSONRequest(t, handler, http.MethodPut, "/test/gateways/gateway-1", map[string]any{
		"phone_number":  "+18005550199",
		"phone_api_key": "phone-key",
	})
	if registration.Code != http.StatusNoContent {
		t.Fatalf("registration status = %d, want 204: %s", registration.Code, registration.Body.String())
	}

	incoming := performJSONRequest(t, handler, http.MethodPost, "/test/gateways/gateway-1/incoming", map[string]any{
		"contact":   "+18005550100",
		"content":   "hello",
		"encrypted": true,
	})
	if incoming.Code != http.StatusOK {
		t.Fatalf("incoming status = %d, want 200: %s", incoming.Code, incoming.Body.String())
	}

	if receivePayload["to"] != "+18005550199" ||
		receivePayload["from"] != "+18005550100" ||
		receivePayload["content"] != "hello" ||
		receivePayload["encrypted"] != true ||
		receivePayload["sim"] != "SIM1" ||
		receivePayload["timestamp"] == "" {
		t.Fatalf("unexpected receive payload: %#v", receivePayload)
	}

	var incomingResponse struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(incoming.Body).Decode(&incomingResponse); err != nil {
		t.Fatalf("decode incoming response: %v", err)
	}
	if incomingResponse.Data["id"] != "message-1" {
		t.Fatalf("incoming message id = %#v, want message-1", incomingResponse.Data["id"])
	}

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", health.Code)
	}

	records := httptest.NewRecorder()
	handler.ServeHTTP(records, httptest.NewRequest(http.MethodGet, "/test/gateways/gateway-1/notifications", nil))
	if records.Code != http.StatusOK {
		t.Fatalf("records status = %d, want 200", records.Code)
	}
}

func TestControlHandlerFiltersNotificationRecordsByMessageID(t *testing.T) {
	t.Parallel()

	instance := newEmulator("http://api.example", http.DefaultClient)
	instance.registerGateway("gateway-1", gatewayRegistration{
		PhoneNumber: "+18005550199",
		PhoneAPIKey: "phone-key",
	})
	instance.beginNotification(
		"notification-1",
		"gateway-1",
		map[string]string{"KEY_MESSAGE_ID": "message-1"},
		"message",
		"message-1",
	)
	instance.beginNotification(
		"notification-2",
		"gateway-1",
		map[string]string{"KEY_MESSAGE_ID": "message-2"},
		"message",
		"message-2",
	)

	response := httptest.NewRecorder()
	instance.controlHandler().ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/test/gateways/gateway-1/notifications?message_id=message-2",
			nil,
		),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("records status = %d, want 200", response.Code)
	}

	var payload struct {
		Data []notificationRecord `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode records: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].MessageID != "message-2" {
		t.Fatalf("filtered records = %#v, want message-2 only", payload.Data)
	}
}

func callbackBody(t *testing.T, data map[string]string) []byte {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"message": map[string]any{
			"token": "https://adapter-emulator:9091/notifications/gateway-1",
			"data":  data,
		},
	})
	if err != nil {
		t.Fatalf("marshal callback: %v", err)
	}
	return body
}

func performJSONRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	target string,
	payload map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
