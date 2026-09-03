package main

import (
	"net/http"
	"strings"
	"sync"
)

type gateway struct {
	PhoneNumber string
	PhoneAPIKey string
}

type notificationRecord struct {
	GatewayID string            `json:"gateway_id"`
	Data      map[string]string `json:"data"`
	MessageID string            `json:"message_id,omitempty"`
	Kind      string            `json:"kind"`
	Processed bool              `json:"processed"`
	Error     string            `json:"error,omitempty"`
}

type emulator struct {
	apiBaseURL string
	client     *http.Client
	mu         sync.RWMutex
	gateways   map[string]gateway
	records    []*notificationRecord
}

func newEmulator(apiBaseURL string, client *http.Client) *emulator {
	return &emulator{
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
		client:     client,
		gateways:   make(map[string]gateway),
	}
}

func (instance *emulator) registerGateway(gatewayID string, registration gatewayRegistration) {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	instance.gateways[gatewayID] = gateway{
		PhoneNumber: registration.PhoneNumber,
		PhoneAPIKey: registration.PhoneAPIKey,
	}
}

func (instance *emulator) loadGateway(gatewayID string) (gateway, bool) {
	instance.mu.RLock()
	defer instance.mu.RUnlock()

	registeredGateway, ok := instance.gateways[gatewayID]
	return registeredGateway, ok
}

func (instance *emulator) recordNotification(
	gatewayID string,
	data map[string]string,
	kind string,
	messageID string,
) *notificationRecord {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	record := &notificationRecord{
		GatewayID: gatewayID,
		Data:      copyStringMap(data),
		MessageID: messageID,
		Kind:      kind,
	}
	instance.records = append(instance.records, record)

	return record
}

func (instance *emulator) markNotificationProcessed(record *notificationRecord) {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	record.Processed = true
	record.Error = ""
}

func (instance *emulator) markNotificationFailed(record *notificationRecord, err error) {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	record.Processed = false
	record.Error = err.Error()
}

func (instance *emulator) listGatewayRecords(gatewayID string) []notificationRecord {
	instance.mu.RLock()
	defer instance.mu.RUnlock()

	records := make([]notificationRecord, 0)
	for _, record := range instance.records {
		if record.GatewayID == gatewayID {
			records = append(records, *copyNotificationRecord(record))
		}
	}

	return records
}

func copyNotificationRecord(record *notificationRecord) *notificationRecord {
	copied := *record
	copied.Data = copyStringMap(record.Data)
	return &copied
}

func copyStringMap(values map[string]string) map[string]string {
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
