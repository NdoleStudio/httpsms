package services

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticHostResolver struct {
	addresses map[string][]netip.Addr
	err       error
}

func (resolver *staticHostResolver) LookupNetIP(_ context.Context, _ string, host string) ([]netip.Addr, error) {
	if resolver.err != nil {
		return nil, resolver.err
	}
	return resolver.addresses[host], nil
}

func TestNotificationEndpointPolicyValidate(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		addresses []netip.Addr
		hasError  bool
	}{
		{name: "public IPv4", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}},
		{name: "public IPv6", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("2606:4700:4700::1111")}},
		{name: "loopback", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}, hasError: true},
		{name: "private", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("10.0.0.5")}, hasError: true},
		{name: "link local", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("169.254.169.254")}, hasError: true},
		{name: "carrier grade NAT", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("100.64.0.1")}, hasError: true},
		{name: "documentation range", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("203.0.113.1")}, hasError: true},
		{name: "unique local IPv6", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("fd00::1")}, hasError: true},
		{name: "mixed public and private", rawURL: "https://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.5")}, hasError: true},
		{name: "embedded credentials", rawURL: "https://username:password@adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}, hasError: true},
		{name: "insecure scheme", rawURL: "http://adapter.example.com/notify", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}, hasError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			endpoint, err := url.Parse(test.rawURL)
			require.NoError(t, err)
			policy := NewNotificationEndpointPolicy(&staticHostResolver{
				addresses: map[string][]netip.Addr{endpoint.Hostname(): test.addresses},
			}, nil)

			addresses, err := policy.Validate(context.Background(), endpoint)

			// URL user information does not affect endpoint network safety.
			hasError := test.hasError && endpoint.User == nil
			if hasError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.addresses, addresses)
		})
	}
}

func TestNotificationEndpointPolicyRejectsNonPublicSpecialPurposeIPv6(t *testing.T) {
	tests := []string{
		"64:ff9b:1::1",
		"100:0:0:1::1",
		"2001:2::1",
		"2001:5::1",
		"2001:10::1",
		"3fff::1",
		"5f00::1",
	}

	for _, address := range tests {
		t.Run(address, func(t *testing.T) {
			endpoint, err := url.Parse("https://adapter.example.com/notify")
			require.NoError(t, err)
			policy := NewNotificationEndpointPolicy(&staticHostResolver{
				addresses: map[string][]netip.Addr{
					endpoint.Hostname(): {netip.MustParseAddr(address)},
				},
			}, nil)

			_, err = policy.Validate(context.Background(), endpoint)

			require.Error(t, err)
		})
	}
}

func TestNotificationEndpointPolicyAllowsGloballyReachableSpecialPurposeIPv6(t *testing.T) {
	tests := []string{
		"64:ff9b::0808:0808",
		"2001::1",
		"2001:1::1",
		"2001:1::2",
		"2001:1::3",
		"2001:3::1",
		"2001:4:112::1",
		"2001:20::1",
		"2001:30::1",
	}

	for _, address := range tests {
		t.Run(address, func(t *testing.T) {
			endpoint, err := url.Parse("https://adapter.example.com/notify")
			require.NoError(t, err)
			policy := NewNotificationEndpointPolicy(&staticHostResolver{
				addresses: map[string][]netip.Addr{
					endpoint.Hostname(): {netip.MustParseAddr(address)},
				},
			}, nil)

			addresses, err := policy.Validate(context.Background(), endpoint)

			require.NoError(t, err)
			assert.Equal(t, []netip.Addr{netip.MustParseAddr(address)}, addresses)
		})
	}
}

func TestNotificationEndpointPolicyRejectsRebindingBeforeDial(t *testing.T) {
	endpoint, err := url.Parse("https://adapter.example.com:9091/notify")
	require.NoError(t, err)

	resolver := &rebindingHostResolver{
		addresses: [][]netip.Addr{
			{netip.MustParseAddr("8.8.8.8")},
			{netip.MustParseAddr("127.0.0.1")},
		},
	}
	policy := NewNotificationEndpointPolicy(resolver, nil)

	_, err = policy.Validate(context.Background(), endpoint)
	require.NoError(t, err)

	dialed := false
	_, err = policy.dialValidated(
		context.Background(),
		"tcp",
		"adapter.example.com:9091",
		func(_ context.Context, _, _ string) (net.Conn, error) {
			dialed = true
			return nil, errors.New("should not dial")
		},
	)

	require.Error(t, err)
	assert.False(t, dialed)
}

func TestNotificationEndpointPolicyAllowsPrivateAddressForExactLocalHost(t *testing.T) {
	endpoint, err := url.Parse("HTTPS://ADAPTER-EMULATOR:9091/notifications/gateway-1")
	require.NoError(t, err)
	policy := NewNotificationEndpointPolicy(&staticHostResolver{
		addresses: map[string][]netip.Addr{
			"adapter-emulator": {netip.MustParseAddr("172.20.0.8")},
		},
	}, []string{"adapter-emulator"})

	addresses, err := policy.Validate(context.Background(), endpoint)

	require.NoError(t, err)
	assert.Equal(t, []netip.Addr{netip.MustParseAddr("172.20.0.8")}, addresses)
}

func TestNotificationEndpointPolicyRejectsNonExactOrLiteralPrivateHosts(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		addresses []netip.Addr
	}{
		{name: "allowlist suffix", rawURL: "https://adapter-emulator.example.com:9091/notify", addresses: []netip.Addr{netip.MustParseAddr("172.20.0.8")}},
		{name: "private IP literal", rawURL: "https://172.20.0.8:9091/notify", addresses: []netip.Addr{netip.MustParseAddr("172.20.0.8")}},
		{name: "non allowlisted host", rawURL: "https://other-emulator:9091/notify", addresses: []netip.Addr{netip.MustParseAddr("172.20.0.8")}},
		{name: "allowlisted loopback", rawURL: "https://adapter-emulator:9091/notify", addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}},
		{name: "allowlisted documentation range", rawURL: "https://adapter-emulator:9091/notify", addresses: []netip.Addr{netip.MustParseAddr("203.0.113.1")}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			endpoint, err := url.Parse(test.rawURL)
			require.NoError(t, err)
			policy := NewNotificationEndpointPolicy(&staticHostResolver{
				addresses: map[string][]netip.Addr{
					endpoint.Hostname(): test.addresses,
				},
			}, []string{"adapter-emulator"})

			_, err = policy.Validate(context.Background(), endpoint)

			require.Error(t, err)
		})
	}
}

type rebindingHostResolver struct {
	addresses [][]netip.Addr
	lookups   int
}

func (resolver *rebindingHostResolver) LookupNetIP(_ context.Context, _ string, _ string) ([]netip.Addr, error) {
	addresses := resolver.addresses[resolver.lookups]
	resolver.lookups++
	return addresses, nil
}
