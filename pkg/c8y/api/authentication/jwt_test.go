package authentication

import (
	"encoding/base64"
	"strings"
	"testing"
)

func makeToken(payload string, padded bool) string {
	enc := base64.RawURLEncoding
	if padded {
		enc = base64.URLEncoding
	}
	return enc.EncodeToString([]byte(`{"alg":"RS256"}`)) + "." + enc.EncodeToString([]byte(payload)) + ".sig"
}

func TestParseToken(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		padded  bool
		url     bool // expect '-' or '_' in the encoded payload
	}{
		{"plain", `{"sub":"u","ten":"t123"}`, false, false},
		{"base64url chars", `{"sub":"u??>>","ten":"t123","x":"~~~???>>>"}`, false, true},
		{"needs padding 1", `{"sub":"user","ten":"t1234"}`, false, false},
		{"needs padding 2", `{"sub":"user","ten":"t12345","x":"???"}`, false, true},
		{"padded input", `{"sub":"user","ten":"t1234"}`, true, false},
		{"padded input with url chars", `{"sub":"u??>>","ten":"t999","x":"~~~???>>>"}`, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tok := makeToken(tc.payload, tc.padded)
			payload := strings.Split(tok, ".")[1]
			if tc.url && !strings.ContainsAny(payload, "-_") {
				t.Fatalf("test vector has no base64url chars: %s", payload)
			}
			claim, err := ParseToken(tok)
			if err != nil {
				t.Fatalf("ParseToken: %v", err)
			}
			want := tc.payload[strings.Index(tc.payload, `"ten":"`)+7:]
			want = want[:strings.Index(want, `"`)]
			if claim.Tenant != want {
				t.Fatalf("tenant = %q, want %q", claim.Tenant, want)
			}
		})
	}
}

func TestParseTokenInvalid(t *testing.T) {
	for _, tok := range []string{"a.b", "a.!!!.c"} {
		if _, err := ParseToken(tok); err == nil {
			t.Errorf("expected error for %q", tok)
		}
	}
}
