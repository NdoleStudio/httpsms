package emails

import "github.com/nyaruka/phonenumbers"

type factory struct{}

func (factory *factory) formatPhoneNumber(number string) string {
	value, _ := phonenumbers.Parse(number, phonenumbers.UNKNOWN_REGION)
	return phonenumbers.Format(value, phonenumbers.E164)
}
