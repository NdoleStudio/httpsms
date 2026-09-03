// Package oauth implements the httpSMS MCP service's OAuth 2.1
// authorization server: Redis-backed authorization/token state, Client ID
// Metadata Document (CIMD) resolution, Dynamic Client Registration (DCR)
// compatibility, and OAuth discovery metadata.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis key namespaces. Every key is the fixed prefix followed by the
// hex-encoded SHA-256 hash of the record's public value (transaction ID,
// authorization code, refresh token, DCR client ID, or confirmation
// handle) -- never the raw value itself, so a Redis key listing or a log
// line that leaks a key name never leaks the bearer secret it protects.
const (
	keyPrefixTransaction  = "httpsms:mcp:oauth:transaction:"
	keyPrefixCode         = "httpsms:mcp:oauth:code:"
	keyPrefixRefresh      = "httpsms:mcp:oauth:refresh:"
	keyPrefixClient       = "httpsms:mcp:oauth:client:"
	keyPrefixConfirmation = "httpsms:mcp:confirmation:"
)

// ErrNotFound is returned by every Get/Consume/Rotate method when the
// requested record does not exist, has already expired, or (for
// Consume/Rotate) has already been redeemed exactly once.
var ErrNotFound = errors.New("oauth: not found")

// AuthorizationTransaction records a single in-flight OAuth authorization
// request from the moment its client, redirect URI, scopes, state, and PKCE
// challenge have been validated until the resulting authorization code is
// issued or the transaction expires unused. Unlike codes/tokens/handles it
// is read (not consumed) so it can be re-read across the Firebase-login
// redirect round trip.
type AuthorizationTransaction struct {
	// ID is the random public value this transaction is looked up by. It
	// is used only to derive the record's Redis key and is never persisted
	// in the stored JSON value.
	ID                  string    `json:"-"`
	ClientID            string    `json:"client_id"`
	RedirectURI         string    `json:"redirect_uri"`
	Scopes              []string  `json:"scopes"`
	State               string    `json:"state"`
	CodeChallenge       string    `json:"code_challenge"`
	CodeChallengeMethod string    `json:"code_challenge_method"`
	ResponseType        string    `json:"response_type"`
	CreatedAt           time.Time `json:"created_at"`
}

// AuthorizationCode is a one-time, PKCE-bound authorization code issued
// after the user completes Firebase login and approves the requested
// scopes.
type AuthorizationCode struct {
	// Code is the random public value the client redeems at the token
	// endpoint. It is never persisted in the stored JSON value.
	Code                string    `json:"-"`
	ClientID            string    `json:"client_id"`
	RedirectURI         string    `json:"redirect_uri"`
	Scopes              []string  `json:"scopes"`
	UserID              string    `json:"user_id"`
	Email               string    `json:"email"`
	CodeChallenge       string    `json:"code_challenge"`
	CodeChallengeMethod string    `json:"code_challenge_method"`
	CreatedAt           time.Time `json:"created_at"`
}

// RefreshGrant is a high-entropy opaque refresh token's server-side record,
// bound to the user, client, granted scopes, and token family (rotation
// lineage) that produced it.
type RefreshGrant struct {
	// Token is the random public refresh-token value. It is never
	// persisted in the stored JSON value.
	Token     string    `json:"-"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	ClientID  string    `json:"client_id"`
	Scopes    []string  `json:"scopes"`
	FamilyID  string    `json:"family_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Client is an OAuth client identity resolved either from a Client ID
// Metadata Document (CIMD) fetched at authorization time or from a Dynamic
// Client Registration (DCR) record created through POST /oauth/register.
type Client struct {
	ID                      string   `json:"client_id"`
	Name                    string   `json:"client_name"`
	URI                     string   `json:"client_uri,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// Confirmation is a short-lived, one-time confirmation handle used by the
// legacy (non-MRTR) primary API-key-rotation flow: the first tool call
// returns a handle and a second call must present it before rotation
// proceeds.
type Confirmation struct {
	// Handle is the random public confirmation value. It is never
	// persisted in the stored JSON value.
	Handle    string    `json:"-"`
	UserID    string    `json:"user_id"`
	ClientID  string    `json:"client_id"`
	Operation string    `json:"operation"`
	CreatedAt time.Time `json:"created_at"`
}

// Store is the persistence boundary for every piece of OAuth server-side
// state: in-flight authorization transactions, one-time authorization
// codes, rotating refresh tokens, Dynamic Client Registration records, and
// one-time confirmation handles.
//
// Every Put method requires ttl > 0. Consume and Rotate methods redeem
// their record exactly once, atomically: a second call with the same
// public value returns ErrNotFound.
type Store interface {
	PutAuthorizationTransaction(context.Context, AuthorizationTransaction, time.Duration) error
	GetAuthorizationTransaction(context.Context, string) (AuthorizationTransaction, error)
	PutAuthorizationCode(context.Context, AuthorizationCode, time.Duration) error
	ConsumeAuthorizationCode(context.Context, string) (AuthorizationCode, error)
	PutRefreshToken(context.Context, RefreshGrant, time.Duration) error
	RotateRefreshToken(context.Context, string, RefreshGrant, time.Duration) error
	PutDynamicClient(context.Context, Client, time.Duration) error
	GetDynamicClient(context.Context, string) (Client, error)
	PutConfirmation(context.Context, Confirmation, time.Duration) error
	ConsumeConfirmation(context.Context, string) (Confirmation, error)
}

// rotateRefreshTokenScript atomically deletes the old refresh-token hash
// and creates the new one with a TTL, or does nothing and reports failure
// when the old hash is already gone (already rotated, replayed, or
// expired). A Lua script run through EVAL is the only way to make "check
// the old key exists, delete it, and create the new key" a single
// indivisible server-side operation: MULTI/EXEC alone cannot branch on the
// old key's existence, and WATCH-based optimistic locking would let a
// replayed rotation race the legitimate one instead of failing closed.
var rotateRefreshTokenScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == false then
	return 0
end
redis.call("DEL", KEYS[1])
redis.call("SET", KEYS[2], ARGV[1], "PX", ARGV[2])
return 1
`)

// RedisStore is the Redis-backed implementation of Store.
//
// RedisStore requires a standalone Redis deployment (a client created with
// redis.NewClient), not a Redis Cluster or Ring client. The five key
// namespaces above are an approved, fixed format that must not change, and
// RotateRefreshToken's Lua script touches two keys derived from unrelated
// hashes (the old and new refresh-token hashes) in a single atomic EVAL --
// Redis Cluster requires all keys touched by one command to hash to the
// same hash slot, and this key format gives no such guarantee, so the
// script would fail against a cluster with a CROSSSLOT error. This is an
// intentional constraint of this service, not an oversight: it is not
// safe to point RedisStore at a Redis Cluster or Ring client.
type RedisStore struct {
	client redis.UniversalClient
}

// NewRedisStore returns a Store backed by client. client must be a
// standalone Redis client (redis.NewClient); NewRedisStore panics if given
// a *redis.ClusterClient or *redis.Ring, since RotateRefreshToken's
// cross-slot Lua script cannot run against a cluster (see the RedisStore
// doc comment). The constructor still accepts the redis.UniversalClient
// interface so callers can pass through *redis.Client without an
// unnecessary concrete-type dependency; only these two known-incompatible
// concrete types are rejected.
func NewRedisStore(client redis.UniversalClient) *RedisStore {
	switch client.(type) {
	case *redis.ClusterClient, *redis.Ring:
		panic("oauth: NewRedisStore requires a standalone Redis client (redis.NewClient); a Redis Cluster or Ring client cannot run the cross-slot refresh-token rotation script")
	}
	return &RedisStore{client: client}
}

// PutAuthorizationTransaction implements Store.
func (s *RedisStore) PutAuthorizationTransaction(ctx context.Context, transaction AuthorizationTransaction, ttl time.Duration) error {
	if transaction.ID == "" {
		return errors.New("oauth: authorization transaction ID must not be empty")
	}
	return putRecord(ctx, s.client, keyPrefixTransaction, transaction.ID, transaction, ttl)
}

// GetAuthorizationTransaction implements Store.
func (s *RedisStore) GetAuthorizationTransaction(ctx context.Context, id string) (AuthorizationTransaction, error) {
	var transaction AuthorizationTransaction
	err := getRecord(ctx, s.client, keyPrefixTransaction, id, &transaction)
	transaction.ID = id
	return transaction, err
}

// PutAuthorizationCode implements Store.
func (s *RedisStore) PutAuthorizationCode(ctx context.Context, code AuthorizationCode, ttl time.Duration) error {
	if code.Code == "" {
		return errors.New("oauth: authorization code value must not be empty")
	}
	return putRecord(ctx, s.client, keyPrefixCode, code.Code, code, ttl)
}

// ConsumeAuthorizationCode implements Store. It atomically fetches and
// deletes the record so the same code can never be redeemed twice.
func (s *RedisStore) ConsumeAuthorizationCode(ctx context.Context, code string) (AuthorizationCode, error) {
	var record AuthorizationCode
	err := consumeRecord(ctx, s.client, keyPrefixCode, code, &record)
	record.Code = code
	return record, err
}

// PutRefreshToken implements Store.
func (s *RedisStore) PutRefreshToken(ctx context.Context, grant RefreshGrant, ttl time.Duration) error {
	if grant.Token == "" {
		return errors.New("oauth: refresh token value must not be empty")
	}
	return putRecord(ctx, s.client, keyPrefixRefresh, grant.Token, grant, ttl)
}

// RotateRefreshToken implements Store. It atomically deletes oldToken's
// record and creates newGrant's record with ttl; a second rotation attempt
// against the same oldToken (replay) returns ErrNotFound.
func (s *RedisStore) RotateRefreshToken(ctx context.Context, oldToken string, newGrant RefreshGrant, ttl time.Duration) error {
	if oldToken == "" {
		return errors.New("oauth: old refresh token value must not be empty")
	}
	if newGrant.Token == "" {
		return errors.New("oauth: new refresh token value must not be empty")
	}
	if ttl <= 0 {
		return errors.New("oauth: refresh token TTL must be positive")
	}

	data, err := json.Marshal(newGrant)
	if err != nil {
		return fmt.Errorf("oauth: cannot marshal refresh grant: %w", err)
	}

	oldKey := hashedKey(keyPrefixRefresh, oldToken)
	newKey := hashedKey(keyPrefixRefresh, newGrant.Token)

	result, err := rotateRefreshTokenScript.Run(ctx, s.client, []string{oldKey, newKey}, data, ttl.Milliseconds()).Int64()
	if err != nil {
		return fmt.Errorf("oauth: cannot rotate refresh token: %w", err)
	}
	if result == 0 {
		return ErrNotFound
	}
	return nil
}

// PutDynamicClient implements Store.
func (s *RedisStore) PutDynamicClient(ctx context.Context, client Client, ttl time.Duration) error {
	if client.ID == "" {
		return errors.New("oauth: dynamic client ID must not be empty")
	}
	return putRecord(ctx, s.client, keyPrefixClient, client.ID, client, ttl)
}

// GetDynamicClient implements Store.
func (s *RedisStore) GetDynamicClient(ctx context.Context, id string) (Client, error) {
	var client Client
	err := getRecord(ctx, s.client, keyPrefixClient, id, &client)
	if err == nil {
		client.ID = id
	}
	return client, err
}

// PutConfirmation implements Store.
func (s *RedisStore) PutConfirmation(ctx context.Context, confirmation Confirmation, ttl time.Duration) error {
	if confirmation.Handle == "" {
		return errors.New("oauth: confirmation handle must not be empty")
	}
	return putRecord(ctx, s.client, keyPrefixConfirmation, confirmation.Handle, confirmation, ttl)
}

// ConsumeConfirmation implements Store. It atomically fetches and deletes
// the record so the same handle can never be redeemed twice.
func (s *RedisStore) ConsumeConfirmation(ctx context.Context, handle string) (Confirmation, error) {
	var record Confirmation
	err := consumeRecord(ctx, s.client, keyPrefixConfirmation, handle, &record)
	record.Handle = handle
	return record, err
}

// hashedKey returns the namespaced Redis key for publicValue under prefix:
// prefix followed by the hex-encoded SHA-256 hash of publicValue. The raw
// value is never used as key material.
func hashedKey(prefix, publicValue string) string {
	sum := sha256.Sum256([]byte(publicValue))
	return prefix + hex.EncodeToString(sum[:])
}

// putRecord serializes value as JSON and stores it under prefix's hashed
// key for publicValue, expiring after ttl.
func putRecord(ctx context.Context, client redis.UniversalClient, prefix, publicValue string, value any, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("oauth: record TTL must be positive")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("oauth: cannot marshal record: %w", err)
	}

	if err := client.Set(ctx, hashedKey(prefix, publicValue), data, ttl).Err(); err != nil {
		return fmt.Errorf("oauth: cannot store record: %w", err)
	}
	return nil
}

// getRecord reads and JSON-decodes the record stored under prefix's hashed
// key for publicValue into dest, without deleting it.
func getRecord(ctx context.Context, client redis.UniversalClient, prefix, publicValue string, dest any) error {
	raw, err := client.Get(ctx, hashedKey(prefix, publicValue)).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("oauth: cannot read record: %w", err)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("oauth: cannot decode record: %w", err)
	}
	return nil
}

// consumeRecord atomically reads and deletes (Redis GETDEL) the record
// stored under prefix's hashed key for publicValue into dest, so a second
// call for the same publicValue returns ErrNotFound.
func consumeRecord(ctx context.Context, client redis.UniversalClient, prefix, publicValue string, dest any) error {
	raw, err := client.GetDel(ctx, hashedKey(prefix, publicValue)).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("oauth: cannot consume record: %w", err)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("oauth: cannot decode record: %w", err)
	}
	return nil
}

// newRandomToken returns a cryptographically random, URL-safe public value
// encoding numBytes of entropy, for use as an authorization code, refresh
// token, confirmation handle, transaction ID, or dynamically registered
// client ID. It is never used directly as Redis key material -- callers
// store only its SHA-256 hash (see hashedKey).
func newRandomToken(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("oauth: cannot generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
