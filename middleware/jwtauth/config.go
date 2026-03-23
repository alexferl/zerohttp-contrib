package jwtauth

import (
	"strconv"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

// Config configures the JWT authentication middleware.
//
// This provides a high-level configuration for the jwx-based JWT middleware.
// For most use cases, you only need to provide:
//   - KeySet: The JWK set containing keys for validation (and optionally signing)
//   - Storage: The storage implementation for token revocation
//
// Example usage:
//
//	cfg := jwtauth.Config{
//	    KeySet:  keySet,  // From jwk.Fetch() or jwk.Import()
//	    Storage: jwtauth.NewMemoryStorage(),
//	}
//	store := jwtauth.NewTokenStore(cfg)
type Config struct {
	// KeySet is the JWK set used for token validation and signing.
	// Required. Use jwk.Fetch() to load from a JWKS endpoint,
	// or jwk.Import() to create from a raw key.
	//
	// Example:
	//   keySet, _ := jwk.Fetch(ctx, "https://auth.example.com/.well-known/jwks.json")
	//   // or
	//   key, _ := jwk.Import([]byte("secret"))
	//   keySet := jwk.NewSet()
	//   keySet.AddKey(key)
	KeySet jwk.Set

	// Storage handles token revocation persistence.
	// Required. Use NewMemoryStorage() for development/testing,
	// or implement the Storage interface for production (Redis, PostgreSQL, etc.).
	Storage Storage

	// Algorithm specifies the JWT signing algorithm.
	// Default: jwa.HS256() (for symmetric keys)
	// Common values: HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512, EdDSA
	Algorithm jwa.SignatureAlgorithm

	// Issuer is the expected issuer (iss claim).
	// If set, tokens without this issuer will be rejected.
	// Optional.
	Issuer string

	// Audience is the expected audience (aud claim).
	// If set, tokens without this audience will be rejected.
	// Optional.
	Audience string

	// RequiredClaims are claims that MUST be present in the token.
	// Validation fails if any are missing.
	// Default: [] (no required claims)
	RequiredClaims []string

	// ValidateIssuer enables issuer validation.
	// Only applies if Issuer is set.
	ValidateIssuer bool

	// ValidateAudience enables audience validation.
	// Only applies if Audience is set.
	ValidateAudience bool

	// ValidateExpiration enables expiration validation.
	// Default: true
	ValidateExpiration bool

	// ValidateNotBefore enables "not before" validation.
	// Default: true
	ValidateNotBefore bool

	// TokenKeyFunc generates the storage key for a token.
	// Default uses "sub:exp" format.
	// Customize this if you want to use jti or another unique identifier.
	//
	// Example:
	//   TokenKeyFunc: func(claims map[string]any) string {
	//       jti, _ := claims["jti"].(string)
	//       return jti
	//   }
	TokenKeyFunc func(claims map[string]any) string

	// KeySelector selects the appropriate key from the KeySet for signing/validation.
	// Default uses the first key in the set (index 0).
	// Customize this for multi-key scenarios (e.g., key rotation with kid).
	//
	// Example:
	//   KeySelector: func(keySet jwk.Set, token jwt.Token) (jwk.Key, error) {
	//       kid, _ := token.Header().KeyID()
	//       return keySet.LookupKeyID(kid)
	//   }
	KeySelector func(keySet jwk.Set, token any) (jwk.Key, error)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Algorithm:          jwa.HS256(),
		ValidateExpiration: true,
		ValidateNotBefore:  true,
		TokenKeyFunc:       defaultTokenKeyFunc,
		KeySelector:        defaultKeySelector,
	}
}

// defaultTokenKeyFunc generates a storage key from claims using "sub:exp" format.
func defaultTokenKeyFunc(claims map[string]any) string {
	sub, _ := claims["sub"].(string)
	exp, _ := claims["exp"].(int64)
	if exp == 0 {
		// Try float64 (JSON numbers are float64 by default)
		if f, ok := claims["exp"].(float64); ok {
			exp = int64(f)
		}
	}
	return sub + ":" + strconv.FormatInt(exp, 10)
}

// defaultKeySelector returns the first key in the set.
func defaultKeySelector(keySet jwk.Set, token any) (jwk.Key, error) {
	if keySet.Len() == 0 {
		return nil, errNoKeys
	}
	key, ok := keySet.Key(0)
	if !ok {
		return nil, errKeyNotFound
	}
	return key, nil
}
