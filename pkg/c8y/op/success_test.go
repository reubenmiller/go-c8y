package op_test

import (
	"errors"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
)

func Test_IsSuccess_ByStatus(t *testing.T) {
	cases := []struct {
		status op.Status
		want   bool
	}{
		{op.StatusOK, true},
		{op.StatusCreated, true},
		{op.StatusUpdated, true},
		{op.StatusNoContent, true},
		{op.StatusSkipped, true},
		{op.StatusDuplicate, false},
		{op.StatusNoMatch, false},
		{op.StatusFailed, false},
		{op.StatusPartial, false},
		{"", false},
	}
	for _, c := range cases {
		r := op.Result[string]{Status: c.status}
		assert.Equal(t, c.want, r.IsSuccess(), "status %q", c.status)
	}
}

func Test_IsSuccess_Constructors(t *testing.T) {
	assert.True(t, op.NoContent("").IsSuccess(), "a successful delete (204) is a success")
	assert.True(t, op.Skipped("", "reason").IsSuccess())
	assert.False(t, op.Duplicate("").IsSuccess())
	assert.False(t, op.NoMatch[string]().IsSuccess())
	assert.False(t, op.Failed[string](errors.New("x"), false).IsSuccess())

	withErr := op.OK("x")
	withErr.Err = errors.New("boom")
	assert.False(t, withErr.IsSuccess(), "an error is never a success")
}
