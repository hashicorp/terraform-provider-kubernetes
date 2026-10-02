// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// ambiguousMergeKeyLists names the strategic-merge lists whose patchMergeKey
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
var ambiguousMergeKeyLists = map[string]string{
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
// changed between original and modified. See ambiguousMergeKeyLists.
//
// live is the object the server holds, as last read, or nil when it is not
// known. It only tells which list entries the server already has, so that
// lists inside an entry that the patch merges into an existing one are
// replaced too, even when original lacks that entry. Pass nil to treat
// original as the server object.
func TwoWayStrategicMergePatch(original, modified, live []byte, dataStruct any) ([]byte, error) {
	patch, err := strategicpatch.CreateTwoWayMergePatch(original, modified, dataStruct)
	if err != nil {
		return nil, err
	}
	meta, err := strategicpatch.NewPatchMetaFromStruct(dataStruct)
	if err != nil {
		return nil, err
	}
	if live == nil {
		live = original
	}
	return replaceAmbiguousLists(patch, original, modified, live, false, meta)
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
	patch, err = replaceAmbiguousLists(patch, original, modified, current, true, meta)
	if err != nil {
		return nil, err
	}
	return strategicpatch.StrategicMergePatch(current, patch, dataStruct)
}

// ambiguousListReplacer rewrites a strategic merge patch so that lists which
// cannot be merged by their merge key are written whole. server is the
// document the patch is applied to: current for a three-way merge, and the
// live object (or original) for a two-way patch.
type ambiguousListReplacer struct {
	threeWay bool
	// rewritten records whether the patch was changed at all, so that an
	// untouched patch is returned exactly as strategicpatch produced it.
	rewritten bool
}

func replaceAmbiguousLists(patchJSON, originalJSON, modifiedJSON, serverJSON []byte, threeWay bool, meta strategicpatch.LookupPatchMeta) ([]byte, error) {
	var patch, original, modified, server map[string]any
	for _, document := range []struct {
		raw    []byte
		target *map[string]any
	}{
		{patchJSON, &patch},
		{originalJSON, &original},
		{modifiedJSON, &modified},
		{serverJSON, &server},
	} {
		if len(document.raw) == 0 {
			continue
		}
		// util/json keeps integers as int64 rather than float64, so values
		// above 2^53 are not rounded when the patch is encoded again.
		if err := json.Unmarshal(document.raw, document.target); err != nil {
			return nil, err
		}
	}
	if patch == nil {
		patch = map[string]any{}
	}
	replacer := &ambiguousListReplacer{threeWay: threeWay}
	if err := replacer.inMap(patch, original, modified, server, meta); err != nil {
		return nil, err
	}
	if !replacer.rewritten {
		return patchJSON, nil
	}
	return json.Marshal(patch)
}

// inMap walks original and modified together. When a list that cannot be
// merged by its merge key has changed, it writes the whole modified list into
// the patch with a $patch: replace directive.
func (r *ambiguousListReplacer) inMap(patch, original, modified, server map[string]any, meta strategicpatch.LookupPatchMeta) error {
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
		originalValue, modifiedValue, serverValue := original[key], modified[key], server[key]
		if patchValue, present := patch[key]; present && patchValue == nil {
			// The patch removes the whole field.
			continue
		}
		switch firstNonNil(modifiedValue, originalValue).(type) {
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
			if err := r.inMap(nestedPatch, asMap(originalValue), asMap(modifiedValue), asMap(serverValue), nestedMeta); err != nil {
				return err
			}
			if !existing && len(nestedPatch) > 0 {
				patch[key] = nestedPatch
				r.rewritten = true
			}
		case []any:
			elementMeta, patchMeta, err := meta.LookupPatchMetadataForSlice(key)
			if err != nil {
				continue
			}
			mergeKey := patchMeta.GetPatchMergeKey()
			if mergeKey == "" || !hasMergeStrategy(patchMeta.GetPatchStrategies()) {
				// Lists without a merge key are always replaced as a whole.
				continue
			}
			originalList, modifiedList, serverList := asSlice(originalValue), asSlice(modifiedValue), asSlice(serverValue)
			if ambiguousMergeKeyLists[key] == mergeKey ||
				hasDuplicateMergeKeys(originalList, mergeKey) ||
				hasDuplicateMergeKeys(modifiedList, mergeKey) ||
				hasDuplicateMergeKeys(serverList, mergeKey) {
				r.replaceList(patch, key, modifiedValue, !reflect.DeepEqual(originalList, modifiedList) ||
					(r.threeWay && !reflect.DeepEqual(serverList, modifiedList)))
				continue
			}
			if err := r.inElements(patch, key, mergeKey, originalList, modifiedList, serverList, elementMeta); err != nil {
				return err
			}
		}
	}
	return nil
}

// replaceList writes the whole modified list into the patch when it changed,
// and otherwise makes sure the patch does not touch it.
func (r *ambiguousListReplacer) replaceList(patch map[string]any, key string, modifiedValue any, changed bool) {
	orderKey := setElementOrderPrefix + key
	if _, ordered := patch[orderKey]; ordered {
		delete(patch, orderKey)
		r.rewritten = true
	}
	switch {
	case !changed:
		if _, present := patch[key]; present {
			delete(patch, key)
			r.rewritten = true
		}
	case modifiedValue == nil:
		patch[key] = nil
		r.rewritten = true
	default:
		modifiedList := asSlice(modifiedValue)
		replacement := make([]any, 0, len(modifiedList)+1)
		replacement = append(replacement, map[string]any{patchDirective: replaceDirective})
		patch[key] = append(replacement, modifiedList...)
		r.rewritten = true
	}
}

// inElements descends into the modified entries that the server already holds,
// such as a container that keeps its name while its ports change. The patch
// is merged into those entries, so their lists need the same treatment. An
// entry the server lacks is appended exactly as the patch carries it, and
// strategicpatch does not process directives inside it, so it is left alone:
// a three-way patch already carries such an entry whole.
func (r *ambiguousListReplacer) inElements(patch map[string]any, key, mergeKey string, original, modified, server []any, meta strategicpatch.LookupPatchMeta) error {
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
		serverItem := findByMergeKey(server, mergeKey, mergeValue)
		if serverItem == nil {
			continue
		}
		patchItem := findByMergeKey(patchList, mergeKey, mergeValue)
		existing := patchItem != nil
		if !existing {
			patchItem = map[string]any{mergeKey: mergeValue}
		}
		if err := r.inMap(patchItem, findByMergeKey(original, mergeKey, mergeValue), modifiedItem, serverItem, meta); err != nil {
			return err
		}
		if !existing && len(patchItem) > 1 {
			patchList = append(patchList, patchItem)
			appended = true
		}
	}
	if appended {
		patch[key] = patchList
		r.rewritten = true
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
