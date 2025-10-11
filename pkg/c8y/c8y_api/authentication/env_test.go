package authentication

import (
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
