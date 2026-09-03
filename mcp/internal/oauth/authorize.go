package oauth

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
)

// Bounds and sizes used by the authorization endpoint and consent flow.
const (
	// authorizationTransactionTTL bounds how long a pending authorization
	// request (created by HandleAuthorize, consumed by
	// HandleFirebaseComplete) survives the browser round trip through
	// Firebase login. It is intentionally longer than AuthorizationCodeTTL:
	// interactive login can take longer than redeeming an already-issued
	// code.
	authorizationTransactionTTL = 10 * time.Minute

	// transactionIDBytes and authorizationCodeBytes are the amount of
	// crypto/rand entropy (see newRandomToken) encoded into, respectively,
	// an authorization transaction ID and a one-time authorization code.
	transactionIDBytes     = 32
	authorizationCodeBytes = 32

	// maxFormBodyBytes bounds every form-encoded request body this package
	// accepts (POST /oauth/firebase/complete and POST /oauth/token). An
	// unbounded ParseForm would otherwise let an unauthenticated client
	// stream an arbitrarily large body into server memory.
	maxFormBodyBytes = 64 << 10 // 64 KiB

	// formMediaType is the only request media type either POST endpoint
	// accepts, per RFC 6749 Section 4.1.3.
	formMediaType = "application/x-www-form-urlencoded"
)

// authorizationRequestParams are the GET /oauth/authorize query parameters
// that must appear at most once. A repeated parameter is rejected outright
// rather than resolved by "first wins" or "last wins", since a server and a
// client (or an intermediary) picking different occurrences is a
// parameter-smuggling primitive.
var authorizationRequestParams = []string{
	"client_id",
	"redirect_uri",
	"response_type",
	"state",
	"code_challenge",
	"code_challenge_method",
	"resource",
	"scope",
}

// firebaseCompleteParams are the POST /oauth/firebase/complete body
// parameters that must appear at most once ("approved_scopes" is
// deliberately excluded: it is legitimately repeated, once per scope).
var firebaseCompleteParams = []string{"transaction_id", "id_token", "denied"}

// codeChallengePattern matches an RFC 7636 S256 code challenge: the
// base64url (no padding) encoding of a SHA-256 digest, i.e. exactly 43
// characters drawn from the base64url alphabet. Any other length or
// character can never match a challenge this server computes, so it is
// rejected at the authorization endpoint rather than failing later.
var codeChallengePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

//go:embed templates/authorize.html
var authorizeTemplateFS embed.FS

// scopeDescriptions maps every OAuth scope this service issues (see
// Scopes) to the human-readable sentence shown on the consent page.
var scopeDescriptions = map[string]string{
	"phones:read":          "View your registered phones and sending numbers",
	"messages:read":        "View your message threads and history",
	"messages:send":        "Send SMS messages on your behalf",
	"phone-api-keys:write": "Create a phone API key",
	"user-api-key:rotate":  "Rotate your primary httpSMS API key",
}

// oauthError is the RFC 6749 Section 5.2 / RFC 8414 JSON error response
// body shape, used by every direct (non-redirect) error response from
// this package's handlers.
type oauthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// ServerConfig configures a Server.
type ServerConfig struct {
	// Issuer is this authorization server's own issuer identifier (e.g.
	// "https://mcp.httpsms.com", matching the "issuer" field this service
	// publishes in its RFC 8414 metadata document). It is echoed back as
	// the "iss" parameter on every authorization response redirect, per
	// RFC 9207, so a client can detect a mix-up attack against another
	// authorization server.
	Issuer string

	// Resource is the exact resource value ("https://mcp.httpsms.com/mcp")
	// every authorization and token request must specify (RFC 8707).
	// Requests naming any other resource, or omitting it, are rejected.
	Resource string

	// FirebaseAPIKey and FirebaseAuthDomain configure the client-side
	// Firebase Authentication SDK embedded in the rendered login/consent
	// page. Neither value is a secret: both are ordinarily public in a
	// browser-side Firebase Web SDK configuration.
	FirebaseAPIKey     string
	FirebaseAuthDomain string

	// AuthorizationCodeTTL, AccessTokenTTL, and RefreshTokenTTL bound the
	// lifetime of, respectively, an issued authorization code, a minted
	// MCP access token, and an issued (or rotated) refresh token.
	AuthorizationCodeTTL time.Duration
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
}

// validate returns an error naming the first missing or invalid field.
func (c ServerConfig) validate() error {
	switch {
	case c.Issuer == "":
		return errors.New("oauth: ServerConfig.Issuer must not be empty")
	case c.Resource == "":
		return errors.New("oauth: ServerConfig.Resource must not be empty")
	case c.FirebaseAPIKey == "":
		return errors.New("oauth: ServerConfig.FirebaseAPIKey must not be empty")
	case c.FirebaseAuthDomain == "":
		return errors.New("oauth: ServerConfig.FirebaseAuthDomain must not be empty")
	case c.AuthorizationCodeTTL <= 0:
		return errors.New("oauth: ServerConfig.AuthorizationCodeTTL must be positive")
	case c.AccessTokenTTL <= 0:
		return errors.New("oauth: ServerConfig.AccessTokenTTL must be positive")
	case c.RefreshTokenTTL <= 0:
		return errors.New("oauth: ServerConfig.RefreshTokenTTL must be positive")
	default:
		return nil
	}
}

// Server implements the httpSMS MCP OAuth 2.1 authorization server's
// interactive endpoints: GET /oauth/authorize, POST
// /oauth/firebase/complete, and POST /oauth/token. It never logs or
// returns, outside the exact responses each endpoint's contract requires,
// any bearer token, authorization code, refresh token, PKCE verifier, or
// Firebase ID token it handles.
type Server struct {
	store     Store
	resolver  *ClientResolver
	keys      *auth.KeySet
	verifier  auth.IdentityVerifier
	config    ServerConfig
	templates *template.Template
}

// NewServer returns a Server backed by store, resolver, keys, and verifier,
// configured by config. It returns an error if any argument is nil or
// config is incomplete, or if the embedded consent-page template fails to
// parse (a build-time invariant, not a runtime condition callers need to
// handle beyond checking the error once at startup).
func NewServer(store Store, resolver *ClientResolver, keys *auth.KeySet, verifier auth.IdentityVerifier, config ServerConfig) (*Server, error) {
	if store == nil {
		return nil, errors.New("oauth: Server requires a Store")
	}
	if resolver == nil {
		return nil, errors.New("oauth: Server requires a ClientResolver")
	}
	if keys == nil {
		return nil, errors.New("oauth: Server requires a KeySet")
	}
	if verifier == nil {
		return nil, errors.New("oauth: Server requires an IdentityVerifier")
	}
	if err := config.validate(); err != nil {
		return nil, err
	}

	templates, err := template.ParseFS(authorizeTemplateFS, "templates/authorize.html")
	if err != nil {
		return nil, fmt.Errorf("oauth: cannot parse authorization templates: %w", err)
	}

	return &Server{
		store:     store,
		resolver:  resolver,
		keys:      keys,
		verifier:  verifier,
		config:    config,
		templates: templates,
	}, nil
}

// HandleAuthorize implements GET /oauth/authorize. It validates the
// client, redirect URI, requested scopes, state, PKCE challenge, and
// resource, then stores a short-lived AuthorizationTransaction and renders
// the Firebase login/consent page.
//
// client_id and redirect_uri are validated first, and only against each
// other (an unresolved client_id, or a redirect_uri not registered for the
// resolved client) responds with a direct 400 rather than a redirect: an
// unvalidated redirect_uri must never be treated as a safe error-reporting
// target. Every failure after that point is reported to the client via
// redirect, carrying the RFC 9207 "iss" parameter.
func (s *Server) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	if repeated := firstRepeatedParam(query, authorizationRequestParams); repeated != "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "each authorization request parameter must appear exactly once")
		return
	}

	clientID := query.Get("client_id")
	if clientID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "client_id is required")
		return
	}

	client, err := s.resolver.Resolve(r.Context(), clientID)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client", "client_id could not be resolved")
		return
	}

	redirectURI := query.Get("redirect_uri")
	if redirectURI == "" || !containsExact(client.RedirectURIs, redirectURI) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "redirect_uri is missing or not registered for this client")
		return
	}

	state := query.Get("state")
	if state == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "state is required")
		return
	}

	if responseType := query.Get("response_type"); responseType != "code" {
		s.redirectError(w, r, redirectURI, state, "unsupported_response_type", "response_type must be \"code\"")
		return
	}

	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")
	if codeChallengeMethod != "S256" || !codeChallengePattern.MatchString(codeChallenge) {
		s.redirectError(w, r, redirectURI, state, "invalid_request", "a S256 code_challenge of 43 base64url characters is required")
		return
	}

	resource := query.Get("resource")
	if resource == "" || resource != s.config.Resource {
		s.redirectError(w, r, redirectURI, state, "invalid_target", "resource must equal the MCP resource URL")
		return
	}

	scopes, err := parseRequestedScopes(query.Get("scope"))
	if err != nil {
		s.redirectError(w, r, redirectURI, state, "invalid_scope", err.Error())
		return
	}

	transactionID, err := newRandomToken(transactionIDBytes)
	if err != nil {
		s.redirectError(w, r, redirectURI, state, "server_error", "cannot start authorization")
		return
	}

	transaction := AuthorizationTransaction{
		ID:                  transactionID,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		Scopes:              scopes,
		State:               state,
		Resource:            resource,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ResponseType:        "code",
		CreatedAt:           time.Now().UTC(),
	}
	if err := s.store.PutAuthorizationTransaction(r.Context(), transaction, authorizationTransactionTTL); err != nil {
		s.redirectError(w, r, redirectURI, state, "server_error", "cannot start authorization")
		return
	}

	s.renderAuthorizePage(w, transaction, client)
}

// HandleFirebaseComplete implements POST /oauth/firebase/complete. It
// verifies the posted Firebase ID token, applies the user's scope
// approval/denial decision, and either redirects back to the client with a
// one-time authorization code or with an "access_denied" error.
//
// The Firebase ID token and approved scopes are read only from the POST
// body (never a query string), matching the requirement that a bearer
// identity token must never appear in a URL (logs, browser history,
// Referer headers). The body must be form-encoded and is bounded to
// maxFormBodyBytes.
//
// The authorization transaction is consumed atomically the moment the
// decision that ends it is made -- an approval whose identity token
// verified, or an explicit denial -- so a captured consent POST can never
// be replayed into a second authorization code. A failed identity
// verification deliberately leaves the transaction intact so the user can
// simply sign in again in the same browser tab.
func (s *Server) HandleFirebaseComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !parseFormRequest(w, r) {
		return
	}

	if repeated := firstRepeatedParam(r.PostForm, firebaseCompleteParams); repeated != "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "each request parameter must appear exactly once")
		return
	}

	transactionID := r.PostFormValue("transaction_id")
	if transactionID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "transaction_id is required")
		return
	}

	transaction, err := s.store.GetAuthorizationTransaction(r.Context(), transactionID)
	if err != nil {
		writeStoreError(w, err, "invalid_request", "authorization transaction not found or expired")
		return
	}

	if r.PostFormValue("denied") != "" {
		if _, err := s.store.ConsumeAuthorizationTransaction(r.Context(), transactionID); err != nil {
			writeStoreError(w, err, "invalid_request", "authorization transaction not found or expired")
			return
		}
		s.redirectError(w, r, transaction.RedirectURI, transaction.State, "access_denied", "the user denied the request")
		return
	}

	idToken := r.PostFormValue("id_token")
	if idToken == "" {
		writeOAuthError(w, http.StatusUnauthorized, "access_denied", "a Firebase ID token is required")
		return
	}

	principal, err := s.verifier.Verify(r.Context(), idToken)
	if err != nil || principal.UserID == "" {
		// The transaction is intentionally *not* consumed here: a failed
		// verification is not a completed authorization decision, so the
		// user may retry. Nothing is issued, so nothing can be replayed.
		writeOAuthError(w, http.StatusUnauthorized, "access_denied", "the identity token could not be verified")
		return
	}

	// The decision is final from here on: consume the transaction
	// atomically so only this completion can ever issue a code for it.
	transaction, err = s.store.ConsumeAuthorizationTransaction(r.Context(), transactionID)
	if err != nil {
		writeStoreError(w, err, "invalid_request", "authorization transaction not found or expired")
		return
	}

	approvedScopes := intersectApprovedScopes(transaction.Scopes, r.PostForm["approved_scopes"])
	if len(approvedScopes) == 0 {
		s.redirectError(w, r, transaction.RedirectURI, transaction.State, "access_denied", "no requested scope was approved")
		return
	}

	code, err := newRandomToken(authorizationCodeBytes)
	if err != nil {
		s.redirectError(w, r, transaction.RedirectURI, transaction.State, "server_error", "cannot issue an authorization code")
		return
	}

	authorizationCode := AuthorizationCode{
		Code:                code,
		ClientID:            transaction.ClientID,
		RedirectURI:         transaction.RedirectURI,
		Scopes:              approvedScopes,
		UserID:              principal.UserID,
		Email:               principal.Email,
		Resource:            transaction.Resource,
		CodeChallenge:       transaction.CodeChallenge,
		CodeChallengeMethod: transaction.CodeChallengeMethod,
		CreatedAt:           time.Now().UTC(),
	}
	if err := s.store.PutAuthorizationCode(r.Context(), authorizationCode, s.config.AuthorizationCodeTTL); err != nil {
		s.redirectError(w, r, transaction.RedirectURI, transaction.State, "server_error", "cannot issue an authorization code")
		return
	}

	target := buildRedirectURL(transaction.RedirectURI, map[string]string{
		"code":  code,
		"state": transaction.State,
		"iss":   s.config.Issuer,
	})
	s.redirect(w, r, target)
}

// renderAuthorizePage writes the Firebase login/consent page for
// transaction and client.
func (s *Server) renderAuthorizePage(w http.ResponseWriter, transaction AuthorizationTransaction, client Client) {
	data := struct {
		FirebaseAPIKey     string
		FirebaseAuthDomain string
		TransactionID      string
		ClientName         string
		Scopes             []struct{ Value, Description string }
	}{
		FirebaseAPIKey:     s.config.FirebaseAPIKey,
		FirebaseAuthDomain: s.config.FirebaseAuthDomain,
		TransactionID:      transaction.ID,
		ClientName:         client.Name,
	}
	for _, scope := range transaction.Scopes {
		description := scopeDescriptions[scope]
		if description == "" {
			description = scope
		}
		data.Scopes = append(data.Scopes, struct{ Value, Description string }{Value: scope, Description: description})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The consent page carries an in-flight authorization transaction and
	// is about to hold a Firebase ID token in the DOM: it must never be
	// cached, framed (clickjacked into an invisible "Allow"), or leak its
	// URL -- which carries the client's redirect URI and state -- through a
	// Referer header.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.WriteHeader(http.StatusOK)
	_ = s.templates.ExecuteTemplate(w, "authorize.html", data)
}

// redirect sends an authorization response (success or error) back to the
// client's redirect URI. Authorization responses carry a one-time code or
// an error plus the client's state, so they must never be cached.
func (s *Server) redirect(w http.ResponseWriter, r *http.Request, target string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	http.Redirect(w, r, target, http.StatusFound)
}

// redirectError redirects to redirectURI with the given OAuth error code
// and human-readable description, plus state (when non-empty) and the RFC
// 9207 "iss" parameter.
func (s *Server) redirectError(w http.ResponseWriter, r *http.Request, redirectURI, state, code, description string) {
	target := buildRedirectURL(redirectURI, map[string]string{
		"error":             code,
		"error_description": description,
		"state":             state,
		"iss":               s.config.Issuer,
	})
	s.redirect(w, r, target)
}

// parseFormRequest enforces the form-encoding contract shared by POST
// /oauth/firebase/complete and POST /oauth/token: the request must declare
// "application/x-www-form-urlencoded" and its body must fit within
// maxFormBodyBytes. It writes the OAuth "invalid_request" error and
// reports false when either bound is violated, so callers can simply
// return.
func parseFormRequest(w http.ResponseWriter, r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, formMediaType) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Content-Type must be application/x-www-form-urlencoded")
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodyBytes)
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("the request body could not be parsed or exceeds the %d byte limit", maxFormBodyBytes))
		return false
	}

	return true
}

// firstRepeatedParam returns the first entry of params that appears more
// than once in values, or "" when none does.
func firstRepeatedParam(values url.Values, params []string) string {
	for _, name := range params {
		if len(values[name]) > 1 {
			return name
		}
	}
	return ""
}

// writeStoreError maps a Store failure onto an OAuth response: a missing,
// expired, or already-consumed record is the requester's problem (400 with
// the caller's error code), while any other failure is an infrastructure
// failure that must not be reported as a client error (500 "server_error").
func writeStoreError(w http.ResponseWriter, err error, code, description string) {
	if errors.Is(err, ErrNotFound) {
		writeOAuthError(w, http.StatusBadRequest, code, description)
		return
	}
	writeOAuthError(w, http.StatusInternalServerError, "server_error", "the request could not be completed")
}

// buildRedirectURL appends params (skipping empty values) to redirectURI's
// query string.
func buildRedirectURL(redirectURI string, params map[string]string) string {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		// redirectURI has already been validated by the caller against a
		// resolved client's registered redirect_uris; this should be
		// unreachable, but fail closed rather than panic.
		return redirectURI
	}

	query := parsed.Query()
	for key, value := range params {
		if value == "" {
			continue
		}
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// containsExact reports whether value is exactly present in list.
func containsExact(list []string, value string) bool {
	for _, candidate := range list {
		if candidate == value {
			return true
		}
	}
	return false
}

// parseRequestedScopes splits raw (an OAuth "scope" parameter) on
// whitespace, validates every entry against the fixed Scopes list, and
// deduplicates the result while preserving the order the client asked in,
// requiring at least one scope.
func parseRequestedScopes(raw string) ([]string, error) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return nil, errors.New("scope is required")
	}

	known := make(map[string]bool, len(Scopes))
	for _, scope := range Scopes {
		known[scope] = true
	}

	seen := make(map[string]bool, len(fields))
	scopes := make([]string, 0, len(fields))
	for _, field := range fields {
		if !known[field] {
			return nil, fmt.Errorf("unsupported scope %q", field)
		}
		if seen[field] {
			continue
		}
		seen[field] = true
		scopes = append(scopes, field)
	}
	return scopes, nil
}

// intersectApprovedScopes returns the entries of approved that were also
// present in requested, deduplicated and in requested's order. This is the
// only place scopes are narrowed during consent: a user can approve fewer
// than the client requested, but the client can never end up with a scope
// it did not request (approved values outside requested are silently
// dropped, not treated as an expansion).
func intersectApprovedScopes(requested []string, approved []string) []string {
	approvedSet := make(map[string]bool, len(approved))
	for _, scope := range approved {
		approvedSet[scope] = true
	}

	var result []string
	for _, scope := range requested {
		if approvedSet[scope] {
			result = append(result, scope)
		}
	}
	return result
}

// writeOAuthError writes an RFC 6749 Section 5.2-shaped JSON error
// response with Cache-Control: no-store, as required of every
// authorization-server error response that might carry sensitive
// information.
func writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(oauthError{Error: code, ErrorDescription: description})
}

// writeJSON writes body as a "200 OK"-or-given-status JSON response with
// Cache-Control: no-store.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
