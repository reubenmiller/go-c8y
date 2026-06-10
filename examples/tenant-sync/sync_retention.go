package main

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/retentionrules"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// SyncRetentionRules ensures retention rules exist with the desired maximum
// age. Rules have no name, so they are matched by their selector combination
// (dataType/fragmentType/type/source) against the rules in the tenant;
// maximumAge (and editable, when set) is updated when it differs.
func (s *Syncer) SyncRetentionRules(ctx context.Context, specs []RetentionRuleSpec) error {
	if len(specs) == 0 {
		return nil
	}

	if s.DryRun {
		for _, spec := range specs {
			s.record(SectionRetentionRules, spec.Selector(), ActionPlanned,
				fmt.Sprintf("ensure maximumAge=%d", spec.MaximumAge), nil)
		}
		return nil
	}

	existing, err := s.listRetentionRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to list retention rules: %w", err)
	}

	for _, spec := range specs {
		selector := spec.Selector()
		detail := fmt.Sprintf("maximumAge=%d", spec.MaximumAge)

		body := map[string]any{
			"dataType":     orWildcard(spec.DataType),
			"fragmentType": orWildcard(spec.FragmentType),
			"type":         orWildcard(spec.Type),
			"source":       orWildcard(spec.Source),
			"maximumAge":   spec.MaximumAge,
		}
		if spec.Editable != nil {
			body["editable"] = *spec.Editable
		}

		rule, found := existing[selector]
		if !found {
			result := s.Client.RetentionRules.Create(ctx, body)
			s.record(SectionRetentionRules, selector, ActionCreated, detail, result.Err)
			continue
		}

		upToDate := rule.MaximumAge() == spec.MaximumAge &&
			(spec.Editable == nil || rule.Editable() == *spec.Editable)
		if upToDate {
			s.record(SectionRetentionRules, selector, ActionUnchanged, detail, nil)
			continue
		}

		result := s.Client.RetentionRules.Update(ctx, rule.ID(), body)
		s.record(SectionRetentionRules, selector, ActionUpdated, detail, retentionUpdateErrorHint(result.Err, rule))
	}
	return nil
}

// listRetentionRules fetches the rules of the target tenant, keyed by their
// selector combination
func (s *Syncer) listRetentionRules(ctx context.Context) (map[string]jsonmodels.RetentionRule, error) {
	rules := make(map[string]jsonmodels.RetentionRule)
	result := s.Client.RetentionRules.List(ctx, retentionrules.ListOptions{
		PaginationOptions: pagination.PaginationOptions{PageSize: 2000},
	})
	for rule, err := range op.Iter2(result) {
		if err != nil {
			return nil, err
		}
		// Missing fields count as wildcards, like in the manifest
		selector := orWildcard(rule.DataType()) + "/" + orWildcard(rule.FragmentType()) + "/" +
			orWildcard(rule.Get("type").String()) + "/" + orWildcard(rule.Get("source").String())
		rules[selector] = rule
	}
	return rules, nil
}

// retentionUpdateErrorHint explains failures updating non-editable rules
func retentionUpdateErrorHint(err error, rule jsonmodels.RetentionRule) error {
	if err != nil && !rule.Editable() {
		return fmt.Errorf("%w (the existing rule is not editable; set editable: true or remove the system rule first)", err)
	}
	return err
}
