package authentication

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAuthFromEnvironment(t *testing.T) {
	t.Setenv("C8Y_USER", "t12345/foo")
	t.Setenv("C8Y_USERNAME", "")
	t.Setenv("C8Y_TOKEN", "abcdefg")
	auth := FromEnvironment()
	assert.Equal(t, auth.Tenant, "t12345")
	assert.Equal(t, auth.Username, "foo")
	assert.Equal(t, auth.Token, "abcdefg")
}

var lookupTestKeys = []string{
	"C8Y_BASEURL", "C8Y_URL", "C8Y_HOST", "C8Y_TENANT", "C8Y_TOKEN", "C8Y_PASSWORD",
	"C8Y_USERNAME", "C8Y_USER", "C8Y_CERTIFICATE_KEY_FILE", "C8Y_CERTIFICATE_FILE",
}

func testJWT(t *testing.T, claims string) string {
	t.Helper()
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"none"}`)) + "." + enc([]byte(claims)) + "." + enc([]byte("sig"))
}

// setEnvironment replaces all C8Y_* settings with values (unset keys are
// cleared) and returns the same values as a map.
func setEnvironment(t *testing.T, values map[string]string) map[string]string {
	t.Helper()
	for _, k := range lookupTestKeys {
		t.Setenv(k, values[k])
	}
	return values
}

func TestFromLookupMatchesFromEnvironment(t *testing.T) {
	tokenWithTenant := testJWT(t, `{"ten":"t999","iss":"https://token.example.com","xsrfToken":"x"}`)
	tests := []struct {
		name   string
		values map[string]string
	}{
		{"empty", map[string]string{}},
		{"basic", map[string]string{"C8Y_TENANT": "t1", "C8Y_USERNAME": "alice", "C8Y_PASSWORD": "secret"}},
		{"user alias", map[string]string{"C8Y_USER": "alice", "C8Y_PASSWORD": "secret"}},
		{"username wins over user", map[string]string{"C8Y_USERNAME": "a", "C8Y_USER": "b"}},
		{"empty username falls back to user", map[string]string{"C8Y_USERNAME": "", "C8Y_USER": "b"}},
		{"tenant/user split", map[string]string{"C8Y_USER": "t12345/foo"}},
		{"tenant/user split overrides tenant", map[string]string{"C8Y_TENANT": "t1", "C8Y_USER": "t2/foo"}},
		{"tenant/user with empty tenant", map[string]string{"C8Y_TENANT": "t1", "C8Y_USER": "/foo"}},
		{"tenant/user with empty user", map[string]string{"C8Y_USER": "t2/"}},
		{"whitespace trimmed", map[string]string{"C8Y_TENANT": "  t1 ", "C8Y_PASSWORD": "\tp\n"}},
		{"whitespace only is unset", map[string]string{"C8Y_USERNAME": "   ", "C8Y_USER": "b"}},
		{"opaque token", map[string]string{"C8Y_TOKEN": "abcdefg", "C8Y_TENANT": "t1"}},
		{"jwt token tenant wins", map[string]string{"C8Y_TOKEN": tokenWithTenant, "C8Y_TENANT": "t1"}},
		{"certificate", map[string]string{"C8Y_CERTIFICATE_FILE": "/a.pem", "C8Y_CERTIFICATE_KEY_FILE": "/a.key"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := setEnvironment(t, tt.values)
			assert.Equal(t, FromEnvironment(), FromLookup(LookupFromMap(values)))
		})
	}
}

func TestFromLookupDoesNotReadEnvironment(t *testing.T) {
	setEnvironment(t, map[string]string{"C8Y_TENANT": "from-env", "C8Y_PASSWORD": "from-env"})
	auth := FromLookup(LookupFromMap(map[string]string{"C8Y_TENANT": "from-map"}))
	assert.Equal(t, "from-map", auth.Tenant)
	assert.Empty(t, auth.Password)
}

func TestHostFromLookup(t *testing.T) {
	withIssuer := testJWT(t, `{"iss":"https://token.example.com","xsrfToken":"x"}`)
	noXSRF := testJWT(t, `{"iss":"https://token.example.com"}`)
	noIssuer := testJWT(t, `{"xsrfToken":"x"}`)
	tests := []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"unset", map[string]string{}, ""},
		{"baseurl first", map[string]string{"C8Y_BASEURL": "https://a", "C8Y_URL": "https://b", "C8Y_HOST": "https://c"}, "https://a"},
		{"url before host", map[string]string{"C8Y_URL": "https://b", "C8Y_HOST": "https://c"}, "https://b"},
		{"host", map[string]string{"C8Y_HOST": " https://c "}, "https://c"},
		{"empty baseurl falls through", map[string]string{"C8Y_BASEURL": "", "C8Y_HOST": "https://c"}, "https://c"},
		{"host preferred over issuer", map[string]string{"C8Y_HOST": "https://c", "C8Y_TOKEN": withIssuer}, "https://c"},
		{"issuer used", map[string]string{"C8Y_TOKEN": withIssuer}, "https://token.example.com"},
		{"issuer needs xsrf token", map[string]string{"C8Y_TOKEN": noXSRF}, ""},
		{"issuer needs issuer", map[string]string{"C8Y_TOKEN": noIssuer}, ""},
		{"opaque token", map[string]string{"C8Y_TOKEN": "abcdefg"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := setEnvironment(t, tt.values)
			assert.Equal(t, tt.want, HostFromLookup(LookupFromMap(values)))
			assert.Equal(t, tt.want, HostFromEnvironment())
		})
	}
}

func TestLookupFromMap(t *testing.T) {
	lookup := LookupFromMap(map[string]string{"A": "1", "EMPTY": ""})
	v, ok := lookup("A")
	assert.True(t, ok)
	assert.Equal(t, "1", v)
	v, ok = lookup("EMPTY")
	assert.True(t, ok)
	assert.Empty(t, v)
	_, ok = lookup("MISSING")
	assert.False(t, ok)
	_, ok = LookupFromMap(nil)("A")
	assert.False(t, ok)
}
