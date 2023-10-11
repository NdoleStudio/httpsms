package emails

import (
	"fmt"
	"net/http"

	"github.com/nyaruka/phonenumbers"
)

type factory struct{}

func (factory *factory) formatPhoneNumber(number string) string {
	value, _ := phonenumbers.Parse(number, phonenumbers.UNKNOWN_REGION)
	return phonenumbers.Format(value, phonenumbers.INTERNATIONAL)
}

func (factory *factory) formatHTTPResponseCode(code *int) string {
	responseCode := "-"
	if code != nil {
		responseCode = fmt.Sprintf("%d - %s", *code, http.StatusText(*code))
	}
	return responseCode
}
