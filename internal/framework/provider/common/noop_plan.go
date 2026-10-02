// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// NoOpPlan keeps computed-only differences from scheduling a spurious update. Unlike
// UseStateForUnknown on mutable metadata, this leaves all computed values unknown
// when any configured value changes or configuration is not yet fully known.
func NoOpPlan(config, plan, state tftypes.Value) (tftypes.Value, bool, error) {
	if config.IsNull() || plan.IsNull() || state.IsNull() {
		return plan, false, nil
	}
	hasUnknownConfig := false
	if err := tftypes.Walk(config, func(_ *tftypes.AttributePath, value tftypes.Value) (bool, error) {
		if !value.IsKnown() {
			hasUnknownConfig = true
			return false, nil
		}
		return true, nil
	}); err != nil {
		return plan, false, err
	}
	if hasUnknownConfig {
		return plan, false, nil
	}

	withPriorComputed, err := tftypes.Transform(plan, func(at *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
		if value.IsKnown() {
			return value, nil
		}
		prior, _, err := tftypes.WalkAttributePath(state, at)
		if errors.Is(err, tftypes.ErrInvalidStep) {
			// A newly added collection element has no corresponding prior value.
			return value, nil
		}
		if err != nil {
			return value, err
		}
		priorValue, ok := prior.(tftypes.Value)
		if !ok {
			return value, fmt.Errorf("prior state at %s has unexpected type %T", at, prior)
		}
		return priorValue, nil
	})
	if err != nil {
		return plan, false, err
	}
	if withPriorComputed.Equal(state) {
		return state, true, nil
	}
	return plan, false, nil
}
