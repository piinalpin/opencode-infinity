// Package access memverifikasi token terpisah untuk model dan admin.
package access

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"opencode-infinity/internal/config"
)

// Tokens menyimpan verifier opaque; nilai token/digest tidak dapat ditelusuri fmt.
type Tokens struct{ gateway, admin func(string) bool }

// GatewayIdentity adalah identitas internal untuk membatasi scope state per token.
type GatewayIdentity struct{ digest [32]byte }

func (GatewayIdentity) String() string   { return "[gateway-identity]" }
func (GatewayIdentity) GoString() string { return "[gateway-identity]" }

type gatewayIdentityContextKey struct{}

func New(c *config.Config) (*Tokens, error) {
	if !c.Validated() {
		return nil, errors.New("access_unvalidated_config")
	}
	return &Tokens{gateway: verifier(c.GatewayToken()), admin: verifier(c.AdminToken())}, nil
}
func verifier(secret config.Secret) func(string) bool {
	expected := sha256.Sum256([]byte(secret.Reveal()))
	return func(candidate string) bool {
		if len(candidate) == 0 || len(candidate) > 4096 {
			return false
		}
		got := sha256.Sum256([]byte(candidate))
		return subtle.ConstantTimeCompare(got[:], expected[:]) == 1
	}
}
func (p *Tokens) Gateway(candidate string) bool {
	return p != nil && p.gateway != nil && p.gateway(candidate)
}
func (p *Tokens) GatewayIdentity(candidate string) (GatewayIdentity, bool) {
	if !p.Gateway(candidate) {
		return GatewayIdentity{}, false
	}
	return GatewayIdentity{digest: sha256.Sum256([]byte(candidate))}, true
}
func (p *Tokens) Admin(candidate string) bool {
	return p != nil && p.admin != nil && p.admin(candidate)
}

func WithGatewayIdentity(ctx context.Context, identity GatewayIdentity) context.Context {
	return context.WithValue(ctx, gatewayIdentityContextKey{}, identity)
}

func GatewayIdentityFromContext(ctx context.Context) (GatewayIdentity, bool) {
	if ctx == nil {
		return GatewayIdentity{}, false
	}
	identity, ok := ctx.Value(gatewayIdentityContextKey{}).(GatewayIdentity)
	return identity, ok && identity != (GatewayIdentity{})
}

// Ready menandai verifier lengkap hasil constructor; zero value tetap tertutup.
func (p *Tokens) Ready() bool { return p != nil && p.gateway != nil && p.admin != nil }
