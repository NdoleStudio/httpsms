package oauth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/mcp/internal/auth"
)

// Bounds and sizes used by the token endpoint.
const (
	// refreshTokenBytes and refreshTokenFamilyIDBytes are the amount of
	// crypto/rand entropy (see newRandomToken) encoded into,
	// respectively, an opaque refresh token and a refresh-token family ID.
	refreshTokenBytes         = 32
	refreshTokenFamilyIDBytes = 16
)

// tokenResponse is the success response body of POST /oauth/token, for
// both the authorization_code and refresh_token grants.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// tokenRequestParams are the POST /oauth/token body parameters that must
// appear at most once; a repeated parameter is a smuggling primitive, not
// a request this server will guess the intent of.
var tokenRequestParams = []string{
	"grant_type",
	"code",
	"code_verifier",
	"client_id",
	"redirect_uri",
	"resource",
	"refresh_token",
	"scope",
}

// HandleToken implements POST /oauth/token: the authorization_code grant
// (exchanging a one-time, PKCE-bound code for tokens) and the
// refresh_token grant (rotating a previously issued refresh token for a
// new access/refresh token pair). The request body must be form-encoded
// and is bounded to maxFormBodyBytes. Every error response is an
// OAuth-compliant JSON body (RFC 6749 Section 5.2) with
// "Cache-Control: no-store".
func (s *Server) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !parseFormRequest(w, r) {
		return
	}

	if repeated := firstRepeatedParam(r.PostForm, tokenRequestParams); repeated != "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "each token request parameter must appear exactly once")
		return
	}

	switch r.PostFormValue("grant_type") {
	case "authorization_code":
		s.handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		s.handleRefreshTokenGrant(w, r)
	case "":
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "grant_type is required")
	default:
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be \"authorization_code\" or \"refresh_token\"")
	}
}

// handleAuthorizationCodeGrant redeems a one-time authorization code for
// an access/refresh token pair. The code, and the client_id, redirect_uri,
// and resource it was bound to at authorization time, and its PKCE
// challenge, must all match exactly; the code is consumed (one-time use)
// before any of those checks run, so even a code rejected for a mismatch
// can never be redeemed again.
func (s *Server) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.PostFormValue("code")
	verifier := r.PostFormValue("code_verifier")
	clientID := r.PostFormValue("client_id")
	redirectURI := r.PostFormValue("redirect_uri")
	resource := r.PostFormValue("resource")

	if code == "" || verifier == "" || clientID == "" || redirectURI == "" || resource == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "code, code_verifier, client_id, redirect_uri, and resource are all required")
		return
	}

	record, err := s.store.ConsumeAuthorizationCode(r.Context(), code)
	if err != nil {
		writeStoreError(w, err, "invalid_grant", "the authorization code is invalid, expired, or already used")
		return
	}

	if record.ClientID != clientID || record.RedirectURI != redirectURI || record.Resource != resource {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "the authorization code does not match the supplied client_id, redirect_uri, or resource")
		return
	}

	if !verifyPKCE(record.CodeChallenge, record.CodeChallengeMethod, verifier) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "code_verifier does not match the authorization request")
		return
	}

	principal := auth.Principal{UserID: record.UserID, Email: record.Email}
	s.issueTokens(w, r.Context(), principal, record.ClientID, record.Scopes, record.Resource, "", "")
}

// handleRefreshTokenGrant rotates refreshToken for a new access/refresh
// token pair. The new refresh token replaces the old one atomically
// (Store.RotateRefreshToken): a replayed old refresh token always fails
// with "invalid_grant", even if a legitimate rotation already consumed it
// moments earlier.
func (s *Server) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.PostFormValue("refresh_token")
	clientID := r.PostFormValue("client_id")

	if refreshToken == "" || clientID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "refresh_token and client_id are required")
		return
	}

	grant, err := s.store.GetRefreshToken(r.Context(), refreshToken)
	if err != nil {
		writeStoreError(w, err, "invalid_grant", "the refresh token is invalid, expired, or already used")
		return
	}

	if grant.ClientID != clientID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "the refresh token was not issued to this client")
		return
	}

	resource := grant.Resource
	if requested := r.PostFormValue("resource"); requested != "" && requested != grant.Resource {
		writeOAuthError(w, http.StatusBadRequest, "invalid_target", "resource does not match the original grant")
		return
	}

	scopes := grant.Scopes
	if rawScope := r.PostFormValue("scope"); rawScope != "" {
		requested := strings.Fields(rawScope)
		if !isSubsetOfScopes(requested, grant.Scopes) {
			writeOAuthError(w, http.StatusBadRequest, "invalid_scope", "requested scope exceeds the originally granted scope")
			return
		}
		scopes = requested
	}

	principal := auth.Principal{UserID: grant.UserID, Email: grant.Email}
	s.issueTokens(w, r.Context(), principal, grant.ClientID, scopes, resource, refreshToken, grant.FamilyID)
}

// issueTokens mints an MCP access token for principal/clientID/scopes and
// either creates (rotateOldToken == "") or atomically rotates
// (rotateOldToken != "") an opaque refresh token, then writes the RFC
// 6749-shaped success response. familyID is reused across rotations of
// the same refresh-token lineage and is freshly generated on first issue.
func (s *Server) issueTokens(w http.ResponseWriter, ctx context.Context, principal auth.Principal, clientID string, scopes []string, resource string, rotateOldToken string, familyID string) {
	accessToken, err := s.keys.SignMCPAccessToken(principal, clientID, scopes, s.config.AccessTokenTTL)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "cannot mint an access token")
		return
	}

	newRefreshToken, err := newRandomToken(refreshTokenBytes)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "cannot issue a refresh token")
		return
	}

	if familyID == "" {
		familyID, err = newRandomToken(refreshTokenFamilyIDBytes)
		if err != nil {
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "cannot issue a refresh token")
			return
		}
	}

	newGrant := RefreshGrant{
		Token:     newRefreshToken,
		UserID:    principal.UserID,
		Email:     principal.Email,
		ClientID:  clientID,
		Scopes:    scopes,
		Resource:  resource,
		FamilyID:  familyID,
		CreatedAt: time.Now().UTC(),
	}

	var storeErr error
	if rotateOldToken == "" {
		storeErr = s.store.PutRefreshToken(ctx, newGrant, s.config.RefreshTokenTTL)
	} else {
		storeErr = s.store.RotateRefreshToken(ctx, rotateOldToken, newGrant, s.config.RefreshTokenTTL)
	}
	if storeErr != nil {
		// Only a lost rotation race (the old token was already consumed)
		// is the client's problem; a Redis or serialization failure is
		// ours and must not be reported as an invalid grant.
		writeStoreError(w, storeErr, "invalid_grant", "the refresh token is invalid, expired, or already used")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.config.AccessTokenTTL.Seconds()),
		RefreshToken: newRefreshToken,
		Scope:        strings.Join(scopes, " "),
	})
}

// verifyPKCE reports whether verifier is the correct RFC 7636 S256 PKCE
// code verifier for challenge. Only the "S256" method is supported; any
// other (or missing) method fails closed.
func verifyPKCE(challenge, method, verifier string) bool {
	if method != "S256" || challenge == "" || verifier == "" {
		return false
	}

	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])

	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// isSubsetOfScopes reports whether every entry of subset is present in
// superset, and subset is non-empty.
func isSubsetOfScopes(subset []string, superset []string) bool {
	if len(subset) == 0 {
		return false
	}

	supersetSet := make(map[string]bool, len(superset))
	for _, scope := range superset {
		supersetSet[scope] = true
	}
	for _, scope := range subset {
		if !supersetSet[scope] {
			return false
		}
	}
	return true
}
