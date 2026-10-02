// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// AmbiguousMergeKeyLists names the strategic-merge lists whose patchMergeKey
// is narrower than the key the API uses to identify an entry. For these lists
// the API accepts several entries that share the merge key, for example
// 53/TCP and 53/UDP container ports, or two topology spread constraints on the
// same topologyKey with different whenUnsatisfiable actions. A strategic merge
// patch matches such entries by the merge key alone, so it merges, drops or
// deletes the wrong ones.
//
// The map is keyed by the JSON field name and its value is the merge key, so
// any list with that field name and merge key is covered wherever it appears.
// TestAmbiguousMergeKeyListsCoverWorkloadAPIs keeps it in step with k8s.io/api.
var AmbiguousMergeKeyLists = map[string]string{
	"ports":                     "containerPort",
	"topologySpreadConstraints": "topologyKey",
}

const (
	patchDirective        = "$patch"
	replaceDirective      = "replace"
	setElementOrderPrefix = "$setElementOrder/"
)

// TwoWayStrategicMergePatch returns a strategic merge patch that turns original
// into modified. It matches strategicpatch.CreateTwoWayMergePatch, so fields that
// Terraform does not manage are left as the server has them, except that a list
// that cannot be merged entry by entry is replaced as a whole whenever it
// changed. See AmbiguousMergeKeyLists.
func TwoWayStrategicMergePatch(original, modified []byte, dataStruct any) ([]byte, error) {
	patch, err := strategicpatch.CreateTwoWayMergePatch(original, modified, dataStruct)
	if err != nil {
		return nil, err
	}
	meta, err := strategicpatch.NewPatchMetaFromStruct(dataStruct)
	if err != nil {
		return nil, err
	}
	return replaceAmbiguousLists(patch, original, modified, nil, meta)
}

// ThreeWayStrategicMerge applies the change from original to modified onto
// current and returns the merged document. Fields present only in current are
// kept, except in lists that cannot be merged entry by entry, which end up
// exactly as in modified whenever original or current differ from it.
func ThreeWayStrategicMerge(original, modified, current []byte, dataStruct any) ([]byte, error) {
	meta, err := strategicpatch.NewPatchMetaFromStruct(dataStruct)
	if err != nil {
		return nil, err
	}
	patch, err := strategicpatch.CreateThreeWayMergePatch(original, modified, current, meta, true)
	if err != nil {
		return nil, err
	}
	patch, err = replaceAmbiguousLists(patch, original, modified, current, meta)
	if err != nil {
		return nil, err
	}
	return strategicpatch.StrategicMergePatch(current, patch, dataStruct)
}

func replaceAmbiguousLists(patchJSON, originalJSON, modifiedJSON, currentJSON []byte, meta strategicpatch.LookupPatchMeta) ([]byte, error) {
	var patch, original, modified, current map[string]any
	for _, document := range []struct {
		raw    []byte
		target *map[string]any
	}{
		{patchJSON, &patch},
		{originalJSON, &original},
		{modifiedJSON, &modified},
		{currentJSON, &current},
	} {
		if len(document.raw) == 0 {
			continue
		}
		if err := json.Unmarshal(document.raw, document.target); err != nil {
			return nil, err
		}
	}
	if patch == nil {
		patch = map[string]any{}
	}
	if err := replaceAmbiguousListsInMap(patch, original, modified, current, currentJSON != nil, meta); err != nil {
		return nil, err
	}
	return json.Marshal(patch)
}

// replaceAmbiguousListsInMap walks original and modified together. When a list
// that cannot be merged by its merge key has changed, it writes the whole
// modified list into the patch with a $patch: replace directive.
func replaceAmbiguousListsInMap(patch, original, modified, current map[string]any, threeWay bool, meta strategicpatch.LookupPatchMeta) error {
	keys := map[string]struct{}{}
	for key := range original {
		keys[key] = struct{}{}
	}
	for key := range modified {
		keys[key] = struct{}{}
	}
	for key := range keys {
		if strings.HasPrefix(key, "$") {
			continue
		}
		originalValue, modifiedValue, currentValue := original[key], modified[key], current[key]
		if patchValue, present := patch[key]; present && patchValue == nil {
			// The patch removes the whole field.
			continue
		}
		switch sample := firstNonNil(modifiedValue, originalValue).(type) {
		case map[string]any:
			nestedMeta, _, err := meta.LookupPatchMetadataForStruct(key)
			if err != nil {
				// Free-form maps such as labels or resource quantities hold no lists.
				continue
			}
			nestedPatch, existing := patch[key].(map[string]any)
			if !existing {
				nestedPatch = map[string]any{}
			}
			if err := replaceAmbiguousListsInMap(nestedPatch, asMap(originalValue), asMap(modifiedValue), asMap(currentValue), threeWay, nestedMeta); err != nil {
				return err
			}
			if !existing && len(nestedPatch) > 0 {
				patch[key] = nestedPatch
			}
		case []any:
			elementMeta, patchMeta, err := meta.LookupPatchMetadataForSlice(key)
			if err != nil {
				continue
			}
			mergeKey := patchMeta.GetPatchMergeKey()
			if mergeKey == "" || !hasMergeStrategy(patchMeta.GetPatchStrategies()) || !isListOfMaps(sample) {
				// Lists without a merge key are always replaced as a whole.
				continue
			}
			originalList, modifiedList, currentList := asSlice(originalValue), asSlice(modifiedValue), asSlice(currentValue)
			if AmbiguousMergeKeyLists[key] == mergeKey ||
				hasDuplicateMergeKeys(originalList, mergeKey) ||
				hasDuplicateMergeKeys(modifiedList, mergeKey) ||
				hasDuplicateMergeKeys(currentList, mergeKey) {
				changed := !reflect.DeepEqual(originalList, modifiedList) ||
					(threeWay && !reflect.DeepEqual(currentList, modifiedList))
				delete(patch, setElementOrderPrefix+key)
				switch {
				case !changed:
					delete(patch, key)
				case modifiedValue == nil:
					patch[key] = nil
				default:
					replacement := make([]any, 0, len(modifiedList)+1)
					replacement = append(replacement, map[string]any{patchDirective: replaceDirective})
					patch[key] = append(replacement, modifiedList...)
				}
				continue
			}
			if err := replaceAmbiguousListsInElements(patch, key, mergeKey, originalList, modifiedList, currentList, threeWay, elementMeta); err != nil {
				return err
			}
		}
	}
	return nil
}

// replaceAmbiguousListsInElements descends into the entries that original and
// modified share, such as a container that keeps its name while its ports
// change. Entries added by modified are already carried whole by the patch.
func replaceAmbiguousListsInElements(patch map[string]any, key, mergeKey string, original, modified, current []any, threeWay bool, meta strategicpatch.LookupPatchMeta) error {
	patchList, _ := patch[key].([]any)
	appended := false
	for _, item := range modified {
		modifiedItem, ok := item.(map[string]any)
		if !ok {
			continue
		}
		mergeValue, ok := modifiedItem[mergeKey]
		if !ok {
			continue
		}
		originalItem := findByMergeKey(original, mergeKey, mergeValue)
		if originalItem == nil {
			continue
		}
		patchItem := findByMergeKey(patchList, mergeKey, mergeValue)
		existing := patchItem != nil
		if !existing {
			patchItem = map[string]any{mergeKey: mergeValue}
		}
		if err := replaceAmbiguousListsInMap(patchItem, originalItem, modifiedItem, findByMergeKey(current, mergeKey, mergeValue), threeWay, meta); err != nil {
			return err
		}
		if !existing && len(patchItem) > 1 {
			patchList = append(patchList, patchItem)
			appended = true
		}
	}
	if appended {
		patch[key] = patchList
	}
	return nil
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func asMap(value any) map[string]any {
	typed, _ := value.(map[string]any)
	return typed
}

func asSlice(value any) []any {
	typed, _ := value.([]any)
	return typed
}

func hasMergeStrategy(strategies []string) bool {
	for _, strategy := range strategies {
		if strategy == "merge" {
			return true
		}
	}
	return false
}

func isListOfMaps(list []any) bool {
	for _, item := range list {
		if _, ok := item.(map[string]any); !ok {
			return false
		}
	}
	return true
}

// findByMergeKey returns the entry with the given merge key value, skipping
// patch directive entries such as {"$patch": "delete"}.
func findByMergeKey(list []any, mergeKey string, value any) map[string]any {
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if _, directive := entry[patchDirective]; directive {
			continue
		}
		if reflect.DeepEqual(entry[mergeKey], value) {
			return entry
		}
	}
	return nil
}

func hasDuplicateMergeKeys(list []any, mergeKey string) bool {
	seen := make(map[string]struct{}, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		value, ok := entry[mergeKey]
		if !ok {
			continue
		}
		identity := fmt.Sprintf("%T:%v", value, value)
		if _, duplicate := seen[identity]; duplicate {
			return true
		}
		seen[identity] = struct{}{}
	}
	return false
}
