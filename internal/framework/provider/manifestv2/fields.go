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
	"slices"
	"sort"
	"strconv"
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

// child returns the ownership of one member; shape is that member's node in
// the fields of all managers. A member listed without members of its own is
// owned as a whole only when no manager lists any: that is an atomic field.
// The entry of a map whose fields another client took over looks the same,
// but that client lists them.
func (o ownership) child(key string, shape map[string]any) ownership {
	if o.all {
		return o
	}
	node, found := o.fields[key]
	if !found {
		return ownership{}
	}
	fields, _ := node.(map[string]any)
	if len(fields) == 0 {
		return ownership{all: len(shape) == 0}
	}
	return ownership{fields: fields}
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

// mergeFields adds the FieldsV1 set from to into. A node owned as a whole
// that gains members keeps its own ownership as ".".
func mergeFields(into, from map[string]any) {
	for key, value := range from {
		source, _ := value.(map[string]any)
		target, exists := into[key].(map[string]any)
		if !exists {
			target = map[string]any{}
			into[key] = target
		} else if (len(target) == 0) != (len(source) == 0) && key != "." {
			target["."] = map[string]any{}
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

// location is a field path as drift and messages show it, and the names of
// its fields, which computed_fields match. A map key that is not a plain name
// is written as ["key"].
type location struct {
	path  string
	names []string
}

var plainName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func (l location) field(name string) location {
	path := l.path + "." + name
	switch {
	case !plainName.MatchString(name):
		path = l.path + `["` + name + `"]`
	case l.path == "":
		path = name
	}
	return location{path: path, names: append(l.names[:len(l.names):len(l.names)], name)}
}

// element names a list element. computed_fields do not tell elements apart.
func (l location) element(selector string) location {
	return location{path: l.path + "[" + selector + "]", names: l.names}
}

var listPosition = regexp.MustCompile(`^(\d+|\*)$`)

// fieldNames reads a path of computed_fields into the names of its fields.
// A list position is not a name: an entry for one element applies to all.
func fieldNames(path string) ([]string, error) {
	invalid := fmt.Errorf("%q is not a field path. Write it like spec.replicas, spec.template.spec.containers[0].image "+
		`or metadata.annotations["example.com/owner"]`, path)
	var names []string
	for rest := path; rest != ""; {
		switch {
		case strings.HasPrefix(rest, `["`):
			end := strings.Index(rest[2:], `"]`)
			if end <= 0 {
				return nil, invalid
			}
			names = append(names, rest[2:2+end])
			rest = rest[end+4:]
		case rest[0] == '[':
			end := strings.IndexByte(rest, ']')
			if end < 0 || !listPosition.MatchString(rest[1:end]) {
				return nil, invalid
			}
			rest = rest[end+1:]
		default:
			end := strings.IndexAny(rest, ".[")
			if end < 0 {
				end = len(rest)
			}
			if end == 0 {
				return nil, invalid
			}
			names = append(names, rest[:end])
			rest = rest[end:]
		}
		if after, dot := strings.CutPrefix(rest, "."); dot {
			if after == "" || after[0] == '[' {
				return nil, invalid
			}
			rest = after
		} else if rest != "" && rest[0] != '[' {
			return nil, invalid
		}
	}
	if len(names) == 0 {
		return nil, invalid
	}
	return names, nil
}

// hasField reports whether a manifest sets the field; in a list, any element.
func hasField(value any, names []string) bool {
	if len(names) == 0 {
		return true
	}
	switch value := value.(type) {
	case map[string]any:
		member, found := value[names[0]]
		return found && hasField(member, names[1:])
	case []any:
		for _, element := range value {
			if hasField(element, names) {
				return true
			}
		}
	}
	return false
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
	// skipped are the computed fields, as field names: they are not compared.
	skipped [][]string
	// handOver turns the comparison around: only the computed fields are
	// compared, to find the ones another client has taken.
	handOver bool
	// accepted holds differences seen right after an apply, by path. Applying
	// again cannot remove them, so they are not reported as drift.
	accepted map[string]string
	// found maps the path of each difference to a fingerprint of the live value.
	found map[string]string
}

func (c *comparison) run(manifest map[string]any, live *unstructured.Unstructured, manager string) map[string]any {
	own := ownership{fields: managedFields(live, appliedBy(manager))}
	shape := managedFields(live, func(metav1.ManagedFieldsEntry) bool { return true })
	result, _ := c.value(location{}, manifest, live.Object, true, own, shape)
	return result.(map[string]any)
}

// liveChanges returns the manifest as it would read if it described the live
// object: values another client changed are replaced by the live ones and
// removed values are left out. The second result is empty when nothing drifted.
func liveChanges(manifest map[string]any, live *unstructured.Unstructured, manager string, skipped [][]string, accepted map[string]string) (map[string]any, map[string]string) {
	c := &comparison{skipped: skipped, accepted: accepted, found: map[string]string{}}
	described := c.run(manifest, live, manager)
	if live.GetAPIVersion() == "v1" && live.GetKind() == "Secret" {
		c.secretStrings(manifest, described, live)
	}
	return described, c.found
}

// leavingComputed returns the manifest to apply to an existing object, without
// the computed fields that another manager owns now. Sending such a field, even
// with the other client's value, would make Terraform an owner of it again,
// and a controller that applies the field itself would then conflict.
func leavingComputed(manifest map[string]any, live *unstructured.Unstructured, manager string, skipped [][]string) map[string]any {
	if len(skipped) == 0 {
		return manifest
	}
	c := &comparison{skipped: skipped, handOver: true, found: map[string]string{}}
	return c.run(manifest, live, manager)
}

func (c *comparison) skip(at location) bool {
	for _, names := range c.skipped {
		if len(at.names) >= len(names) && slices.Equal(at.names[:len(names)], names) {
			return true
		}
	}
	return false
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
		at := location{}.field("stringData").field(key)
		delete(c.found, at.path)
		encoded, exists := stored[key].(string)
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if exists && err == nil && string(decoded) == want || c.skip(at) {
			out[key] = want
		} else if value, keep := c.differs(at, want, string(decoded), exists); keep {
			out[key] = value
		}
	}
	described["stringData"] = out
}

// value compares one manifest value with its live counterpart. shape is the
// ownership of all managers, which shows how the API merges each list.
func (c *comparison) value(at location, want, have any, exists bool, own ownership, shape map[string]any) (any, bool) {
	if ignoredPaths[at.path] || !c.handOver && c.skip(at) {
		return want, true
	}
	if own.all {
		// A map owned as a whole is still compared member by member: the
		// entry of a map whose fields another client removed looks the same.
		if members, isMap := want.(map[string]any); isMap && !c.handOver {
			return c.members(at, members, have, exists, ownership{}, shape)
		}
		return want, true
	}
	// Every manager lists an atomic field, which the API replaces as a whole,
	// without its members. It is compared as a whole too.
	atomic := at.path != "" && shape != nil && len(shape) == 0
	switch want := want.(type) {
	case map[string]any:
		if !atomic {
			return c.members(at, want, have, exists, own, shape)
		}
	case []any:
		haveList, _ := have.([]any)
		switch {
		case atomic:
		case hasPrefixedKey(shape, "k:") && allObjects(want):
			return c.keyedList(at, want, haveList, own, shape), true
		case hasPrefixedKey(shape, "v:"):
			return c.valueSet(at, want, haveList, own), true
		case allObjects(want) && len(want) == len(haveList):
			// A list no manager describes: elements pair by position.
			out := make([]any, len(want))
			for i := range want {
				out[i], _ = c.value(at.element(strconv.Itoa(i)), want[i], haveList[i], true, ownership{}, nil)
			}
			return out, true
		}
	}
	if c.handOver {
		// A computed field that another manager owns is left to it. One that
		// no manager owns yet, such as a key added to a computed map, is sent.
		return want, !c.skip(at) || shape == nil
	}
	// An empty list that is absent is no difference, like an empty map: a
	// client that rewrites the object leaves both out.
	if list, isList := want.([]any); exists && same(want, have) || !exists && isList && len(list) == 0 {
		return want, true
	}
	return c.differs(at, want, have, exists)
}

func (c *comparison) members(at location, want map[string]any, have any, exists bool, own ownership, shape map[string]any) (any, bool) {
	haveMap, _ := have.(map[string]any)
	out := make(map[string]any, len(want))
	for name, wanted := range want {
		live, found := haveMap[name]
		childShape, _ := shape["f:"+name].(map[string]any)
		if value, keep := c.value(at.field(name), wanted, live, found, own.child("f:"+name, childShape), childShape); keep {
			out[name] = value
		}
	}
	return out, exists || len(out) > 0 || len(want) == 0
}

// keyedList handles a list the API merges by key, such as containers by name.
// Elements other clients added are theirs and are not part of the result.
func (c *comparison) keyedList(at location, want, have []any, own ownership, shape map[string]any) []any {
	out := make([]any, 0, len(want))
	for i, element := range want {
		item := element.(map[string]any)
		key, keys := elementKey(item, own, shape)
		if key == "" {
			if value, keep := c.differs(at.element(strconv.Itoa(i)), item, nil, false); keep {
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
		if value, keep := c.value(at.element(describe(keys)), item, live, found, own.child(key, childShape), childShape); keep {
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
// finalizers. A missing element is named by its position in the manifest, not
// by its value, which may be sensitive.
func (c *comparison) valueSet(at location, want, have []any, own ownership) []any {
	out := make([]any, 0, len(want))
	for i, element := range want {
		encoded, _ := json.Marshal(element)
		_, owned := own.fields["v:"+string(encoded)]
		present := false
		for _, candidate := range have {
			present = present || same(element, candidate)
		}
		if owned || present {
			out = append(out, element)
		} else if value, keep := c.differs(at.element(strconv.Itoa(i)), element, nil, false); keep {
			out = append(out, value)
		}
	}
	return out
}

func (c *comparison) differs(at location, want, have any, exists bool) (any, bool) {
	if c.handOver {
		// Only reached for list elements the live object does not have.
		return want, true
	}
	print := absent
	if exists {
		encoded, _ := json.Marshal(have)
		sum := sha256.Sum256(encoded)
		print = hex.EncodeToString(sum[:8])
	}
	if c.accepted[at.path] == print {
		return want, true
	}
	c.found[at.path] = print
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

// split divides a FieldsV1 set into the fields a manifest value sets and the
// rest.
func split(fields map[string]any, value any) (in, out map[string]any) {
	in, out = map[string]any{}, map[string]any{}
	for key, node := range fields {
		members, _ := node.(map[string]any)
		member, set := memberOf(value, key)
		switch {
		case !set:
			out[key] = node
		case len(members) == 0:
			in[key] = node
		default:
			memberIn, memberOut := split(members, member)
			if len(memberIn) > 0 {
				in[key] = memberIn
			}
			if len(memberOut) > 0 {
				out[key] = memberOut
			}
		}
	}
	return in, out
}

// memberOf returns the part of a manifest value that a FieldsV1 key names,
// and whether the manifest sets it.
func memberOf(value any, key string) (any, bool) {
	list, _ := value.([]any)
	kind, text, _ := strings.Cut(key, ":")
	switch kind {
	case ".":
		return value, true
	case "f":
		object, _ := value.(map[string]any)
		member, found := object[text]
		return member, found
	case "k":
		var keys map[string]any
		if json.Unmarshal([]byte(text), &keys) != nil {
			return nil, false
		}
		for _, element := range list {
			if item, ok := element.(map[string]any); ok && hasKeys(item, keys, false) {
				return item, true
			}
		}
	case "v":
		var want any
		if json.Unmarshal([]byte(text), &want) != nil {
			return nil, false
		}
		for _, element := range list {
			if same(want, element) {
				return element, true
			}
		}
	case "i":
		if index, err := strconv.Atoi(text); err == nil && index >= 0 && index < len(list) {
			return list[index], true
		}
	}
	return nil, false
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
