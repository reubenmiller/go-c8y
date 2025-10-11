package pagination

import (
	"testing"

	"github.com/google/go-querystring/query"
	"github.com/stretchr/testify/assert"
)

func TestPageSizeEncoding(t *testing.T) {
	opts := PaginationOptions{
		PageSize: 5,
	}
	q, err := query.Values(opts)
	assert.NoError(t, err)
	encoded := q.Encode()
	assert.Equal(t, encoded, "pageSize=5")
}

func TestPageSizeEncoding_WithZero(t *testing.T) {
	opts := PaginationOptions{
		PageSize: 0,
	}
	q, err := query.Values(opts)
	assert.NoError(t, err)
	encoded := q.Encode()
	assert.Equal(t, encoded, "")
}

func Test_HostTrim(t *testing.T) {
	assert.Equal(t, trimHost("https://hello.com/hello"), "/hello")
}
