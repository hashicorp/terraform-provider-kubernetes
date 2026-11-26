// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Drift detection relies on the field ownership the API server records for
// server-side apply. After an apply, the manager's entry in managedFields lists
// every field of the manifest. A field leaves that entry when another client
// changes or removes it, so a manifest field that is still listed needs no
// comparison: its live value is the applied one, however the server spells it.
// Only fields that are no longer listed are compared by value.

// ownership is the part of a subtree that a field manager owns.
type ownership struct {
	all    bool
	fields map[string]any // a FieldsV1 node
}

func (o ownership) child(key string) ownership {
	if o.all {
		return o
	}
	node, found := o.fields[key]
	if !found {
		return ownership{}
	}
	fields, _ := node.(map[string]any)
	return ownership{all: len(fields) == 0, fields: fields}
}

// managedFields merges the FieldsV1 sets of the entries that match.
func managedFields(obj *unstructured.Unstructured, match func(metav1.ManagedFieldsEntry) bool) map[string]any {
	merged := map[string]any{}
	for _, entry := range obj.GetManagedFields() {
		var fields map[string]any
		if entry.FieldsV1 == nil || !match(entry) || json.Unmarshal(entry.FieldsV1.GetRawBytes(), &fields) != nil {
			continue
		}
		mergeFields(merged, fields)
	}
	return merged
}

func mergeFields(into, from map[string]any) {
	for key, value := range from {
		source, _ := value.(map[string]any)
		target, exists := into[key].(map[string]any)
		if !exists {
			target = map[string]any{}
			into[key] = target
		}
		mergeFields(target, source)
	}
}

// appliedBy selects what a manager owns through server-side apply.
func appliedBy(manager string) func(metav1.ManagedFieldsEntry) bool {
	return func(entry metav1.ManagedFieldsEntry) bool {
		return entry.Manager == manager && entry.Operation == metav1.ManagedFieldsOperationApply && entry.Subresource == ""
	}
}

// ignoredPaths are never compared: the identity cannot drift, and the server
// owns the rest.
var ignoredPaths = map[string]bool{
	"apiVersion": true, "kind": true, "status": true,
	"metadata.name": true, "metadata.namespace": true,
	"metadata.creationTimestamp": true, "metadata.deletionGracePeriodSeconds": true,
	"metadata.deletionTimestamp": true, "metadata.generation": true, "metadata.managedFields": true,
	"metadata.resourceVersion": true, "metadata.selfLink": true, "metadata.uid": true,
}

// absent is the fingerprint of a value the live object does not have.
const absent = "absent"

type comparison struct {
	// skipped are the computed_fields: manifest paths that are not compared.
	skipped []string
	// handOver turns the comparison around: only the computed fields are
	// compared, to find the ones another client has taken.
	handOver bool
	// accepted holds differences seen right after an apply, by path. Applying
	// again cannot remove them, so they are not reported as drift.
	accepted map[string]string
	// found maps the path of each difference to a fingerprint of the live value.
	found map[string]string
}

// liveChanges returns the manifest as it would read if it described the live
// object: values another client changed are replaced by the live ones and
// removed values are left out. The second result is empty when nothing drifted.
func liveChanges(manifest map[string]any, live *unstructured.Unstructured, manager string, computedFields []string, accepted map[string]string) (map[string]any, map[string]string) {
	c := &comparison{accepted: accepted, found: map[string]string{}}
	for _, field := range computedFields {
		c.skipped = append(c.skipped, plainPath(mapKeys.ReplaceAllString(field, ".$1")))
	}
	own := ownership{fields: managedFields(live, appliedBy(manager))}
	shape := managedFields(live, func(metav1.ManagedFieldsEntry) bool { return true })
	result, _ := c.value("", manifest, live.Object, true, own, shape)
	described := result.(map[string]any)
	if live.GetAPIVersion() == "v1" && live.GetKind() == "Secret" {
		c.secretStrings(manifest, described, live)
	}
	return described, c.found
}

var (
	mapKeys       = regexp.MustCompile(`\["([^"]*)"\]`)
	listSelectors = regexp.MustCompile(`\[[^\]]*\]`)
)

// plainPath drops the list positions and keys from a path. computed_fields
// do not tell the elements of a list apart.
func plainPath(path string) string {
	return listSelectors.ReplaceAllString(path, "")
}

func (c *comparison) skip(path string) bool {
	plain := plainPath(path)
	for _, field := range c.skipped {
		if plain == field || strings.HasPrefix(plain, field+".") {
			return true
		}
	}
	return false
}

// leavingComputed returns the manifest to apply to an existing object, without
// the computed fields this manager no longer owns. Sending such a field, even
// with the other client's value, would make Terraform an owner of it again,
// and a controller that applies the field itself would then conflict.
func leavingComputed(manifest map[string]any, live *unstructured.Unstructured, manager string, computedFields []string) map[string]any {
	if len(computedFields) == 0 {
		return manifest
	}
	c := &comparison{handOver: true, found: map[string]string{}}
	for _, field := range computedFields {
		c.skipped = append(c.skipped, plainPath(mapKeys.ReplaceAllString(field, ".$1")))
	}
	own := ownership{fields: managedFields(live, appliedBy(manager))}
	shape := managedFields(live, func(metav1.ManagedFieldsEntry) bool { return true })
	result, _ := c.value("", manifest, live.Object, true, own, shape)
	return result.(map[string]any)
}

// secretStrings compares the stringData of a Secret, which the API server
// stores as data and returns in no other form.
func (c *comparison) secretStrings(manifest, described map[string]any, live *unstructured.Unstructured) {
	written, _ := manifest["stringData"].(map[string]any)
	if len(written) == 0 {
		return
	}
	stored, _, _ := unstructured.NestedMap(live.Object, "data")
	out := make(map[string]any, len(written))
	for key, want := range written {
		delete(c.found, "stringData."+key)
		encoded, exists := stored[key].(string)
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if exists && err == nil && string(decoded) == want || c.skip("stringData."+key) {
			out[key] = want
		} else if value, keep := c.differs("stringData."+key, want, string(decoded), exists); keep {
			out[key] = value
		}
	}
	described["stringData"] = out
}

// value compares one manifest value with its live counterpart. shape is the
// ownership of all managers, which shows how the API merges each list.
func (c *comparison) value(path string, want, have any, exists bool, own ownership, shape map[string]any) (any, bool) {
	if own.all || ignoredPaths[path] || !c.handOver && c.skip(path) {
		return want, true
	}
	switch want := want.(type) {
	case map[string]any:
		haveMap, _ := have.(map[string]any)
		out := make(map[string]any, len(want))
		for name, wanted := range want {
			live, found := haveMap[name]
			childShape, _ := shape["f:"+name].(map[string]any)
			if value, keep := c.value(strings.TrimPrefix(path+"."+name, "."), wanted, live, found, own.child("f:"+name), childShape); keep {
				out[name] = value
			}
		}
		return out, exists || len(out) > 0 || len(want) == 0
	case []any:
		haveList, _ := have.([]any)
		switch {
		case hasPrefixedKey(shape, "k:") && allObjects(want):
			return c.keyedList(path, want, haveList, own, shape), true
		case hasPrefixedKey(shape, "v:"):
			return c.valueSet(path, want, haveList, own), true
		case allObjects(want) && len(want) == len(haveList):
			// A list the API replaces as a whole: elements pair by position.
			out := make([]any, len(want))
			for i := range want {
				out[i], _ = c.value(fmt.Sprintf("%s[%d]", path, i), want[i], haveList[i], true, ownership{}, nil)
			}
			return out, true
		}
	}
	if c.handOver && c.skip(path) {
		return nil, false
	}
	// An empty list that is absent is no difference, like an empty map: a
	// client that rewrites the object leaves both out.
	if list, isList := want.([]any); exists && same(want, have) || !exists && isList && len(list) == 0 {
		return want, true
	}
	return c.differs(path, want, have, exists)
}

// keyedList handles a list the API merges by key, such as containers by name.
// Elements other clients added are theirs and are not part of the result.
func (c *comparison) keyedList(path string, want, have []any, own ownership, shape map[string]any) []any {
	out := make([]any, 0, len(want))
	for i, element := range want {
		item := element.(map[string]any)
		key, keys := elementKey(item, own, shape)
		if key == "" {
			if value, keep := c.differs(fmt.Sprintf("%s[%d]", path, i), item, nil, false); keep {
				out = append(out, value)
			}
			continue
		}
		var live any
		found := false
		for _, candidate := range have {
			if m, ok := candidate.(map[string]any); ok && hasKeys(m, keys, true) {
				live, found = m, true
				break
			}
		}
		childShape, _ := shape[key].(map[string]any)
		if value, keep := c.value(path+"["+describe(keys)+"]", item, live, found, own.child(key), childShape); keep {
			out = append(out, value)
		}
	}
	return out
}

// elementKey finds the list key an element is merged by. The manifest may
// leave out key fields the server defaults, such as a port's protocol, so a key
// matches when the fields the element does set agree. A key the manager owns
// wins over one it does not.
func elementKey(item map[string]any, own ownership, shape map[string]any) (string, map[string]any) {
	candidates := make([]string, 0, len(shape))
	for key := range shape {
		if strings.HasPrefix(key, "k:") {
			candidates = append(candidates, key)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		_, iOwned := own.fields[candidates[i]]
		_, jOwned := own.fields[candidates[j]]
		if iOwned != jOwned {
			return iOwned
		}
		return candidates[i] < candidates[j]
	})
	for _, key := range candidates {
		var keys map[string]any
		if json.Unmarshal([]byte(key[2:]), &keys) == nil && hasKeys(item, keys, false) {
			return key, keys
		}
	}
	return "", nil
}

// hasKeys reports whether item agrees with the key fields. With all unset, it
// is enough that the fields item sets agree, as long as it sets one.
func hasKeys(item, keys map[string]any, all bool) bool {
	matched := false
	for name, value := range keys {
		got, set := item[name]
		if !set && !all {
			continue
		}
		if !set || !same(got, value) {
			return false
		}
		matched = true
	}
	return matched
}

// valueSet handles a list of scalars the API merges as a set, such as
// finalizers.
func (c *comparison) valueSet(path string, want, have []any, own ownership) []any {
	out := make([]any, 0, len(want))
	for _, element := range want {
		encoded, _ := json.Marshal(element)
		_, owned := own.fields["v:"+string(encoded)]
		present := false
		for _, candidate := range have {
			present = present || same(element, candidate)
		}
		if owned || present {
			out = append(out, element)
		} else if value, keep := c.differs(path+"["+string(encoded)+"]", element, nil, false); keep {
			out = append(out, value)
		}
	}
	return out
}

func (c *comparison) differs(path string, want, have any, exists bool) (any, bool) {
	if c.handOver && !c.skip(path) {
		return want, true
	}
	print := absent
	if exists {
		encoded, _ := json.Marshal(have)
		sum := sha256.Sum256(encoded)
		print = hex.EncodeToString(sum[:8])
	}
	if c.accepted[path] == print {
		return want, true
	}
	c.found[path] = print
	return have, exists
}

// same compares a manifest value with a live value no manager entry vouches
// for, as JSON: true is not "true" and 1 is not "1". Nothing here knows the
// kind's schema, so a spelling the API would normalize, such as 1 for "1" in
// a quantity, counts as a difference until an apply makes the field owned.
func same(want, have any) bool {
	switch want := want.(type) {
	case map[string]any:
		haveMap, ok := have.(map[string]any)
		if !ok || len(want) != len(haveMap) {
			return false
		}
		for name, value := range want {
			if other, found := haveMap[name]; !found || !same(value, other) {
				return false
			}
		}
		return true
	case []any:
		haveList, ok := have.([]any)
		if !ok || len(want) != len(haveList) {
			return false
		}
		for i := range want {
			if !same(want[i], haveList[i]) {
				return false
			}
		}
		return true
	}
	wantNumber, wantIsNumber := number(want)
	haveNumber, haveIsNumber := number(have)
	if wantIsNumber || haveIsNumber {
		return wantIsNumber && haveIsNumber && wantNumber == haveNumber
	}
	return want == have
}

// number returns a JSON number, which decoders deliver as either type.
func number(value any) (float64, bool) {
	switch value := value.(type) {
	case int64:
		return float64(value), true
	case float64:
		return value, true
	}
	return 0, false
}

// project returns the part of a live value that a FieldsV1 set covers.
func project(value any, fields map[string]any) any {
	if len(fields) == 0 {
		return value
	}
	switch value := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, sub := range fields {
			name, isField := strings.CutPrefix(key, "f:")
			if member, found := value[name]; isField && found {
				subFields, _ := sub.(map[string]any)
				out[name] = project(member, subFields)
			}
		}
		return out
	case []any:
		out := []any{}
		for _, element := range value {
			for key, sub := range fields {
				subFields, _ := sub.(map[string]any)
				item, isObject := element.(map[string]any)
				var keys map[string]any
				var member any
				switch {
				case strings.HasPrefix(key, "k:") && isObject && json.Unmarshal([]byte(key[2:]), &keys) == nil && hasKeys(item, keys, true):
					out = append(out, project(item, subFields))
				case strings.HasPrefix(key, "v:") && json.Unmarshal([]byte(key[2:]), &member) == nil && same(member, element):
					out = append(out, element)
				}
			}
		}
		return out
	}
	return value
}

func allObjects(list []any) bool {
	for _, element := range list {
		if _, ok := element.(map[string]any); !ok {
			return false
		}
	}
	return len(list) > 0
}

func hasPrefixedKey(fields map[string]any, prefix string) bool {
	for key := range fields {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// describe renders list keys for a path, such as name=web.
func describe(keys map[string]any) string {
	if name, ok := keys["name"]; ok {
		return fmt.Sprintf("name=%v", name)
	}
	parts := make([]string, 0, len(keys))
	for name, value := range keys {
		if _, nested := value.(map[string]any); !nested {
			parts = append(parts, fmt.Sprintf("%s=%v", name, value))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
