package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	q := NewInventoryQuery().
		AddOrderBy("name").
		AddOrderBy("creationTime").
		AddFilterEqStr("type", "c8y_Software").
		AddFilterEqStr("name", "foo").
		AddFilterEqStr("softwareType", "bar").
		Build()

	assert.Equal(t, "$filter=((type eq 'c8y_Software') and (name eq 'foo') and (softwareType eq 'bar')) $orderby=name,creationTime", q)
}
