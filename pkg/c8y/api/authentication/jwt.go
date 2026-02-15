package authentication

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Token Claim
//
// Notes:
// Issuer - Cumulocity URL if the token was issued by Cumulocity
type TokenClaim struct {
	User      string `json:"sub,omitempty"`
	Tenant    string `json:"ten,omitempty"`
	XSRFToken string `json:"xsrfToken,omitempty"`
	TGA       bool   `json:"tfa,omitempty"`
	jwt.RegisteredClaims
}

// Parse a JWT claims
func ParseToken(tokenString string) (*TokenClaim, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token. expected 3 fields")
	}
	raw, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	claim := &TokenClaim{}
	err = json.Unmarshal(raw, claim)
	return claim, err
}
