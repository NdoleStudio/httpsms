package telemetry

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const (
	redactedLogValue        = "[redacted]"
	omittedRequestBodyValue = "[request body omitted]"
)

// RedactJSONFields returns a log-safe JSON body with matching field values removed.
func RedactJSONFields(body []byte, fields ...string) string {
	if len(body) == 0 {
		return ""
	}

	var value any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return omittedRequestBodyValue
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return omittedRequestBodyValue
	}

	sensitiveFields := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		sensitiveFields[strings.ToLower(field)] = struct{}{}
	}
	if len(sensitiveFields) > 0 {
		if _, ok := value.(map[string]any); !ok {
			return omittedRequestBodyValue
		}
	}
	redactJSONValue(value, sensitiveFields)

	redacted, err := json.Marshal(value)
	if err != nil {
		return omittedRequestBodyValue
	}
	return string(redacted)
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return errors.New("request body contains multiple JSON values")
	}
	return err
}

func redactJSONValue(value any, sensitiveFields map[string]struct{}) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := sensitiveFields[strings.ToLower(key)]; ok {
				typed[key] = redactedLogValue
				continue
			}
			redactJSONValue(child, sensitiveFields)
		}
	case []any:
		for _, child := range typed {
			redactJSONValue(child, sensitiveFields)
		}
	}
}
