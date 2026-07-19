package emails

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/nyaruka/phonenumbers"
)

type factory struct{}

func (factory *factory) formatPhoneNumber(number string) string {
	value, _ := phonenumbers.Parse(number, phonenumbers.UNKNOWN_REGION)
	return phonenumbers.Format(value, phonenumbers.INTERNATIONAL)
}

func (factory *factory) formatBool(value bool) string {
	if value == true {
		return "Yes"
	}
	return "No"
}

func (factory *factory) formatQuantity(value uint) string {
	formatted := strconv.FormatUint(uint64(value), 10)
	for index := len(formatted) - 3; index > 0; index -= 3 {
		formatted = formatted[:index] + "," + formatted[index:]
	}

	return formatted
}

func (factory *factory) formatHTTPResponseCode(code *int) string {
	responseCode := "-"
	if code != nil {
		responseCode = fmt.Sprintf("%d - %s", *code, http.StatusText(*code))
	}
	return responseCode
}
