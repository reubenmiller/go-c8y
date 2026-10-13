package managedobjects

import (
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
)

// ResolveListStrategy picks the pagination strategy for an inventory-style list
// (managed objects, devices, device groups — all backed by /inventory/managedObjects).
//
//   - Auto (default): the _id keyset, unless the caller supplied a non-_id
//     $orderby or a plain filter parameter, in which case offset is used.
//   - id: the _id keyset; an error if a conflicting $orderby or a plain filter
//     parameter is present.
//   - offset: classic currentPage paging.
//   - time: rejected — time keyset does not apply to inventory.
//
// query is the entity's raw Cumulocity query (its Query / q field), inspected
// for a conflicting $orderby. opts is the entity's list options, inspected for
// plain filter parameters (type, fragmentType, owner, ...): the platform ignores
// those when a query is sent, so the keyset cursor (which is a query) would
// silently drop them.
func ResolveListStrategy(kind pagination.StrategyKind, query string, opts any) (pagination.Strategy, error) {
	conflict := model.QueryHasConflictingOrder(query)
	filtered := hasPlainFilters(opts)
	switch kind {
	case pagination.StrategyOffset:
		return pagination.OffsetStrategy{}, nil
	case pagination.StrategyIDKeyset:
		if conflict {
			return nil, fmt.Errorf("pagination strategy %q cannot honour a custom $orderby; use %q instead", kind, pagination.StrategyOffset)
		}
		if filtered {
			return nil, fmt.Errorf("pagination strategy %q cannot be combined with plain filter parameters (type, fragmentType, owner, ...); use a query or %q instead", kind, pagination.StrategyOffset)
		}
		return pagination.IDKeysetStrategy{}, nil
	case pagination.StrategyTimeKeyset:
		return nil, fmt.Errorf("pagination strategy %q does not apply to inventory; use %q or %q", kind, pagination.StrategyIDKeyset, pagination.StrategyOffset)
	case pagination.StrategyAuto, "":
		if conflict || filtered {
			return pagination.OffsetStrategy{}, nil
		}
		return pagination.IDKeysetStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown pagination strategy %q", kind)
	}
}

// plainFilterParams are the inventory list parameters the platform ignores once
// a query is present. The device query (q) is included: it is a separate query
// language that cannot be combined with the keyset's query.
var plainFilterParams = []string{
	"type", "fragmentType", "text", "owner", "onlyRoots",
	"childAdditionId", "childAssetId", "childDeviceId", "ids", "q",
}

// hasPlainFilters reports whether the list options encode any plain filter
// parameter.
func hasPlainFilters(opts any) bool {
	if opts == nil {
		return false
	}
	params := core.QueryParameters(opts)
	for _, name := range plainFilterParams {
		if params.Get(name) != "" {
			return true
		}
	}
	return false
}
