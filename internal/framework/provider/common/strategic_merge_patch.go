// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// ambiguousMergeKeyLists maps list field names to a patchMergeKey that does not
// identify an entry on its own: the API accepts both 53/TCP and 53/UDP ports,
// and several topology spread constraints on one topologyKey.
var ambiguousMergeKeyLists = map[string]string{
	"ports":                     "containerPort",
	"topologySpreadConstraints": "topologyKey",
}

const setElementOrderPrefix = "$setElementOrder/"

// ThreeWayStrategicMerge moves the JSON document current from original to
// modified the way a three-way strategic merge patch does: fields modified
// does not mention keep their current values. Lists the patch changes whose
// entries cannot be matched by merge key (see ambiguousMergeKeyLists, or lists
// holding duplicate merge keys) are set to their value in modified, in its
// exact order.
func ThreeWayStrategicMerge(original, modified, current []byte, dataStruct any) ([]byte, error) {
	meta, err := strategicpatch.NewPatchMetaFromStruct(dataStruct)
	if err != nil {
		return nil, err
	}
	patch, err := strategicpatch.CreateThreeWayMergePatch(original, modified, current, meta, true)
	if err != nil {
		return nil, err
	}
	// Decode the way strategicpatch does so merge-key values compare equal.
	var patchMap, currentMap, modifiedMap map[string]any
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(current, &currentMap); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(modified, &modifiedMap); err != nil {
		return nil, err
	}
	if err := replaceAmbiguousLists(patchMap, currentMap, modifiedMap, meta); err != nil {
		return nil, err
	}
	patch, err = json.Marshal(patchMap)
	if err != nil {
		return nil, err
	}
	mergedJSON, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(current, patch, meta)
	if err != nil {
		return nil, err
	}
	var merged map[string]any
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		return nil, err
	}
	// Strategic merge sorts even a replaced list by merge key, so the
	// replacements are copied into the result verbatim.
	if err := copyReplacedLists(patchMap, merged, meta); err != nil {
		return nil, err
	}
	return json.Marshal(merged)
}

// StrategicMergeSpecOps returns a JSON patch operation that replaces the spec
// of live with its spec moved from original to modified (see
// ThreeWayStrategicMerge). Callers send it with ResourceVersionGuard, since it
// overwrites the whole spec read from live.
func StrategicMergeSpecOps(live *unstructured.Unstructured, original, modified, dataStruct any) (kubernetes.PatchOperations, error) {
	var docs [3][]byte
	for i, spec := range []any{original, modified, live.Object["spec"]} {
		doc, err := json.Marshal(map[string]any{"spec": spec})
		if err != nil {
			return nil, err
		}
		docs[i] = doc
	}
	mergedJSON, err := ThreeWayStrategicMerge(docs[0], docs[1], docs[2], dataStruct)
	if err != nil {
		return nil, err
	}
	var merged map[string]any
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		return nil, err
	}
	return kubernetes.PatchOperations{&kubernetes.ReplaceOperation{Path: "/spec", Value: merged["spec"]}}, nil
}

// ResourceVersionGuard returns a JSON patch operation that makes the server
// reject the patch with a Conflict if the object changed since resourceVersion.
func ResourceVersionGuard(resourceVersion string) kubernetes.PatchOperation {
	return &kubernetes.ReplaceOperation{Path: "/metadata/resourceVersion", Value: resourceVersion}
}

// replaceAmbiguousLists rewrites each ambiguous list in patch into a
// "$patch: replace" list holding the entries of modified.
func replaceAmbiguousLists(patch, live, modified map[string]any, meta strategicpatch.LookupPatchMeta) error {
	lists := map[string]bool{}
	for key, value := range patch {
		if name, ok := strings.CutPrefix(key, setElementOrderPrefix); ok {
			lists[name] = true
			continue
		}
		if strings.HasPrefix(key, "$") {
			continue
		}
		switch value := value.(type) {
		case map[string]any:
			nestedMeta, _, err := meta.LookupPatchMetadataForStruct(key)
			if err != nil {
				// A field dataStruct does not know; nothing to rewrite.
				continue
			}
			liveNested, _ := live[key].(map[string]any)
			modifiedNested, _ := modified[key].(map[string]any)
			if err := replaceAmbiguousLists(value, liveNested, modifiedNested, nestedMeta); err != nil {
				return err
			}
		case []any:
			lists[key] = true
		}
	}
	for name := range lists {
		itemMeta, patchMeta, err := meta.LookupPatchMetadataForSlice(name)
		if err != nil {
			return err
		}
		mergeKey := patchMeta.GetPatchMergeKey()
		if mergeKey == "" || !slices.Contains(patchMeta.GetPatchStrategies(), "merge") {
			continue
		}
		liveList, _ := live[name].([]any)
		modifiedList, _ := modified[name].([]any)
		if ambiguousMergeKeyLists[name] == mergeKey || hasDuplicateMergeKeys(liveList, mergeKey) || hasDuplicateMergeKeys(modifiedList, mergeKey) {
			delete(patch, setElementOrderPrefix+name)
			if modifiedList == nil {
				patch[name] = nil
			} else {
				patch[name] = append([]any{replaceDirective()}, modifiedList...)
			}
			continue
		}
		patchList, _ := patch[name].([]any)
		for _, item := range patchList {
			entry, ok := item.(map[string]any)
			if !ok || entry[mergeKey] == nil {
				continue
			}
			liveEntry := findByMergeKey(liveList, mergeKey, entry[mergeKey])
			modifiedEntry := findByMergeKey(modifiedList, mergeKey, entry[mergeKey])
			if err := replaceAmbiguousLists(entry, liveEntry, modifiedEntry, itemMeta); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyReplacedLists sets every "$patch: replace" list in patch on merged.
func copyReplacedLists(patch, merged map[string]any, meta strategicpatch.LookupPatchMeta) error {
	for key, value := range patch {
		if strings.HasPrefix(key, "$") {
			continue
		}
		switch value := value.(type) {
		case map[string]any:
			nestedMeta, _, err := meta.LookupPatchMetadataForStruct(key)
			if err != nil {
				continue
			}
			if nested, ok := merged[key].(map[string]any); ok {
				if err := copyReplacedLists(value, nested, nestedMeta); err != nil {
					return err
				}
			}
		case []any:
			if len(value) > 0 && reflect.DeepEqual(value[0], replaceDirective()) {
				merged[key] = value[1:]
				continue
			}
			itemMeta, patchMeta, err := meta.LookupPatchMetadataForSlice(key)
			if err != nil {
				return err
			}
			mergeKey := patchMeta.GetPatchMergeKey()
			if mergeKey == "" {
				continue
			}
			mergedList, _ := merged[key].([]any)
			for _, item := range value {
				entry, ok := item.(map[string]any)
				if !ok || entry[mergeKey] == nil {
					continue
				}
				if mergedEntry := findByMergeKey(mergedList, mergeKey, entry[mergeKey]); mergedEntry != nil {
					if err := copyReplacedLists(entry, mergedEntry, itemMeta); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func replaceDirective() map[string]any {
	return map[string]any{"$patch": "replace"}
}

func findByMergeKey(list []any, mergeKey string, value any) map[string]any {
	for _, item := range list {
		if entry, ok := item.(map[string]any); ok && reflect.DeepEqual(entry[mergeKey], value) {
			return entry
		}
	}
	return nil
}

func hasDuplicateMergeKeys(list []any, mergeKey string) bool {
	seen := make(map[string]bool, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok || entry[mergeKey] == nil {
			continue
		}
		identity := fmt.Sprintf("%T:%v", entry[mergeKey], entry[mergeKey])
		if seen[identity] {
			return true
		}
		seen[identity] = true
	}
	return false
}
