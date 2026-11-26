package applications

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// RotateBinariesOptions controls which binaries RotateBinaries deletes.
type RotateBinariesOptions struct {
	// Keep is the number of binaries (application versions) to keep. The
	// oldest binaries (by creation time) beyond this number are deleted. Zero
	// or less keeps all binaries.
	Keep int

	// ActiveVersionID is the id of the application's active binary, which is
	// never deleted. When empty, it is read from the application
	// (activeVersionId).
	ActiveVersionID string

	// Protect lists the ids of further binaries which are never deleted.
	// Protected binaries still count towards Keep.
	Protect []string
}

// RotateBinaries deletes the oldest binaries of an application so that at most
// opt.Keep binaries remain. The binaries are ordered by their creation time.
// The active binary and the binaries in opt.Protect are never deleted, but they
// count towards opt.Keep, so more than opt.Keep binaries can remain when the
// protected binaries are among the oldest. The application id may be a plain id
// or a resolver string ("name:appName").
//
// Use it to clean up old binaries without uploading a new one, for example
// after lowering the number of binaries to keep. DeployHosted uses it when
// DeployHostedOptions.KeepBinaries is set.
//
// The result's Meta contains "applicationId", "deletedBinaries" (the ids of
// the deleted binaries) and "binariesToDelete" (the ids selected for deletion,
// including any which failed to be deleted). The Status is StatusSkipped when
// opt.Keep is zero or less, StatusOK when nothing was deleted, StatusUpdated
// when binaries were deleted and StatusPartial when some deletions failed.
// When a deletion fails, the remaining binaries are still deleted and the
// errors are joined in Err.
//
// In dry-run mode the application and its binaries are still read, so
// "binariesToDelete" shows what would be deleted, but nothing is deleted.
func (s *Service) RotateBinaries(ctx context.Context, appID string, opt RotateBinariesOptions) op.Result[core.NoContent] {
	return op.ExecuteOrDefer(ctx, op.WithMeta(op.WithExecutor(op.Result[core.NoContent]{}, func(execCtx context.Context) op.Result[core.NoContent] {
		return s.rotateBinaries(execCtx, appID, opt)
	}), "operation", "rotateBinaries"))
}

func (s *Service) rotateBinaries(ctx context.Context, appID string, opt RotateBinariesOptions) op.Result[core.NoContent] {
	deleted := []string{}
	toDelete := []string{}
	withMeta := func(r op.Result[core.NoContent], id string) op.Result[core.NoContent] {
		r = op.WithMeta(r, "operation", "rotateBinaries")
		if id != "" {
			r = op.WithMeta(r, "applicationId", id)
		}
		return op.WithMeta(op.WithMeta(r, "deletedBinaries", deleted), "binariesToDelete", toDelete)
	}

	if opt.Keep <= 0 {
		return withMeta(op.Skipped(core.NoContent{}, "no limit on the number of binaries to keep"), "")
	}

	// Reads always reach the server, even in dry-run mode, so a dry run shows
	// which binaries would be deleted
	readCtx := ctxhelpers.ResolutionContext(ctx)
	id, err := s.ResolveID(readCtx, appID, nil)
	if err != nil {
		return withMeta(op.Failed[core.NoContent](fmt.Errorf("failed to resolve application. id=%s. %w", appID, err), false), "")
	}

	activeID := opt.ActiveVersionID
	if activeID == "" {
		app := s.Get(readCtx, id)
		if app.Err != nil {
			return withMeta(op.Failed[core.NoContent](fmt.Errorf("failed to get the active application binary. %w", app.Err), false), id)
		}
		activeID = app.Data.Get("activeVersionId").String()
	}

	listResult := s.ListBinaries(readCtx, id)
	if listResult.Err != nil {
		return withMeta(op.Failed[core.NoContent](fmt.Errorf("failed to list application binaries. %w", listResult.Err), false), id)
	}
	binaries := []jsonmodels.ApplicationBinary{}
	for item, err := range listResult.Items() {
		if err != nil {
			return withMeta(op.Failed[core.NoContent](fmt.Errorf("failed to list application binaries. %w", err), false), id)
		}
		binaries = append(binaries, item)
	}

	// Newest first. Binary ids are not ordered, so use the creation time; the
	// stable sort keeps the listing order for binaries without one.
	sort.SliceStable(binaries, func(i, j int) bool {
		return binaries[i].Get("created").Time().After(binaries[j].Get("created").Time())
	})

	protected := map[string]bool{}
	for _, p := range append([]string{activeID}, opt.Protect...) {
		if p != "" {
			protected[p] = true
		}
	}
	remaining := len(binaries)
	for i := len(binaries) - 1; i >= 0 && remaining > opt.Keep; i-- {
		if binaryID := binaries[i].ID(); !protected[binaryID] {
			toDelete = append(toDelete, binaryID)
			remaining--
		}
	}

	if len(toDelete) == 0 || ctxhelpers.IsDryRun(ctx) {
		return withMeta(op.OK(core.NoContent{}), id)
	}

	var errs []error
	for _, binaryID := range toDelete {
		if res := s.DeleteBinary(ctxhelpers.InternalStep(ctx), id, binaryID); res.Err != nil {
			errs = append(errs, fmt.Errorf("failed to delete application binary. binaryId=%s. %w", binaryID, res.Err))
			continue
		}
		deleted = append(deleted, binaryID)
	}

	result := op.OK(core.NoContent{})
	result.Status = op.StatusUpdated
	if err := errors.Join(errs...); err != nil {
		result = op.Failed[core.NoContent](err, false)
		if len(deleted) > 0 {
			result.Status = op.StatusPartial
		}
	}
	return withMeta(result, id)
}

// deletedBinaries returns the ids of the binaries a RotateBinaries result
// deleted.
func deletedBinaries(r op.Result[core.NoContent]) []string {
	if ids, ok := r.Meta["deletedBinaries"].([]string); ok {
		return ids
	}
	return []string{}
}
