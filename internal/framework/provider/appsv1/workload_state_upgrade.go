// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"fmt"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
)

// upgradeWorkloadState is shared by same-type upgrades and legacy alias moves.
// Retained spec/template blocks remain singleton lists; only selected attributes
// change to objects. Source schema version 0 also needs the old quantity repair.
func upgradeWorkloadState(version int64, singleton string) func(map[string]any) error {
	return func(raw map[string]any) error {
		if err := podspec.UpgradeState(raw, "state", []string{"spec", "template", "spec"}, version == 0); err != nil {
			return err
		}
		specs, _ := raw["spec"].([]any) // The shared traversal validated the retained blocks.
		for _, entry := range specs {
			spec := entry.(map[string]any)
			object, err := unwrapWorkloadSingleton(spec, singleton, "state.spec[0]."+singleton)
			if err != nil {
				return err
			}
			if singleton == "strategy" && object != nil {
				if _, err := unwrapWorkloadSingleton(object, "rolling_update", "state.spec[0].strategy.rolling_update"); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// Historical absence is null for strategy, rolling_update and retention policy.
// Unlike container resources, none of these attributes denotes a zero value object.
func unwrapWorkloadSingleton(parent map[string]any, name, location string) (map[string]any, error) {
	value := parent[name]
	if value == nil {
		return nil, nil
	}
	if object, ok := value.(map[string]any); ok {
		return object, nil
	}
	list, ok := value.([]any)
	if !ok || len(list) > 1 {
		return nil, fmt.Errorf("%s must contain at most one object", location)
	}
	if len(list) == 0 {
		parent[name] = nil
		return nil, nil
	}
	object, ok := list[0].(map[string]any)
	if !ok || object == nil {
		return nil, fmt.Errorf("%s[0] must be an object", location)
	}
	parent[name] = object
	return object, nil
}
