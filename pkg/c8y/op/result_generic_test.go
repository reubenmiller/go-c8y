package op_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type myDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestResultMap(t *testing.T) {
	r := op.WithHTTPStatus(op.OK(jsonmodels.NewManagedObject([]byte(`{"id":"1","name":"dev1"}`))), http.StatusOK)
	names := r.Map(func(mo jsonmodels.ManagedObject) string { return mo.Name() })
	assert.Equal(t, "dev1", names.Data)
	assert.Equal(t, http.StatusOK, names.HTTPStatus)

	called := false
	failed := op.Failed[jsonmodels.ManagedObject](errors.New("boom"), false).Map(func(jsonmodels.ManagedObject) string {
		called = true
		return ""
	})
	assert.False(t, called, "fn is not called for a failed result")
	assert.EqualError(t, failed.Err, "boom")
}

func TestResultMap_KeepsDryRunRequest(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/inventory/managedObjects", nil)
	dryRun := op.WithRequest(op.Created(jsonmodels.NewManagedObject(nil)), req)
	mapped := dryRun.Map(func(jsonmodels.ManagedObject) json.RawMessage { return json.RawMessage(`{"id":"1"}`) })
	assert.Same(t, req, mapped.Request)
	n := 0
	for range mapped.Documents() {
		n++
	}
	assert.Zero(t, n, "a mapped dry-run result still renders nothing")
}

func TestResultFlatMap(t *testing.T) {
	r := op.OK(1).FlatMap(func(i int) op.Result[string] { return op.OK("x") })
	assert.Equal(t, "x", r.Data)
}

func TestResultDecode(t *testing.T) {
	r := op.OK(jsonmodels.NewManagedObject([]byte(`{"id":"1","name":"dev1"}`)))
	dev, err := r.Decode[myDevice]()
	require.NoError(t, err)
	assert.Equal(t, myDevice{ID: "1", Name: "dev1"}, dev)

	_, err = op.OK(int64(3)).Decode[myDevice]()
	assert.ErrorIs(t, err, op.ErrNoJSONDocument)
}

func TestResultIterAsCollectSeq(t *testing.T) {
	collection := jsondoc.New([]byte(`[{"id":"1","name":"a"},{"id":"2","name":"b"},{"id":"3","name":"c"}]`))
	r := op.OK(collection)

	var got []myDevice
	for d, err := range r.IterAs[myDevice]() {
		require.NoError(t, err)
		got = append(got, d)
	}
	assert.Len(t, got, 3)
	assert.Equal(t, "b", got[1].Name)

	all, err := r.Collect()
	require.NoError(t, err)
	assert.Len(t, all, 3)

	n := 0
	for range r.Seq() {
		n++
	}
	assert.Equal(t, 3, n)
}
