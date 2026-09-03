package validators

import (
	"context"
	"net/url"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/stretchr/testify/assert"
)

func TestPhoneHandlerValidatorAcceptsHTTPSNotificationURL(t *testing.T) {
	validator := newPhoneHandlerValidator()

	errors := validator.ValidateFCMToken(context.Background(), requests.PhoneFCMToken{
		PhoneNumber: "+18005550199",
		FcmToken:    "https://adapter.example.com/notify",
		SIM:         entities.SIM1.String(),
	})

	assert.Empty(t, errors)
}

func TestPhoneHandlerValidatorAcceptsHTTPSNotificationURLOnUpsert(t *testing.T) {
	validator := newPhoneHandlerValidator()

	errors := validator.ValidateUpsert(context.Background(), "", requests.PhoneUpsert{
		PhoneNumber:              "+18005550199",
		FcmToken:                 "https://adapter.example.com/notify",
		SIM:                      entities.SIM1.String(),
		MessageExpirationSeconds: 60,
	})

	assert.Empty(t, errors)
}

func TestPhoneHandlerValidatorAcceptsPrivateAndLoopbackNotificationHosts(t *testing.T) {
	validator := newPhoneHandlerValidator()

	for _, token := range []string{
		"https://localhost/notify",
		"https://127.0.0.1/notify",
		"https://10.0.0.5/notify",
	} {
		errors := validator.ValidateFCMToken(context.Background(), requests.PhoneFCMToken{
			PhoneNumber: "+18005550199",
			FcmToken:    token,
			SIM:         entities.SIM1.String(),
		})

		assert.Empty(t, errors, token)
	}
}

func TestPhoneHandlerValidatorRejectsInvalidNotificationURLs(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{name: "insecure HTTP", token: "http://adapter.example.com/notify"},
		{name: "missing host", token: "https:///notify"},
		{name: "malformed HTTPS", token: "https://%"},
	}

	validationPaths := []struct {
		name     string
		validate func(*PhoneHandlerValidator, string) map[string][]string
	}{
		{
			name: "FCM token upsert",
			validate: func(validator *PhoneHandlerValidator, token string) map[string][]string {
				return validator.ValidateFCMToken(context.Background(), requests.PhoneFCMToken{
					PhoneNumber: "+18005550199",
					FcmToken:    token,
					SIM:         entities.SIM1.String(),
				})
			},
		},
		{
			name: "phone upsert",
			validate: func(validator *PhoneHandlerValidator, token string) map[string][]string {
				return validator.ValidateUpsert(context.Background(), "", requests.PhoneUpsert{
					PhoneNumber:              "+18005550199",
					FcmToken:                 token,
					SIM:                      entities.SIM1.String(),
					MessageExpirationSeconds: 60,
				})
			},
		},
	}

	for _, validationPath := range validationPaths {
		t.Run(validationPath.name, func(t *testing.T) {
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					validationErrors := validationPath.validate(newPhoneHandlerValidator(), test.token)

					assert.NotEmpty(t, validationErrors["fcm_token"])
				})
			}
		})
	}
}

func TestPhoneHandlerValidatorAcceptsOpaqueFirebaseNotificationToken(t *testing.T) {
	validator := newPhoneHandlerValidator()

	errors := validator.ValidateFCMToken(context.Background(), requests.PhoneFCMToken{
		PhoneNumber: "+18005550199",
		FcmToken:    "opaque-firebase-registration-token",
		SIM:         entities.SIM1.String(),
	})

	assert.Empty(t, errors)
}

func TestPhoneHandlerValidatorAcceptsNotificationURLWithUserInformation(t *testing.T) {
	validator := newPhoneHandlerValidator()
	endpoint := &url.URL{
		Scheme: "https",
		User:   url.UserPassword("adapter-user", "adapter-password"),
		Host:   "adapter.example.com",
		Path:   "/notify",
	}

	errors := validator.ValidateFCMToken(context.Background(), requests.PhoneFCMToken{
		PhoneNumber: "+18005550199",
		FcmToken:    endpoint.String(),
		SIM:         entities.SIM1.String(),
	})

	assert.Empty(t, errors)
}

func newPhoneHandlerValidator() *PhoneHandlerValidator {
	logger := &contactValidatorNoopLogger{}
	return NewPhoneHandlerValidator(
		logger,
		telemetry.NewOtelLogger("test", logger),
		nil,
	)
}
