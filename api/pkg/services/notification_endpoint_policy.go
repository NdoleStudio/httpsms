package services

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strings"

	"github.com/NdoleStudio/stacktrace"
)

var blockedNotificationPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("100:0:0:1::/64"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

var globallyReachableNotificationPrefixExceptions = []netip.Prefix{
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2001:1::1/128"),
	netip.MustParsePrefix("2001:1::2/128"),
	netip.MustParsePrefix("2001:1::3/128"),
	netip.MustParsePrefix("2001:3::/32"),
	netip.MustParsePrefix("2001:4:112::/48"),
	netip.MustParsePrefix("2001:20::/28"),
	netip.MustParsePrefix("2001:30::/28"),
}

type HostResolver interface {
	LookupNetIP(ctx context.Context, network string, host string) ([]netip.Addr, error)
}

type NotificationEndpointPolicy struct {
	resolver            HostResolver
	allowedPrivateHosts map[string]struct{}
}

func NewNotificationEndpointPolicy(resolver HostResolver, allowedPrivateHosts []string) *NotificationEndpointPolicy {
	privateHosts := make(map[string]struct{}, len(allowedPrivateHosts))
	for _, host := range allowedPrivateHosts {
		privateHosts[strings.ToLower(host)] = struct{}{}
	}

	return &NotificationEndpointPolicy{
		resolver:            resolver,
		allowedPrivateHosts: privateHosts,
	}
}

func (policy *NotificationEndpointPolicy) Validate(ctx context.Context, endpoint *url.URL) ([]netip.Addr, error) {
	if policy == nil || policy.resolver == nil {
		return nil, newNotificationEndpointPolicyViolation("notification endpoint policy is required")
	}
	if endpoint == nil {
		return nil, newNotificationEndpointPolicyViolation("notification endpoint is required")
	}
	if !strings.EqualFold(endpoint.Scheme, "https") {
		return nil, newNotificationEndpointPolicyViolation("notification endpoint must use HTTPS")
	}
	if endpoint.User != nil {
		return nil, newNotificationEndpointPolicyViolation("notification endpoint must not contain user information")
	}

	host := strings.ToLower(endpoint.Hostname())
	if host == "" {
		return nil, newNotificationEndpointPolicyViolation("notification endpoint must contain a hostname")
	}
	if literal, err := netip.ParseAddr(host); err == nil && !isPublicNotificationAddress(literal) {
		return nil, newNotificationEndpointPolicyViolation("notification endpoint must not use a non-public IP literal")
	}

	addresses, err := policy.resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, newNotificationEndpointResolutionFailure(err)
	}
	if len(addresses) == 0 {
		return nil, newNotificationEndpointResolutionFailure(
			stacktrace.NewError("notification endpoint hostname did not resolve"),
		)
	}

	_, privateHostAllowed := policy.allowedPrivateHosts[host]
	for _, address := range addresses {
		if isPublicNotificationAddress(address) {
			continue
		}
		if privateHostAllowed && address.Unmap().IsPrivate() {
			continue
		}
		return nil, newNotificationEndpointPolicyViolation(
			"notification endpoint hostname resolved to a non-public address",
		)
	}

	return addresses, nil
}

func (policy *NotificationEndpointPolicy) DialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		return policy.dialValidated(ctx, network, address, dialer.DialContext)
	}
}

func (policy *NotificationEndpointPolicy) dialValidated(
	ctx context.Context,
	network string,
	address string,
	dial func(context.Context, string, string) (net.Conn, error),
) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, stacktrace.Propagatef(err, "cannot split notification endpoint address")
	}

	endpoint := &url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)}
	addresses, err := policy.Validate(ctx, endpoint)
	if err != nil {
		return nil, stacktrace.Propagatef(err, "notification endpoint is not public")
	}

	var lastErr error
	for _, resolved := range addresses {
		connection, dialErr := dial(ctx, network, net.JoinHostPort(resolved.String(), port))
		if dialErr == nil {
			return connection, nil
		}
		lastErr = dialErr
	}

	return nil, stacktrace.Propagatef(lastErr, "cannot connect to notification endpoint")
}

func isPublicNotificationAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() {
		return false
	}
	for _, prefix := range globallyReachableNotificationPrefixExceptions {
		if prefix.Contains(address) {
			return true
		}
	}
	for _, prefix := range blockedNotificationPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

type notificationEndpointPolicyViolation struct {
	message string
}

func (violation *notificationEndpointPolicyViolation) Error() string {
	return violation.message
}

func newNotificationEndpointPolicyViolation(message string) error {
	return stacktrace.Propagatef(
		&notificationEndpointPolicyViolation{message: message},
		"notification endpoint policy rejected destination",
	)
}

func isNotificationEndpointPolicyViolation(err error) bool {
	var violation *notificationEndpointPolicyViolation
	return errors.As(err, &violation)
}

type notificationEndpointResolutionFailure struct {
	cause error
}

func (failure *notificationEndpointResolutionFailure) Error() string {
	return failure.cause.Error()
}

func (failure *notificationEndpointResolutionFailure) Unwrap() error {
	return failure.cause
}

func newNotificationEndpointResolutionFailure(cause error) error {
	return stacktrace.Propagatef(
		&notificationEndpointResolutionFailure{cause: cause},
		"cannot resolve notification endpoint hostname",
	)
}
