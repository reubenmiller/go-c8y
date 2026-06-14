package managedobjects

import (
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
)

// ResolveListStrategy picks the pagination strategy for an inventory-style list
// (managed objects, devices, device groups — all backed by /inventory/managedObjects).
//
//   - Auto (default): the _id keyset, unless the caller supplied a non-_id
//     $orderby, in which case offset is used so that ordering is preserved.
//   - id: the _id keyset; an error if a conflicting $orderby is present.
//   - offset: classic currentPage paging.
//   - time: rejected — time keyset does not apply to inventory.
//
// query is the entity's raw Cumulocity query (its Query / q field), inspected
// for a conflicting $orderby.
func ResolveListStrategy(kind pagination.StrategyKind, query string) (pagination.Strategy, error) {
	conflict := model.QueryHasConflictingOrder(query)
	switch kind {
	case pagination.StrategyOffset:
		return pagination.OffsetStrategy{}, nil
	case pagination.StrategyIDKeyset:
		if conflict {
			return nil, fmt.Errorf("pagination strategy %q cannot honour a custom $orderby; use %q instead", kind, pagination.StrategyOffset)
		}
		return pagination.IDKeysetStrategy{}, nil
	case pagination.StrategyTimeKeyset:
		return nil, fmt.Errorf("pagination strategy %q does not apply to inventory; use %q or %q", kind, pagination.StrategyIDKeyset, pagination.StrategyOffset)
	case pagination.StrategyAuto, "":
		if conflict {
			return pagination.OffsetStrategy{}, nil
		}
		return pagination.IDKeysetStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown pagination strategy %q", kind)
	}
}
