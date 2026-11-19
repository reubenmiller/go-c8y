package managedobjects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/tidwall/gjson"
)

// ExternalID is an external identifier to bind to a managed object when it is
// created. See WithExternalIDs.
type ExternalID struct {
	Type       string `json:"type"`
	ExternalID string `json:"externalId"`
}

// WithExternalIDs returns a copy of body with the "externalIds" field set, so
// Create and CreateBulk bind the external IDs in the same request:
//
//	body := managedobjects.WithExternalIDs(map[string]any{"name": "dev1"},
//		managedobjects.ExternalID{Type: "c8y_Serial", ExternalID: "SN1"})
//	result := client.ManagedObjects.Create(ctx, body)
//
// The platform rejects the request if an external ID is already bound to
// another managed object. body may be a map or any value that marshals to a
// JSON object; it is not modified.
func WithExternalIDs(body any, ids ...ExternalID) map[string]any {
	out := map[string]any{}
	switch b := body.(type) {
	case nil:
	case map[string]any:
		maps.Copy(out, b)
	default:
		if raw, err := json.Marshal(b); err == nil {
			_ = json.Unmarshal(raw, &out)
		}
	}
	out["externalIds"] = append([]ExternalID(nil), ids...)
	return out
}

// validateBulkItems checks the limits the platform applies to bulk requests,
// so invalid requests fail before anything is sent.
func validateBulkItems(items []any, checkExternalIDs bool) error {
	if len(items) == 0 {
		return errors.New("bulk request needs at least 1 item")
	}
	if len(items) > DefaultBulkBatchSize {
		return fmt.Errorf("bulk request has %d items; the limit is %d items per request (use CreateBulkStream or UpdateBulkStream for more)", len(items), DefaultBulkBatchSize)
	}
	if !checkExternalIDs {
		return nil
	}
	seen := map[string]int{}
	for i, item := range items {
		raw, err := json.Marshal(item)
		if err != nil {
			continue
		}
		for _, id := range gjson.GetBytes(raw, "externalIds").Array() {
			typ, value := id.Get("type").String(), id.Get("externalId").String()
			key := typ + "\x00" + value
			if j, ok := seen[key]; ok {
				return fmt.Errorf("external ID %s/%s is used by items %d and %d; an external ID may occur only once in a bulk request", typ, value, j, i)
			}
			seen[key] = i
		}
	}
	return nil
}

// Recovery lookup settings. Variables so tests can shorten the backoff.
var (
	recoverLookupAttempts = 5
	recoverLookupBackoff  = 500 * time.Millisecond
)

// RecoverCreateBulk completes a bulk create that failed after part of it was
// written (err.Applied == core.BatchAppliedPartial), without creating
// duplicates. items is the slice that was passed to the failed request.
//
// It follows the platform's documented procedure:
//  1. find the created managed objects by the marker fragment (err.Marker),
//     retrying the lookup but never the original bulk request;
//  2. create only the positions the lookup did not return;
//  3. remove the marker fragment from the objects that carry it.
//
// It returns all managed objects in the order of items. If only the marker
// removal fails, it returns all objects together with an error; the marker is
// harmless and can be removed later.
//
// Its signature matches pipeline.BatchRecoverer.
func (s *Service) RecoverCreateBulk(ctx context.Context, items []any, err *core.BatchError) ([]jsonmodels.ManagedObject, error) {
	if err == nil || err.Applied != core.BatchAppliedPartial || err.Marker == "" {
		return nil, errors.New("recover bulk create: the error does not report a partial write (no marker)")
	}

	found, lookupErr := s.findBatchMarker(ctx, err.Marker, len(items))
	if lookupErr != nil {
		return nil, fmt.Errorf("recover bulk create: find created managed objects: %w", lookupErr)
	}

	results := make([]jsonmodels.ManagedObject, len(items))
	missing := make([]int, 0, len(items))
	for i := range items {
		if mo, ok := found[i]; ok {
			results[i] = mo
		} else {
			missing = append(missing, i)
		}
	}

	if len(missing) > 0 {
		toCreate := make([]any, 0, len(missing))
		for _, i := range missing {
			toCreate = append(toCreate, items[i])
		}
		created := s.CreateBulk(ctx, toCreate)
		if created.Err != nil {
			return nil, fmt.Errorf("recover bulk create: create the %d missing managed objects: %w", len(missing), created.Err)
		}
		pos := 0
		for doc := range created.Data.Iter() {
			if pos >= len(missing) {
				break
			}
			results[missing[pos]] = jsonmodels.NewManagedObject(doc.Bytes())
			pos++
		}
		if pos != len(missing) {
			return nil, fmt.Errorf("recover bulk create: expected %d created managed objects, got %d", len(missing), pos)
		}
	}

	if len(found) > 0 {
		updates := make([]any, 0, len(found))
		for _, mo := range found {
			updates = append(updates, map[string]any{"id": mo.ID(), err.Marker: nil})
		}
		if res := s.UpdateBulk(ctx, updates); res.Err != nil {
			return results, fmt.Errorf("recover bulk create: all managed objects exist, but removing marker %s failed: %w", err.Marker, res.Err)
		}
	}
	return results, nil
}

// findBatchMarker returns the managed objects carrying the marker fragment,
// keyed by their position in the original request.
func (s *Service) findBatchMarker(ctx context.Context, marker string, size int) (map[int]jsonmodels.ManagedObject, error) {
	var lastErr error
	for attempt := range recoverLookupAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * recoverLookupBackoff):
			}
		}
		res := s.List(ctx, ListOptions{
			FragmentType:      marker,
			PaginationOptions: pagination.NewPaginationOptions(DefaultBulkBatchSize),
		})
		if res.Err != nil {
			lastErr = res.Err
			continue
		}
		found := map[int]jsonmodels.ManagedObject{}
		for doc := range res.Data.Iter() {
			idx := gjson.GetBytes(doc.Bytes(), gjson.Escape(marker)+".index")
			if !idx.Exists() {
				continue
			}
			i := int(idx.Int())
			if i >= 0 && i < size {
				found[i] = jsonmodels.NewManagedObject(doc.Bytes())
			}
		}
		return found, nil
	}
	return nil, lastErr
}
