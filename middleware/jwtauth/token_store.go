package jwtauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/alexferl/zerohttp/middleware/jwtauth"
)

var (
	// errNoKeys is returned when the key set is empty.
	errNoKeys = errors.New("key set contains no keys")

	// errKeyNotFound is returned when a key cannot be found in the set.
	errKeyNotFound = errors.New("key not found in key set")

	// errMissingKeySet is returned when KeySet is not provided.
	errMissingKeySet = errors.New("key set is required")

	// errMissingStorage is returned when Storage is not provided.
	errMissingStorage = errors.New("storage is required")

	// errInvalidIssuer is returned when the issuer claim doesn't match.
	errInvalidIssuer = errors.New("invalid issuer")

	// errInvalidAudience is returned when the audience claim doesn't match.
	errInvalidAudience = errors.New("invalid audience")
)

// TokenStore implements the zerohttp jwtauth.Store interface using
// github.com/lestrrat-go/jwx/v3 for JWT operations.
//
// This implementation supports multiple signing algorithms (HS256, RS256, ES256, EdDSA)
// and provides pluggable storage for token revocation.
//
// Example usage:
//
//	// Create a key set
//
// rawKey := []byte("your-secret-key-at-least-32-bytes-long!")
// key, _ := jwk.Import(rawKey)
// keySet := jwk.NewSet()
// keySet.AddKey(key)
//
// // Create storage (use Redis in production)
// storage := storage.NewMemoryStorage()
//
// // Create the token store
//
//	cfg := jwtauth.Config{
//		    KeySet:  keySet,
//		    Storage: storage,
//		}
//		store := jwtauth.NewTokenStore(cfg)
//
// // Use with zerohttp
//
//	jwtCfg := zconfig.JWTAuthConfig{
//	    TokenStore: store,
//	}
//
// app.Use(middleware.JWTAuth(jwtCfg))
type TokenStore struct {
	config  Config
	adapter *StorageAdapter
}

// NewTokenStore creates a new TokenStore with the given configuration.
//
// Required config fields:
//   - KeySet: A jwk.Set containing at least one key
//   - Storage: A storage.Storage implementation for token revocation
//
// Other fields will use sensible defaults if not provided.
//
// Panic if KeySet is nil or empty, or if Storage is nil.
func NewTokenStore(cfg Config) *TokenStore {
	if cfg.KeySet == nil {
		panic(errMissingKeySet)
	}

	if cfg.KeySet.Len() == 0 {
		panic(errNoKeys)
	}

	if cfg.Storage == nil {
		panic(errMissingStorage)
	}

	// Apply defaults
	if cfg.Algorithm.String() == "" {
		cfg.Algorithm = jwa.HS256()
	}
	if cfg.TokenKeyFunc == nil {
		cfg.TokenKeyFunc = defaultTokenKeyFunc
	}
	if cfg.KeySelector == nil {
		cfg.KeySelector = defaultKeySelector
	}

	return &TokenStore{
		config:  cfg,
		adapter: NewStorageAdapter(cfg.Storage),
	}
}

// Validate parses and validates a JWT token, returning the claims as map[string]any.
//
// This method implements the jwtauth.Store interface for zerohttp.
// It performs the following validations:
//   - Signature verification
//   - Expiration check (if enabled in config)
//   - Not-before check (if enabled in config)
//   - Issuer validation (if configured)
//   - Audience validation (if configured)
//
// The returned claims are normalized to map[string]any for maximum compatibility
// with the zerohttp middleware.
func (s *TokenStore) Validate(_ context.Context, tokenString string) (jwtauth.JWTClaims, error) {
	// Parse the token with signature verification
	// We need to use the concrete jwk.Key type for jwt.ParseString
	jwkKey, err := s.getJWKKey(0)
	if err != nil {
		return nil, err
	}

	parseOptions := []jwt.ParseOption{
		jwt.WithKey(s.config.Algorithm, jwkKey),
	}

	// Add validation options
	if s.config.ValidateExpiration {
		parseOptions = append(parseOptions, jwt.WithValidate(true))
	}

	token, err := jwt.ParseString(tokenString, parseOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Validate issuer if configured
	if s.config.ValidateIssuer && s.config.Issuer != "" {
		if iss, ok := token.Issuer(); !ok || iss != s.config.Issuer {
			return nil, errInvalidIssuer
		}
	}

	// Validate audience if configured
	if s.config.ValidateAudience && s.config.Audience != "" {
		aud, ok := token.Audience()
		if !ok {
			return nil, errInvalidAudience
		}
		found := false
		for _, a := range aud {
			if a == s.config.Audience {
				found = true
				break
			}
		}
		if !found {
			return nil, errInvalidAudience
		}
	}

	// Convert token to map[string]any
	claims := tokenToMap(token)
	return claims, nil
}

// Generate creates a new signed JWT token for the given claims.
//
// This method implements the jwtauth.Store interface for zerohttp.
// It automatically handles:
//   - Setting the token type claim ("type": "refresh" for refresh tokens)
//   - Setting the expiration time based on the TTL
//   - Adding issued-at timestamp
//   - Adding issuer and audience if configured
//
// The claims parameter can be either map[string]any or any type that can be
// converted to claims (via reflection or by implementing a Claims interface).
func (s *TokenStore) Generate(_ context.Context, claims jwtauth.JWTClaims, tokenType jwtauth.TokenType, ttl time.Duration) (string, error) {
	builder := jwt.NewBuilder()

	// Extract map claims and add to builder
	claimMap, err := normalizeClaims(claims)
	if err != nil {
		return "", fmt.Errorf("failed to normalize claims: %w", err)
	}

	// Set standard claims from the map
	for k, v := range claimMap {
		switch k {
		case "sub":
			if s, ok := v.(string); ok {
				builder.Subject(s)
			}
		case "iss":
			if s, ok := v.(string); ok {
				builder.Issuer(s)
			}
		case "aud":
			switch aud := v.(type) {
			case string:
				builder.Audience([]string{aud})
			case []string:
				builder.Audience(aud)
			case []interface{}:
				audiences := make([]string, 0, len(aud))
				for _, a := range aud {
					if s, ok := a.(string); ok {
						audiences = append(audiences, s)
					}
				}
				builder.Audience(audiences)
			}
		case "iat":
			switch t := v.(type) {
			case int64:
				builder.IssuedAt(time.Unix(t, 0))
			case float64:
				builder.IssuedAt(time.Unix(int64(t), 0))
			case time.Time:
				builder.IssuedAt(t)
			}
		case "nbf":
			switch t := v.(type) {
			case int64:
				builder.NotBefore(time.Unix(t, 0))
			case float64:
				builder.NotBefore(time.Unix(int64(t), 0))
			case time.Time:
				builder.NotBefore(t)
			}
		case "exp":
			// exp is handled below with TTL
		case "jti":
			if s, ok := v.(string); ok {
				builder.JwtID(s)
			}
		default:
			builder.Claim(k, v)
		}
	}

	// Set configured issuer if not already set
	if s.config.Issuer != "" {
		if _, ok := claimMap["iss"]; !ok {
			builder.Issuer(s.config.Issuer)
		}
	}

	// Set configured audience if not already set
	if s.config.Audience != "" {
		if _, ok := claimMap["aud"]; !ok {
			builder.Audience([]string{s.config.Audience})
		}
	}

	// Set expiration
	if ttl > 0 {
		builder.Expiration(time.Now().Add(ttl))
	}

	// Set issued at if not already set
	if _, ok := claimMap["iat"]; !ok {
		builder.IssuedAt(time.Now())
	}

	// Add token type for refresh tokens
	if tokenType == jwtauth.RefreshToken {
		builder.Claim("type", jwtauth.TokenTypeRefresh)
	}

	// Build the token
	token, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	// Get the signing key
	jwkKey, err := s.getJWKKey(0)
	if err != nil {
		return "", err
	}

	// Sign the token
	signed, err := jwt.Sign(token, jwt.WithKey(s.config.Algorithm, jwkKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

// Revoke invalidates a refresh token by storing its revocation in the configured storage.
//
// This method implements the config.TokenStore interface for zerohttp.
// It revokes:
//   - The specific token (by sub:jti, sub:sid, or sub:exp)
//   - The entire session (by sid claim, if present)
//
// The claims parameter should be the normalized claims map (map[string]any).
// The storage TTL is set to the remaining time until the token expires.
func (s *TokenStore) Revoke(ctx context.Context, claims map[string]any) error {
	// Revoke by token key
	key := s.config.TokenKeyFunc(claims)
	if key != "" {
		// Calculate TTL based on expiration
		ttl := s.calculateTTL(claims)
		if err := s.adapter.RevokeToken(ctx, key, ttl); err != nil {
			return fmt.Errorf("failed to revoke token: %w", err)
		}
	}

	// Revoke entire session if sid claim exists
	if sid, ok := claims["sid"].(string); ok && sid != "" {
		ttl := s.calculateTTL(claims)
		if err := s.adapter.RevokeSession(ctx, sid, ttl); err != nil {
			return fmt.Errorf("failed to revoke session: %w", err)
		}
	}

	return nil
}

// IsRevoked checks if a refresh token has been revoked.
//
// This method implements the config.TokenStore interface for zerohttp.
// It checks:
//   - If the specific token is revoked (by sub:jti, sub:sid, or sub:exp)
//   - If the entire session is revoked (by sid claim)
//
// Returns true if either the token or its session has been revoked.
func (s *TokenStore) IsRevoked(ctx context.Context, claims map[string]any) (bool, error) {
	// Check if session is revoked
	if sid, ok := claims["sid"].(string); ok && sid != "" {
		revoked, err := s.adapter.IsSessionRevoked(ctx, sid)
		if err != nil {
			return false, fmt.Errorf("failed to check session revocation: %w", err)
		}
		if revoked {
			return true, nil
		}
	}

	// Check if specific token is revoked
	key := s.config.TokenKeyFunc(claims)
	if key != "" {
		revoked, err := s.adapter.IsTokenRevoked(ctx, key)
		if err != nil {
			return false, fmt.Errorf("failed to check token revocation: %w", err)
		}
		if revoked {
			return true, nil
		}
	}

	return false, nil
}

// Close closes the token store and releases resources.
// This method implements the jwtauth.Store interface for zerohttp.
// It closes the underlying storage.
func (s *TokenStore) Close() error {
	return s.adapter.Close()
}

// getJWKKey retrieves the jwk.Key at the specified index.
func (s *TokenStore) getJWKKey(idx int) (jwk.Key, error) {
	key, ok := s.config.KeySet.Key(idx)
	if !ok {
		return nil, errKeyNotFound
	}
	return key, nil
}

// calculateTTL calculates the remaining time until token expiration.
func (s *TokenStore) calculateTTL(claims map[string]any) time.Duration {
	var exp time.Time

	switch v := claims["exp"].(type) {
	case int64:
		exp = time.Unix(v, 0)
	case float64:
		exp = time.Unix(int64(v), 0)
	case time.Time:
		exp = v
	default:
		// No expiration set, return 0
		return 0
	}

	ttl := time.Until(exp)
	if ttl < 0 {
		return 0
	}
	return ttl
}

// tokenToMap converts a jwt.Token to map[string]any.
func tokenToMap(token jwt.Token) map[string]any {
	m := make(map[string]any)

	// Standard claims
	if sub, ok := token.Subject(); ok {
		m["sub"] = sub
	}
	if iss, ok := token.Issuer(); ok {
		m["iss"] = iss
	}
	if aud, ok := token.Audience(); ok {
		m["aud"] = aud
	}
	if exp, ok := token.Expiration(); ok {
		m["exp"] = exp.Unix()
	}
	if iat, ok := token.IssuedAt(); ok {
		m["iat"] = iat.Unix()
	}
	if nbf, ok := token.NotBefore(); ok {
		m["nbf"] = nbf.Unix()
	}
	if jti, ok := token.JwtID(); ok {
		m["jti"] = jti
	}

	// Extract all private claims using Keys() and Get()
	for _, key := range token.Keys() {
		if isStandardClaim(key) {
			continue
		}
		var v any
		if err := token.Get(key, &v); err == nil {
			m[key] = v
		}
	}

	return m
}

// isStandardClaim returns true if the claim key is a standard JWT claim
// that was already extracted above.
func isStandardClaim(key string) bool {
	switch key {
	case "sub", "iss", "aud", "exp", "iat", "nbf", "jti":
		return true
	}
	return false
}

// normalizeClaims converts various claim types to map[string]any.
func normalizeClaims(claims jwtauth.JWTClaims) (map[string]any, error) {
	if claims == nil {
		return make(map[string]any), nil
	}

	switch c := claims.(type) {
	case map[string]any:
		return c, nil
	default:
		// For other types, we can't easily convert without reflection
		// Return an error to indicate unsupported type
		return nil, fmt.Errorf("unsupported claims type: %T", claims)
	}
}
