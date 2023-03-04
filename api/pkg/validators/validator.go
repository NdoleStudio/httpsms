package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/NdoleStudio/httpsms/pkg/events"

	"github.com/nyaruka/phonenumbers"
	"github.com/thedevsaddam/govalidator"
)

type validator struct{}

const (
	phoneNumberRule         = "phoneNumber"
	multiplePhoneNumberRule = "multiplePhoneNumber"
	webhookEventsRule       = "webhookEvents"
)

func init() {
	// custom rules to take fixed length word.
	// e.g: max_word:5 will throw error if the field contains more than 5 words
	govalidator.AddCustomRule(phoneNumberRule, func(field string, rule string, message string, value interface{}) error {
		phoneNumber, ok := value.(string)
		if !ok {
			return fmt.Errorf("the %s field must be a valid E.164 phone number: https://en.wikipedia.org/wiki/E.164", field)
		}

		_, err := phonenumbers.Parse(phoneNumber, phonenumbers.UNKNOWN_REGION)
		if err != nil {
			return fmt.Errorf("the %s field must be a valid E.164 phone number: https://en.wikipedia.org/wiki/E.164", field)
		}

		return nil
	})

	govalidator.AddCustomRule(multiplePhoneNumberRule, func(field string, rule string, message string, value interface{}) error {
		phoneNumbers, ok := value.([]string)
		if !ok {
			return fmt.Errorf("the %s field must be an array of a valid E.164 phone number: https://en.wikipedia.org/wiki/E.164", field)
		}

		for index, number := range phoneNumbers {
			_, err := phonenumbers.Parse(number, phonenumbers.UNKNOWN_REGION)
			if err != nil {
				return fmt.Errorf("the %s value in index [%d] must be a valid E.164 phone number: https://en.wikipedia.org/wiki/E.164", field, index)
			}
		}

		return nil
	})

	govalidator.AddCustomRule(webhookEventsRule, func(field string, rule string, message string, value interface{}) error {
		input, ok := value.([]string)
		if !ok {
			return fmt.Errorf("the %s field must be a string array", field)
		}

		if len(input) == 0 {
			return fmt.Errorf("the %s field is an empty array", field)
		}

		validEvents := map[string]bool{
			events.EventTypeMessagePhoneReceived: true,
		}

		for _, event := range input {
			if _, ok := validEvents[event]; !ok {
				return fmt.Errorf("the %s field has an invalid event with name [%s]", field, event)
			}
		}

		return nil
	})
}

// ValidateUUID that the payload is a UUID
func (validator *validator) ValidateUUID(_ context.Context, ID string, name string) url.Values {
	request := map[string]string{
		name: ID,
	}

	v := govalidator.New(govalidator.Options{
		Data: &request,
		Rules: govalidator.MapData{
			name: []string{
				"required",
				"uuid",
			},
		},
	})

	return v.ValidateStruct()
}
