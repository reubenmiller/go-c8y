// Package compilecost guards how much code the SDK forces into the packages
// that use it.
//
// Go compiles every method of a generic type for each of its instantiations,
// in every package that refers to them. The SDK has over a hundred
// op.Result[T] instantiations reachable from api.Client, so one more method on
// Result adds over a hundred functions to every importing package, and the
// build time of large programs (go-c8y-cli has hundreds of packages) grows
// with it. A method cannot be removed after the v2.0.0 release without a
// breaking change, so this test catches the growth before it is merged.
package compilecost

import (
	"os/exec"
	"strings"
	"testing"
)

// budget is the most functions the probe package may contain. Raise it only
// for a deliberate API addition, and say why in the pull request. The count
// was 3,615 when the budget was set (5,291 before op.Result was slimmed).
//
// Raised from 3,800 to 3,850 for the Digital Twin Manager read-only service
// (client.DigitalTwin): six op.Result and four pagination.Iterator
// instantiations for its models, 3,836 functions in all.
const budget = 3850

func TestCompiledFunctionBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the api package")
	}
	out, err := exec.Command("go", "list", "-export", "-f", "{{.Export}}", "./probe").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	export := strings.TrimSpace(string(out))
	nm, err := exec.Command("go", "tool", "nm", export).Output()
	if err != nil {
		t.Fatalf("go tool nm: %v", err)
	}
	count := 0
	for line := range strings.Lines(string(nm)) {
		fields := strings.Fields(line)
		if len(fields) >= 3 && (fields[1] == "T" || fields[1] == "t") {
			count++
		}
	}
	t.Logf("functions compiled into a package that uses api.Client: %d (budget %d)", count, budget)
	if count > budget {
		t.Errorf("a package that uses api.Client now compiles %d functions, over the budget of %d.\n"+
			"Each method on a generic type such as op.Result costs every importing package one function per instantiation.\n"+
			"Prefer a package function (see the op package documentation), or raise the budget if the addition is deliberate.", count, budget)
	}
}
