package main

import (
	"net/http"
	"sort"
	"strings"
	"sync"
)

type gateway struct {
	PhoneNumber string
	PhoneAPIKey string
}

type notificationRecord struct {
	NotificationID string            `json:"notification_id"`
	GatewayID      string            `json:"gateway_id"`
	Data           map[string]string `json:"data"`
	MessageID      string            `json:"message_id,omitempty"`
	Kind           string            `json:"kind"`
	Attempts       int               `json:"attempts"`
	Processed      bool              `json:"processed"`
	Error          string            `json:"error,omitempty"`
}

type emulator struct {
	apiBaseURL string
	client     *http.Client
	mu         sync.RWMutex
	gateways   map[string]gateway
	records    map[string]*notificationRecord
}

func newEmulator(apiBaseURL string, client *http.Client) *emulator {
	return &emulator{
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
		client:     client,
		gateways:   make(map[string]gateway),
		records:    make(map[string]*notificationRecord),
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

func (instance *emulator) beginNotification(
	notificationID string,
	gatewayID string,
	data map[string]string,
	kind string,
	messageID string,
) (*notificationRecord, bool) {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	if record, ok := instance.records[notificationID]; ok {
		record.Attempts++
		if record.Processed || record.Error == "" {
			return copyNotificationRecord(record), false
		}
		record.Error = ""
		return copyNotificationRecord(record), true
	}

	record := &notificationRecord{
		NotificationID: notificationID,
		GatewayID:      gatewayID,
		Data:           copyStringMap(data),
		MessageID:      messageID,
		Kind:           kind,
		Attempts:       1,
	}
	instance.records[notificationID] = record

	return copyNotificationRecord(record), true
}

func (instance *emulator) markNotificationProcessed(notificationID string) {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	if record, ok := instance.records[notificationID]; ok {
		record.Processed = true
		record.Error = ""
	}
}

func (instance *emulator) markNotificationFailed(notificationID string, err error) {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	if record, ok := instance.records[notificationID]; ok {
		record.Processed = false
		record.Error = err.Error()
	}
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
	sort.Slice(records, func(left int, right int) bool {
		return records[left].NotificationID < records[right].NotificationID
	})

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
