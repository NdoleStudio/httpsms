package requests

import (
	"strconv"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

type request struct{}

// getLimit gets the take as a string
func (input *request) sanitizeAddress(value string) string {
	value = strings.TrimRight(value, " ")
	if len(value) > 0 && value[0] == ' ' {
		value = strings.Replace(value, " ", "+", 1)
	}

	if number, err := phonenumbers.Parse(value, phonenumbers.UNKNOWN_REGION); err == nil {
		value = phonenumbers.Format(number, phonenumbers.E164)
	}

	return value
}

// getLimit gets the take as a string
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
