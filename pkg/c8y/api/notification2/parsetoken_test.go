package notification2

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestServiceParseTokenBase64URL(t *testing.T) {
	payload := `{"sub":"u??>>","topic":"t123/dev/sub","x":"~~~???>>>"}`
	for _, padded := range []bool{false, true} {
		enc := base64.RawURLEncoding
		if padded {
			enc = base64.URLEncoding
		}
		p := enc.EncodeToString([]byte(payload))
		if !strings.ContainsAny(p, "-_") {
			t.Fatalf("vector lacks base64url chars: %s", p)
		}
		claim, err := (&Service{}).ParseToken("h." + p + ".s")
		if err != nil {
			t.Fatalf("padded=%v: %v", padded, err)
		}
		if claim.Tenant() != "t123" {
			t.Fatalf("padded=%v tenant=%q", padded, claim.Tenant())
		}
	}
}
