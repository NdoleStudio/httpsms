package validators

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/stretchr/testify/assert"
)

type phoneValidatorStaticHostResolver struct {
	addresses map[string][]netip.Addr
	err       error
}

func (resolver *phoneValidatorStaticHostResolver) LookupNetIP(
	_ context.Context,
	_ string,
	host string,
) ([]netip.Addr, error) {
	if resolver.err != nil {
		return nil, resolver.err
	}
	return resolver.addresses[host], nil
}

func TestPhoneHandlerValidatorAcceptsPublicHTTPSNotificationURL(t *testing.T) {
	validator := newPhoneHandlerValidatorWithAddresses(map[string][]netip.Addr{
		"adapter.example.com": {netip.MustParseAddr("8.8.8.8")},
	})

	errors := validator.ValidateFCMToken(context.Background(), requests.PhoneFCMToken{
		PhoneNumber: "+18005550199",
		FcmToken:    "https://adapter.example.com/notify",
		SIM:         entities.SIM1.String(),
	})

	assert.Empty(t, errors)
}

func TestPhoneHandlerValidatorAcceptsPublicHTTPSNotificationURLOnUpsert(t *testing.T) {
	validator := newPhoneHandlerValidatorWithAddresses(map[string][]netip.Addr{
		"adapter.example.com": {netip.MustParseAddr("8.8.8.8")},
	})

	errors := validator.ValidateUpsert(context.Background(), "", requests.PhoneUpsert{
		PhoneNumber:              "+18005550199",
		FcmToken:                 "https://adapter.example.com/notify",
		SIM:                      entities.SIM1.String(),
		MessageExpirationSeconds: 60,
	})

	assert.Empty(t, errors)
}

func TestPhoneHandlerValidatorRejectsUnsafeNotificationURLs(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		addresses []netip.Addr
	}{
		{
			name:      "insecure HTTP",
			token:     "http://adapter.example.com/notify",
			addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")},
		},
		{
			name:      "loopback resolution",
			token:     "https://adapter.example.com/notify",
			addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")},
		},
		{
			name:      "private resolution",
			token:     "https://adapter.example.com/notify",
			addresses: []netip.Addr{netip.MustParseAddr("10.0.0.5")},
		},
		{
			name:  "mixed public and private resolution",
			token: "https://adapter.example.com/notify",
			addresses: []netip.Addr{
				netip.MustParseAddr("8.8.8.8"),
				netip.MustParseAddr("10.0.0.5"),
			},
		},
		{
			name:  "malformed HTTPS",
			token: "https://%",
		},
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
					validator := newPhoneHandlerValidatorWithAddresses(map[string][]netip.Addr{
						"adapter.example.com": test.addresses,
					})

					validationErrors := validationPath.validate(validator, test.token)

					assert.NotEmpty(t, validationErrors["fcm_token"])
				})
			}
		})
	}
}

func TestPhoneHandlerValidatorAcceptsOpaqueFirebaseNotificationTokenWithoutResolution(t *testing.T) {
	logger := &contactValidatorNoopLogger{}
	validator := NewPhoneHandlerValidator(
		logger,
		telemetry.NewOtelLogger("test", logger),
		nil,
		services.NewNotificationEndpointPolicy(&phoneValidatorStaticHostResolver{
			err: errors.New("resolver must not be called for Firebase tokens"),
		}, nil),
	)

	errors := validator.ValidateFCMToken(context.Background(), requests.PhoneFCMToken{
		PhoneNumber: "+18005550199",
		FcmToken:    "opaque-firebase-registration-token",
		SIM:         entities.SIM1.String(),
	})

	assert.Empty(t, errors)
}

func TestPhoneHandlerValidatorAcceptsNotificationURLWithUserInformation(t *testing.T) {
	validator := newPhoneHandlerValidatorWithAddresses(map[string][]netip.Addr{
		"adapter.example.com": {netip.MustParseAddr("8.8.8.8")},
	})

	errors := validator.ValidateFCMToken(context.Background(), requests.PhoneFCMToken{
		PhoneNumber: "+18005550199",
		FcmToken:    "https://adapter-user:adapter-password@adapter.example.com/notify",
		SIM:         entities.SIM1.String(),
	})

	assert.Empty(t, errors)
}

func newPhoneHandlerValidatorWithAddresses(addresses map[string][]netip.Addr) *PhoneHandlerValidator {
	logger := &contactValidatorNoopLogger{}
	return NewPhoneHandlerValidator(
		logger,
		telemetry.NewOtelLogger("test", logger),
		nil,
		services.NewNotificationEndpointPolicy(&phoneValidatorStaticHostResolver{
			addresses: addresses,
		}, nil),
	)
}
