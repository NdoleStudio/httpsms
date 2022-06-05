package requests

import (
	"strconv"
	"strings"
)

type request struct{}

func (input *request) sanitizeAddress(value string) string {
	value = strings.TrimRight(value, " ")
	if len(value) > 0 && value[0] == ' ' {
		value = strings.Replace(value, " ", "+", 1)
	}
	return value
}

func (input *request) getInt(value string) int {
	val, _ := strconv.Atoi(value)
	return val
}
