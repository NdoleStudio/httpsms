package validators

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/cache"
	"github.com/NdoleStudio/httpsms/pkg/events"

	"github.com/nyaruka/phonenumbers"
	"github.com/thedevsaddam/govalidator"
)

type validator struct{}

const (
	phoneNumberRule                = "phoneNumber"
	multiplePhoneNumberRule        = "multiplePhoneNumber"
	contactPhoneNumberRule         = "contactPhoneNumber"
	multipleContactPhoneNumberRule = "multipleContactPhoneNumber"
	multipleAttachmentURLRule      = "multipleAttachmentURL"
	multipleInRule                 = "multipleIn"
	webhookEventsRule              = "webhookEvents"
)

func init() {
	// custom rules to take fixed length word.
	// e.g: max_word:5 will throw error if the field contains more than 5 words
	govalidator.AddCustomRule(phoneNumberRule, func(field string, rule string, message string, value interface{}) error {
		phoneNumber, ok := value.(string)
		if !ok {
			return fmt.Errorf("The %s field must be a valid E.164 phone number in the international format e.g +18005550100", field)
		}

		_, err := phonenumbers.Parse(phoneNumber, phonenumbers.UNKNOWN_REGION)
		if err != nil {
			return fmt.Errorf("The %s field must be a valid E.164 phone number in the international format e.g +18005550100", field)
		}

		return nil
	})

	govalidator.AddCustomRule(multiplePhoneNumberRule, func(field string, rule string, message string, value interface{}) error {
		phoneNumbers, ok := value.([]string)
		if !ok {
			return fmt.Errorf("The %s field must be an array of valid phone numbers", field)
		}

		for index, number := range phoneNumbers {
			_, err := phonenumbers.Parse(number, phonenumbers.UNKNOWN_REGION)
			if err != nil {
				return fmt.Errorf("The %s field in index [%d] must be a valid E.164 phone number in the international format e.g +18005550100", field, index)
			}
		}

		return nil
	})

	// custom rules to take fixed length word.
	// e.g: max_word:5 will throw error if the field contains more than 5 words
	govalidator.AddCustomRule(contactPhoneNumberRule, func(field string, rule string, message string, value interface{}) error {
		phoneNumber, ok := value.(string)
		if !ok {
			return fmt.Errorf("The %s field must contain only digits and must be less than 14 characters", field)
		}

		if match, err := regexp.MatchString("^\\+?[0-9]\\d{1,14}$", phoneNumber); err != nil || !match {
			return fmt.Errorf("The %s field must contain only digits and must be less than 14 characters", field)
		}

		return nil
	})

	govalidator.AddCustomRule(multipleContactPhoneNumberRule, func(field string, rule string, message string, value interface{}) error {
		phoneNumbers, ok := value.([]string)
		if !ok {
			return fmt.Errorf("The %s field must be an array of valid phone numbers", field)
		}

		for index, number := range phoneNumbers {
			if match, err := regexp.MatchString("^\\+?[0-9]\\d{1,14}$", number); err != nil || !match {
				return fmt.Errorf("The %s field in index [%d] must contain only digits and must be less than 14 characters", field, index)
			}
		}

		return nil
	})

	govalidator.AddCustomRule(multipleAttachmentURLRule, func(field string, rule string, message string, value interface{}) error {
		attachments, ok := value.([]string)
		if !ok {
			return fmt.Errorf("The %s field must be an array of valid attachment URLs", field)
		}

		for index, attachment := range attachments {
			u, err := url.ParseRequestURI(attachment)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return fmt.Errorf("The attachment %d with URL [%s] must be a valid URL e.g https://placehold.co/600x400", index, attachment)
			}
		}
		return nil
	})

	govalidator.AddCustomRule(multipleInRule, func(field string, rule string, message string, value interface{}) error {
		values, ok := value.([]string)
		if !ok {
			return fmt.Errorf("the %s field must be a string array", field)
		}

		allowlist := strings.Split(strings.TrimPrefix(rule, multipleInRule+":"), ",")
		contains := func(str string) bool {
			for _, a := range allowlist {
				if a == str {
					return true
				}
			}
			return false
		}

		for index, item := range values {
			if !contains(item) {
				return fmt.Errorf("the %s field in contains an invalid value [%s] at index [%d]", field, item, index)
			}
		}

		return nil
	})

	govalidator.AddCustomRule(webhookEventsRule, func(field string, rule string, message string, value interface{}) error {
		input, ok := value.([]string)
		if !ok {
			return fmt.Errorf("The %s field must be a string array", field)
		}

		if len(input) == 0 {
			return fmt.Errorf("The %s field is an empty array", field)
		}

		validEvents := map[string]bool{
			events.EventTypeMessagePhoneReceived:  true,
			events.EventTypeMessagePhoneSent:      true,
			events.EventTypeMessagePhoneDelivered: true,
			events.EventTypeMessageSendFailed:     true,
			events.EventTypeMessageSendExpired:    true,
			events.EventTypePhoneHeartbeatOnline:  true,
			events.EventTypePhoneHeartbeatOffline: true,
			events.MessageCallMissed:              true,
		}

		for _, event := range input {
			if _, ok := validEvents[event]; !ok {
				return fmt.Errorf("The %s field has an invalid event with name [%s]", field, event)
			}
		}

		return nil
	})
}

// ValidateUUID that the payload is a UUID
func (validator *validator) ValidateUUID(ID string, name string) url.Values {
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

func validateAttachmentURL(ctx context.Context, c cache.Cache, attachmentURL string) error {
	cacheKey := "mms-url-validation:" + attachmentURL

	if cachedVal, err := c.Get(ctx, cacheKey); err == nil {
		if cachedVal == "valid" {
			return nil
		}
		return fmt.Errorf(cachedVal)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(http.MethodHead, attachmentURL, nil)
	if err != nil {
		errMsg := fmt.Sprintf("invalid url format")
		saveToCache(ctx, c, cacheKey, errMsg)
		return fmt.Errorf(errMsg)
	}

	resp, err := client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("could not reach the url")
		saveToCache(ctx, c, cacheKey, errMsg)
		return fmt.Errorf(errMsg)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("url returned an error status code: %d", resp.StatusCode)
		saveToCache(ctx, c, cacheKey, errMsg)
		return fmt.Errorf(errMsg)
	}

	const maxSizeBytes = 1.5 * 1024 * 1024

	if resp.ContentLength > int64(maxSizeBytes) {
		errMsg := fmt.Sprintf("file size (%.2f MB) exceeds the 1.5 MB carrier limit", float64(resp.ContentLength)/(1024*1024))
		saveToCache(ctx, c, cacheKey, errMsg)
		return fmt.Errorf(errMsg)
	}

	saveToCache(ctx, c, cacheKey, "valid")
	return nil
}

func saveToCache(ctx context.Context, c cache.Cache, key string, value string) {
	_ = c.Set(ctx, key, value, 15*time.Minute)
}
