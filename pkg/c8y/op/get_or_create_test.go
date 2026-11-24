package op

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type item struct {
	Key   string
	Value string
}

type store map[string]item

func (s store) get(_ context.Context, key string) (item, error) {
	if v, ok := s[key]; ok {
		return v, nil
	}
	return item{}, ErrNotFound
}

func (s store) put(_ context.Context, v item) (item, error) {
	s[v.Key] = v
	return v, nil
}

func key(v item) string { return v.Key }

func Test_GetOrCreate(t *testing.T) {
	s := store{"a": {Key: "a", Value: "old"}}
	fn := GetOrCreate(s.get, s.put, key)

	res, err := fn(context.Background(), item{Key: "a", Value: "new"})
	require.NoError(t, err)
	assert.Equal(t, StatusOK, res.Status)
	assert.Equal(t, "old", res.Data.Value)

	res, err = fn(context.Background(), item{Key: "b", Value: "new"})
	require.NoError(t, err)
	assert.Equal(t, StatusCreated, res.Status)
	assert.Equal(t, "b", res.Meta["key"])
	assert.Contains(t, s, "b")
}

func Test_GetOrCreate_GetError(t *testing.T) {
	boom := errors.New("boom")
	getter := func(context.Context, string) (item, error) { return item{}, boom }
	creator := func(context.Context, item) (item, error) {
		t.Fatal("must not create when the lookup failed")
		return item{}, nil
	}
	res, err := GetOrCreate(getter, creator, key)(context.Background(), item{Key: "a"})
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, StatusFailed, res.Status)
}

func Test_Upsert(t *testing.T) {
	s := store{"a": {Key: "a", Value: "old"}}
	fn := Upsert(s.get, s.put, s.put, key)

	res, err := fn(context.Background(), item{Key: "a", Value: "new"})
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, res.Status)
	assert.Equal(t, "new", s["a"].Value)

	res, err = fn(context.Background(), item{Key: "b", Value: "new"})
	require.NoError(t, err)
	assert.Equal(t, StatusCreated, res.Status)
}

func Test_UpsertWithMerge(t *testing.T) {
	s := store{"a": {Key: "a", Value: "same"}}
	merge := func(existing, desired item) item {
		if desired.Value != "" {
			existing.Value = desired.Value
		}
		return existing
	}
	fn := UpsertWithMerge(s.get, s.put, s.put, key, merge)

	res, err := fn(context.Background(), item{Key: "a", Value: "same"})
	require.NoError(t, err)
	assert.Equal(t, StatusSkipped, res.Status)

	res, err = fn(context.Background(), item{Key: "a", Value: "changed"})
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, res.Status)
	assert.Equal(t, true, res.Meta["changed"])

	res, err = fn(context.Background(), item{Key: "c", Value: "x"})
	require.NoError(t, err)
	assert.Equal(t, StatusCreated, res.Status)
}
