package authentication

import (
	"errors"
	"strings"
)

const (
	// LoginTypeOAuth2Internal OAuth2 internal mode
	LoginTypeOAuth2Internal = "OAUTH2_INTERNAL"

	// LoginTypeOAuth2 OAuth2 external provider
	LoginTypeOAuth2 = "OAUTH2"

	// LoginTypeBasic Basic authentication
	LoginTypeBasic = "BASIC"

	// LoginTypeNone no authentication
	LoginTypeNone = "NONE"
)

var (
	ErrInvalidLoginType = errors.New("invalid login type")
)

// Parse the login type and select as default if no value options are found
// It returns the selected method, and if the input was valid or not
func ParseLoginType(v string) (string, error) {
	v = strings.ToUpper(v)
	switch v {
	case LoginTypeBasic:
		return LoginTypeBasic, nil
	case LoginTypeNone:
		return LoginTypeNone, nil
	case LoginTypeOAuth2Internal:
		return LoginTypeOAuth2Internal, nil
	case LoginTypeOAuth2:
		return LoginTypeOAuth2, nil
	default:
		return "", ErrInvalidLoginType
	}
}
