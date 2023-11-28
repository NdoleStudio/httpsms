package requests

import (
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/nyaruka/phonenumbers"
)

type request struct{}

// getLimit gets the take as a string
func (input *request) sanitizeAddress(value string) string {
	value = strings.TrimRight(value, " ")
	if len(value) > 0 && value[0] == ' ' {
		value = strings.Replace(value, " ", "+", 1)
	}

	if !strings.HasPrefix(value, "+") && input.isDigits(value) && len(value) > 9 {
		value = "+" + value
	}

	if number, err := phonenumbers.Parse(value, phonenumbers.UNKNOWN_REGION); err == nil {
		value = phonenumbers.Format(number, phonenumbers.E164)
	}

	return value
}

// sanitizeBool sanitizes a boolean string
func (input *request) sanitizeBool(value string) string {
	value = strings.TrimSpace(value)
	if value == "1" || strings.ToLower(value) == "true" {
		value = "true"
	}

	if value == "0" || strings.ToLower(value) == "false" {
		value = "false"
	}

	return value
}

func (input *request) sanitizeURL(value string) string {
	value = strings.TrimSpace(value)
	website, err := url.Parse(value)
	if err != nil {
		return value
	}
	if website.Scheme == "" {
		return "https://" + value
	}
	return value
}

func (input *request) sanitizeStringPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (input *request) removeStringDuplicates(values []string) []string {
	cache := map[string]struct{}{}
	for _, value := range values {
		cache[value] = struct{}{}
	}

	var result []string
	for key := range cache {
		result = append(result, key)
	}

	return result
}

func (input *request) sanitizeMessageID(value string) string {
	id := strings.Builder{}
	for _, char := range value {
		if char == '.' {
			return id.String()
		}
		id.WriteRune(char)
	}
	return id.String()
}

// getLimit gets the take as a string
func (input *request) getBool(value string) bool {
	if value == "true" {
		return true
	}
	return false
}

// getLimit gets the take as a string
func (input *request) getInt(value string) int {
	val, _ := strconv.Atoi(value)
	return val
}

func (input *request) isDigits(value string) bool {
	for _, c := range value {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}
