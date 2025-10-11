package authentication

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Token claims
// ------------
//
//	{
//	  "aud": "example.cumulocity.com",
//	  "exp": 1688664540,
//	  "iat": 1687454940,
//	  "iss": "example.cumulocity.com",
//	  "jti": "955544f5-52fe-4f19-b577-3452e37a879e",
//	  "nbf": 1687454940,
//	  "sub": "myuser",
//	  "tci": "955544f5-52fe-4f19-b577-3452e37a879e",
//	  "ten": "t12345",
//	  "tfa": false,
//	  "xsrfToken": "UilS6oa3Z9GQi6e7k1RH"
//	}
//
// Token Claim
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
