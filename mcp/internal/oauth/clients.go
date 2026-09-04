package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Sentinel errors returned by ClientResolver.Resolve and the DCR
// registration handler. Callers should use errors.Is against these rather
// than matching error strings.
var (
	// ErrUnsafeClientMetadataURL is returned when a client_id's scheme,
	// host, or resolved address is not a safe target for a server-side
	// fetch (non-HTTPS, or a private/loopback/link-local/otherwise
	// non-public address).
	ErrUnsafeClientMetadataURL = errors.New("oauth: unsafe client metadata document URL")

	// ErrClientMetadataRedirected is returned when fetching a client
	// metadata document received a redirect response; redirects are never
	// followed.
	ErrClientMetadataRedirected = errors.New("oauth: client metadata document fetch was redirected")

	// ErrClientMetadataTooLarge is returned when a client metadata
	// document response exceeds maxClientMetadataBytes.
	ErrClientMetadataTooLarge = errors.New("oauth: client metadata document exceeds size limit")

	// ErrClientMetadataInvalid is returned when a client metadata document
	// is not valid JSON, is missing a required field, or was not served
	// with an application/json Content-Type.
	ErrClientMetadataInvalid = errors.New("oauth: client metadata document is invalid")

	// errTransportCannotBePinned is returned when the *http.Client given
	// to NewClientResolver uses a RoundTripper this package cannot pin a
	// validated IP address into (anything other than *http.Transport or
	// nil). Resolution fails closed instead of silently fetching through
	// an unpinned transport, which would let the transport's own DNS
	// resolution re-resolve the hostname a second, unvalidated time.
	errTransportCannotBePinned = errors.New("oauth: http client transport does not support IP pinning")

	// ErrClientIDMismatch is returned when a CIMD document's own
	// "client_id" field does not exactly equal the URL used to fetch it.
	ErrClientIDMismatch = errors.New("oauth: client metadata document client_id mismatch")

	// ErrUnsupportedGrantType is returned when a client's grant_types is
	// empty, omits "authorization_code", or names an unsupported grant.
	ErrUnsupportedGrantType = errors.New("oauth: unsupported client grant_types")

	// ErrUnsupportedResponseType is returned when a client's
	// response_types is not exactly ["code"].
	ErrUnsupportedResponseType = errors.New("oauth: unsupported client response_types")

	// ErrUnsupportedAuthMethod is returned when a client's
	// token_endpoint_auth_method is not "none".
	ErrUnsupportedAuthMethod = errors.New("oauth: unsupported client token_endpoint_auth_method")

	// ErrInvalidRedirectURI is returned when a client's redirect_uris
	// contains an entry that is not an absolute HTTPS URL, or an absolute
	// HTTP URL with a loopback host.
	ErrInvalidRedirectURI = errors.New("oauth: invalid client redirect_uri")
)

// Limits and timeouts applied to every client metadata document fetch, per
// the design's SSRF hardening requirements.
const (
	maxClientMetadataBytes = 256 * 1024
	cimdFetchTimeout       = 5 * time.Second
	cimdCacheTTL           = 15 * time.Minute
	dynamicClientTTL       = 24 * time.Hour
	dynamicClientIDBytes   = 24

	// cimdCacheMaxEntries bounds how many validated Client ID Metadata Documents are held in
	// memory at once. Any HTTPS URL that serves a valid document is a cacheable client_id,
	// so without a ceiling a stream of distinct authorization requests would grow the cache
	// without limit for the full 15-minute TTL.
	cimdCacheMaxEntries = 1024
)

// supportedGrantTypes are the only grant_types a client may declare.
var supportedGrantTypes = map[string]bool{
	"authorization_code": true,
	"refresh_token":      true,
}

// ClientResolver resolves an OAuth client_id to its Client identity, either
// by fetching and validating a Client ID Metadata Document (CIMD) when
// clientID is an HTTPS URL, or by looking up a Dynamic Client Registration
// (DCR) record in store otherwise.
type ClientResolver struct {
	httpClient *http.Client
	store      Store

	// lookupIP resolves host to its IP addresses for the SSRF safety
	// check. It defaults to a wrapper around net.DefaultResolver and is
	// only ever overridden in tests (same-package field access), letting
	// tests simulate a client_id host resolving to a public address while
	// the actual document fetch is still served locally and deterministically.
	lookupIP func(ctx context.Context, host string) ([]net.IP, error)

	// canPinTransport reports whether httpClient's Transport was
	// successfully wrapped to honor a pinned IP address (see
	// withPinnedIP). fetch refuses to proceed when this is false rather
	// than silently falling back to letting the transport re-resolve the
	// hostname itself.
	canPinTransport bool

	cacheMu sync.Mutex
	cache   map[string]cachedClient
}

// cachedClient is a validated Client document cached for cimdCacheTTL.
type cachedClient struct {
	client    Client
	expiresAt time.Time
}

// NewClientResolver returns a ClientResolver that fetches Client ID
// Metadata Documents using httpClient (with redirects disabled) and falls
// back to store for Dynamic Client Registration lookups.
//
// The document fetch resolves and validates the client_id's host exactly
// once per Resolve call: httpClient's Transport is wrapped so the actual
// TCP connection is pinned to the same IP address that was just validated
// as public, instead of letting the transport's own dialer re-resolve the
// hostname a second time (which a DNS-rebinding attacker could answer with
// a private address after passing validation). The wrap preserves the
// original hostname for the Host header and TLS SNI, since only the raw
// dial address changes -- the request URL is never rewritten.
func NewClientResolver(httpClient *http.Client, store Store) *ClientResolver {
	safeClient := *httpClient
	safeClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	pinnedTransport, canPin := newPinnedTransport(safeClient.Transport)
	safeClient.Transport = pinnedTransport

	return &ClientResolver{
		httpClient:      &safeClient,
		store:           store,
		lookupIP:        defaultLookupIP,
		canPinTransport: canPin,
		cache:           make(map[string]cachedClient),
	}
}

// pinnedIPContextKey is the context key under which fetch stashes the
// validated IP address that the pinned transport returned by
// newPinnedTransport must connect to.
type pinnedIPContextKey struct{}

// withPinnedIP returns a context carrying ip as the address the pinned
// transport must dial for the request built from it, regardless of what
// the request's hostname would otherwise resolve to.
func withPinnedIP(ctx context.Context, ip net.IP) context.Context {
	return context.WithValue(ctx, pinnedIPContextKey{}, ip)
}

// pinnedIPFromContext returns the IP address stashed by withPinnedIP, if
// any.
func pinnedIPFromContext(ctx context.Context) (net.IP, bool) {
	ip, ok := ctx.Value(pinnedIPContextKey{}).(net.IP)
	return ip, ok
}

// newPinnedTransport returns a RoundTripper that behaves exactly like base
// (or a fresh clone of http.DefaultTransport when base is nil), except
// that its dial address's host is replaced with the IP address stashed via
// withPinnedIP on the request's context, when present. The port and every
// other transport behavior (TLS config, proxies, timeouts, and -- in
// tests -- a stubbed DialContext that redirects to a local test server)
// are left untouched, so the request's Host header and TLS ServerName,
// both driven by the unmodified request URL, keep the original hostname.
//
// It reports ok=false when base is a RoundTripper this package cannot
// safely wrap (anything other than *http.Transport or nil); callers must
// then refuse to fetch rather than silently using an unpinned connection.
func newPinnedTransport(base http.RoundTripper) (transport http.RoundTripper, ok bool) {
	var httpTransport *http.Transport
	switch t := base.(type) {
	case *http.Transport:
		httpTransport = t.Clone()
	case nil:
		defaultTransport, isHTTPTransport := http.DefaultTransport.(*http.Transport)
		if !isHTTPTransport {
			return base, false
		}
		httpTransport = defaultTransport.Clone()
	default:
		return base, false
	}

	originalDial := httpTransport.DialContext
	if originalDial == nil {
		originalDial = (&net.Dialer{}).DialContext
	}
	httpTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if ip, hasPin := pinnedIPFromContext(ctx); hasPin {
			if _, port, splitErr := net.SplitHostPort(addr); splitErr == nil {
				addr = net.JoinHostPort(ip.String(), port)
			}
		}
		return originalDial(ctx, network, addr)
	}
	return httpTransport, true
}

// defaultLookupIP resolves host through net.DefaultResolver.
func defaultLookupIP(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	ips := make([]net.IP, len(addrs))
	for i, addr := range addrs {
		ips[i] = addr.IP
	}
	return ips, nil
}

// Resolve returns the Client identified by clientID. When clientID is an
// absolute URL it is resolved as a Client ID Metadata Document; otherwise
// it is looked up as a Dynamic Client Registration record.
func (r *ClientResolver) Resolve(ctx context.Context, clientID string) (Client, error) {
	parsed, err := url.Parse(clientID)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return r.store.GetDynamicClient(ctx, clientID)
	}
	return r.resolveCIMD(ctx, clientID, parsed)
}

// resolveCIMD fetches and validates the Client ID Metadata Document at
// clientID, using a cached result when available.
func (r *ClientResolver) resolveCIMD(ctx context.Context, clientID string, parsed *url.URL) (Client, error) {
	if parsed.Scheme != "https" {
		return Client{}, fmt.Errorf("%w: client metadata document must use https, got %q", ErrUnsafeClientMetadataURL, parsed.Scheme)
	}

	// Resolve and validate the host exactly once. pinnedIP is the single
	// address every subsequent step trusts: validateHostIsPublic already
	// proved it (and, for a DNS name, every other address the name
	// resolved to) is not private/loopback/link-local, and fetch pins the
	// actual connection to this same address so the transport's own
	// dialer never gets a chance to re-resolve the hostname and receive a
	// different (rebound) answer.
	pinnedIP, err := r.validateHostIsPublic(ctx, parsed.Hostname())
	if err != nil {
		return Client{}, err
	}

	if client, ok := r.cached(clientID); ok {
		return client, nil
	}

	body, err := r.fetch(ctx, clientID, pinnedIP)
	if err != nil {
		return Client{}, err
	}

	client, err := parseCIMDDocument(clientID, body)
	if err != nil {
		return Client{}, err
	}

	r.cacheClient(clientID, client)
	return client, nil
}

// validateHostIsPublic returns ErrUnsafeClientMetadataURL when host (a
// literal IP or a DNS name) does not resolve exclusively to public
// addresses. On success it returns the single IP address the caller must
// pin its connection to: host itself when host is already a literal IP, or
// the first of host's resolved addresses (all of which were just proven
// public) when host is a DNS name.
func (r *ClientResolver) validateHostIsPublic(ctx context.Context, host string) (net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return nil, fmt.Errorf("%w: %q is not a public address", ErrUnsafeClientMetadataURL, host)
		}
		return ip, nil
	}

	ips, err := r.lookupIP(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("oauth: cannot resolve client metadata document host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("%w: %q did not resolve to any address", ErrUnsafeClientMetadataURL, host)
	}
	for _, candidate := range ips {
		if !isPublicIP(candidate) {
			return nil, fmt.Errorf("%w: %q resolves to a non-public address", ErrUnsafeClientMetadataURL, host)
		}
	}
	return ips[0], nil
}

// fetch retrieves clientID's document body, rejecting redirects, requiring
// an application/json response, and limiting the response to
// maxClientMetadataBytes. The connection is pinned to pinnedIP: the
// request's URL (and therefore its Host header and TLS ServerName) still
// names clientID's original hostname, but the raw TCP dial address's host
// is replaced with pinnedIP so the transport's dialer cannot resolve the
// hostname a second, unvalidated time.
func (r *ClientResolver) fetch(ctx context.Context, clientID string, pinnedIP net.IP) ([]byte, error) {
	if !r.canPinTransport {
		return nil, fmt.Errorf("%w: cannot safely fetch client metadata document", errTransportCannotBePinned)
	}

	ctx, cancel := context.WithTimeout(ctx, cimdFetchTimeout)
	defer cancel()
	ctx = withPinnedIP(ctx, pinnedIP)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, nil)
	if err != nil {
		return nil, fmt.Errorf("oauth: cannot build client metadata document request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: cannot fetch client metadata document: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, fmt.Errorf("%w: received status %d", ErrClientMetadataRedirected, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: received status %d", ErrClientMetadataInvalid, resp.StatusCode)
	}

	if err := requireJSONContentType(resp.Header.Get("Content-Type")); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxClientMetadataBytes+1))
	if err != nil {
		return nil, fmt.Errorf("oauth: cannot read client metadata document: %w", err)
	}
	if len(body) > maxClientMetadataBytes {
		return nil, ErrClientMetadataTooLarge
	}
	return body, nil
}

// requireJSONContentType returns ErrClientMetadataInvalid when
// contentType's media type is not exactly "application/json". Parameters
// such as "; charset=utf-8" are permitted and ignored.
func requireJSONContentType(contentType string) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("%w: unparseable content-type %q, want \"application/json\"", ErrClientMetadataInvalid, contentType)
	}
	if mediaType != "application/json" {
		return fmt.Errorf("%w: unexpected content-type %q, want \"application/json\"", ErrClientMetadataInvalid, mediaType)
	}
	return nil
}

// cached returns the still-valid cached Client for clientID, if any.
func (r *ClientResolver) cached(clientID string) (Client, bool) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	entry, ok := r.cache[clientID]
	if !ok || time.Now().After(entry.expiresAt) {
		return Client{}, false
	}
	return entry.client, true
}

// cacheClient caches client for clientID for cimdCacheTTL, keeping the cache
// bounded at cimdCacheMaxEntries entries: when a new client_id would exceed
// the ceiling, every already-expired entry is purged first and, if that
// frees nothing, the entry closest to expiring is evicted.
func (r *ClientResolver) cacheClient(clientID string, client Client) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	now := time.Now()

	if _, replacing := r.cache[clientID]; !replacing && len(r.cache) >= cimdCacheMaxEntries {
		r.purgeExpiredLocked(now)
		if len(r.cache) >= cimdCacheMaxEntries {
			r.evictOldestLocked()
		}
	}

	r.cache[clientID] = cachedClient{client: client, expiresAt: now.Add(cimdCacheTTL)}
}

// purgeExpiredLocked deletes every cache entry that has expired by now. The
// caller must hold cacheMu.
func (r *ClientResolver) purgeExpiredLocked(now time.Time) {
	for id, entry := range r.cache {
		if now.After(entry.expiresAt) {
			delete(r.cache, id)
		}
	}
}

// evictOldestLocked deletes the unexpired entry closest to expiring, which
// is the entry cached longest ago since every entry shares the same TTL.
// Ties are broken by client_id so eviction never depends on Go's randomized
// map iteration order. The caller must hold cacheMu.
func (r *ClientResolver) evictOldestLocked() {
	oldestID := ""
	var oldestExpiry time.Time

	for id, entry := range r.cache {
		if oldestID == "" || entry.expiresAt.Before(oldestExpiry) ||
			(entry.expiresAt.Equal(oldestExpiry) && id < oldestID) {
			oldestID, oldestExpiry = id, entry.expiresAt
		}
	}

	if oldestID != "" {
		delete(r.cache, oldestID)
	}
}

// parseCIMDDocument decodes and validates a Client ID Metadata Document
// fetched from clientID, requiring its own "client_id" field to exactly
// equal clientID (the CIMD authentication mechanism).
func parseCIMDDocument(clientID string, body []byte) (Client, error) {
	client, err := parseClientDocument(body)
	if err != nil {
		return Client{}, err
	}
	if client.ID != clientID {
		return Client{}, fmt.Errorf("%w: document client_id %q does not match requested %q", ErrClientIDMismatch, client.ID, clientID)
	}
	return client, nil
}

// parseClientDocument decodes body as a Client and validates every field
// required of both a CIMD document and a Dynamic Client Registration
// request, except the CIMD-only client_id/URL equality check.
func parseClientDocument(body []byte) (Client, error) {
	var client Client
	if err := json.Unmarshal(body, &client); err != nil {
		return Client{}, fmt.Errorf("%w: %v", ErrClientMetadataInvalid, err)
	}

	if client.Name == "" {
		return Client{}, fmt.Errorf("%w: client_name is required", ErrClientMetadataInvalid)
	}

	if len(client.RedirectURIs) == 0 {
		return Client{}, fmt.Errorf("%w: redirect_uris is required", ErrClientMetadataInvalid)
	}
	for _, redirectURI := range client.RedirectURIs {
		if err := validateRedirectURI(redirectURI); err != nil {
			return Client{}, err
		}
	}

	if err := validateGrantTypes(client.GrantTypes); err != nil {
		return Client{}, err
	}
	if err := validateResponseTypes(client.ResponseTypes); err != nil {
		return Client{}, err
	}
	if client.TokenEndpointAuthMethod != "none" {
		return Client{}, fmt.Errorf("%w: must be \"none\", got %q", ErrUnsupportedAuthMethod, client.TokenEndpointAuthMethod)
	}

	return client, nil
}

// validateRedirectURI requires redirectURI to be an absolute HTTPS URL, or
// an absolute HTTP URL whose host is a loopback address (RFC 8252 native
// app exception). No other scheme, and no non-loopback HTTP target, is
// permitted.
func validateRedirectURI(redirectURI string) error {
	parsed, err := url.Parse(redirectURI)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return fmt.Errorf("%w: %q is not an absolute URL", ErrInvalidRedirectURI, redirectURI)
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(parsed.Hostname()) {
			return nil
		}
		return fmt.Errorf("%w: %q uses http with a non-loopback host", ErrInvalidRedirectURI, redirectURI)
	default:
		return fmt.Errorf("%w: %q must use https, or http with a loopback host", ErrInvalidRedirectURI, redirectURI)
	}
}

// isLoopbackHost reports whether host is "localhost" or a literal loopback
// IP address.
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// validateGrantTypes requires a non-empty grant_types naming only
// supported grants and always including "authorization_code".
func validateGrantTypes(grantTypes []string) error {
	if len(grantTypes) == 0 {
		return fmt.Errorf("%w: grant_types is required", ErrUnsupportedGrantType)
	}

	hasAuthorizationCode := false
	for _, grantType := range grantTypes {
		if !supportedGrantTypes[grantType] {
			return fmt.Errorf("%w: %q is not supported", ErrUnsupportedGrantType, grantType)
		}
		if grantType == "authorization_code" {
			hasAuthorizationCode = true
		}
	}
	if !hasAuthorizationCode {
		return fmt.Errorf("%w: must include \"authorization_code\"", ErrUnsupportedGrantType)
	}
	return nil
}

// validateResponseTypes requires response_types to be exactly ["code"].
func validateResponseTypes(responseTypes []string) error {
	if len(responseTypes) != 1 || responseTypes[0] != "code" {
		return fmt.Errorf("%w: must be [\"code\"], got %v", ErrUnsupportedResponseType, responseTypes)
	}
	return nil
}

// isPublicIP reports whether ip is safe to connect to from the
// authorization server: neither loopback, private, link-local, multicast,
// unspecified, nor within the shared/carrier-grade NAT range
// (100.64.0.0/10), which is not covered by net.IP.IsPrivate.
func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return false
	}
	if sharedAddressSpace.Contains(ip) {
		return false
	}
	return true
}

// sharedAddressSpace is the IPv4 carrier-grade NAT range (RFC 6598),
// commonly used for internal cloud/service-mesh networking and therefore
// excluded from "public" alongside RFC 1918 private space.
var sharedAddressSpace = mustParseCIDR("100.64.0.0/10")

func mustParseCIDR(cidr string) *net.IPNet {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return network
}

// registrationRequest is the subset of RFC 7591 Dynamic Client Registration
// request fields this service accepts. Unlike a CIMD document, the client
// never supplies its own client_id: the server assigns one.
type registrationRequest struct {
	Name                    string   `json:"client_name"`
	URI                     string   `json:"client_uri,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// registrationError is the RFC 7591 error response body.
type registrationError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// NewRegistrationHandler returns an http.HandlerFunc implementing
// POST /oauth/register (RFC 7591 Dynamic Client Registration): it validates
// the submitted client metadata with the same rules enforced on a CIMD
// document (redirect URIs, grant/response types, and
// token_endpoint_auth_method=none), assigns a random client_id, stores the
// resulting record in store for 24 hours, and responds 201 Created with the
// stored Client.
func NewRegistrationHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxClientMetadataBytes+1))
		if err != nil {
			writeRegistrationError(w, http.StatusBadRequest, "invalid_client_metadata", "cannot read request body")
			return
		}
		if len(body) > maxClientMetadataBytes {
			writeRegistrationError(w, http.StatusBadRequest, "invalid_client_metadata", "request body exceeds size limit")
			return
		}

		var request registrationRequest
		if err := json.Unmarshal(body, &request); err != nil {
			writeRegistrationError(w, http.StatusBadRequest, "invalid_client_metadata", "request body is not valid JSON")
			return
		}

		clientID, err := newRandomToken(dynamicClientIDBytes)
		if err != nil {
			writeRegistrationError(w, http.StatusInternalServerError, "server_error", "cannot generate client_id")
			return
		}

		candidate := Client{
			ID:                      clientID,
			Name:                    request.Name,
			URI:                     request.URI,
			RedirectURIs:            request.RedirectURIs,
			GrantTypes:              request.GrantTypes,
			ResponseTypes:           request.ResponseTypes,
			TokenEndpointAuthMethod: request.TokenEndpointAuthMethod,
		}

		validated, err := parseClientDocument(mustMarshal(candidate))
		if err != nil {
			writeRegistrationError(w, http.StatusBadRequest, "invalid_client_metadata", err.Error())
			return
		}
		validated.ID = clientID

		if err := store.PutDynamicClient(r.Context(), validated, dynamicClientTTL); err != nil {
			writeRegistrationError(w, http.StatusInternalServerError, "server_error", "cannot store client registration")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		// A registration response carries the newly assigned client_id
		// and the exact metadata this server will honour for it. Like
		// every other authorization-server response, it must never be
		// stored by an intermediary or a browser cache (RFC 6749
		// Section 5.1's no-store requirement applies to the whole
		// authorization-server surface, not just token responses).
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(validated)
	}
}

// mustMarshal marshals v, panicking on failure. It is used only for values
// this package itself constructs (never for attacker-controlled input),
// where a marshal failure would indicate a programmer error.
func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// writeRegistrationError writes an RFC 7591 error response. Like the
// success response, it is never cacheable.
func writeRegistrationError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(registrationError{Error: code, ErrorDescription: description})
}
