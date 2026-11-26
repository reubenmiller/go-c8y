package contexthelpers

import (
	"context"
	"testing"
)

func TestInternalStepTurnsOptionsOff(t *testing.T) {
	ctx := WithConflictAsError(WithNotFoundAsError(context.Background()))
	step := InternalStep(ctx)
	if NotFoundAsError(step) || ConflictAsError(step) {
		t.Errorf("InternalStep kept an option: notFound=%v conflict=%v", NotFoundAsError(step), ConflictAsError(step))
	}
	if !NotFoundAsError(ctx) || !ConflictAsError(ctx) {
		t.Error("InternalStep changed the parent context")
	}
}

func TestInternalStepWithoutOptions(t *testing.T) {
	ctx := context.Background()
	if got := InternalStep(ctx); got != ctx {
		t.Error("InternalStep wrapped a context without options")
	}
}
