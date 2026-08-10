// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// tfMap builds a types.Map for tests. A nil argument yields a null map; a non-nil
// but empty argument yields a known empty map. The distinction is load-bearing —
// see MIGRATION_FINDINGS_namespace_v1.md §3.
func tfMap(kv map[string]string) types.Map {
	if kv == nil {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(kv))
	for k, v := range kv {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}

func TestExpandMetadata(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   []MetadataModel
		want metav1.ObjectMeta
	}{
		{
			name: "empty slice returns zero ObjectMeta",
			in:   []MetadataModel{},
			want: metav1.ObjectMeta{},
		},
		{
			name: "name only, no labels or annotations",
			in: []MetadataModel{{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(nil),
				Annotations: tfMap(nil),
			}},
			want: metav1.ObjectMeta{Name: "demo"},
		},
		{
			name: "labels, no annotations",
			in: []MetadataModel{{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(map[string]string{"env": "demo"}),
				Annotations: tfMap(nil),
			}},
			want: metav1.ObjectMeta{
				Name:   "demo",
				Labels: map[string]string{"env": "demo"},
			},
		},
		{
			name: "annotations, no labels",
			in: []MetadataModel{{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(nil),
				Annotations: tfMap(map[string]string{"owner": "platform"}),
			}},
			want: metav1.ObjectMeta{
				Name:        "demo",
				Annotations: map[string]string{"owner": "platform"},
			},
		},
		{
			name: "labels and annotations",
			in: []MetadataModel{{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(map[string]string{"env": "demo", "team": "infra"}),
				Annotations: tfMap(map[string]string{"owner": "platform"}),
			}},
			want: metav1.ObjectMeta{
				Name:        "demo",
				Labels:      map[string]string{"env": "demo", "team": "infra"},
				Annotations: map[string]string{"owner": "platform"},
			},
		},
		{
			name: "generate_name and no name",
			in: []MetadataModel{{
				Name:         types.StringNull(),
				GenerateName: types.StringValue("demo-"),
				Labels:       tfMap(nil),
				Annotations:  tfMap(nil),
			}},
			want: metav1.ObjectMeta{GenerateName: "demo-"},
		},
		{
			name: "unknown name is not written through",
			in: []MetadataModel{{
				Name:         types.StringUnknown(),
				GenerateName: types.StringValue("demo-"),
				Labels:       tfMap(nil),
				Annotations:  tfMap(nil),
			}},
			want: metav1.ObjectMeta{GenerateName: "demo-"},
		},
		{
			// Divergence from SDKv2, pinned deliberately: structures.go guards with
			// `len(v) > 0`, so an explicit empty map leaves the field nil there and
			// non-nil-but-empty here. ObjectMeta tags both as omitempty, so the two
			// serialise identically and the API cannot tell them apart.
			name: "explicit empty maps produce empty, not nil",
			in: []MetadataModel{{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(map[string]string{}),
				Annotations: tfMap(map[string]string{}),
			}},
			want: metav1.ObjectMeta{
				Name:        "demo",
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, diags := expandMetadata(context.Background(), tc.in)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ObjectMeta mismatch\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

// TestExpandMapForPatch covers the null/unknown paths explicitly: types.Map.Elements()
// returns a defensive copy built from a possibly-nil internal map, and both len() and
// range over a nil map are safe in Go — so these return empty rather than panicking.
func TestExpandMapForPatch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   types.Map
		want map[string]interface{}
	}{
		{"null map", types.MapNull(types.StringType), map[string]interface{}{}},
		{"unknown map", types.MapUnknown(types.StringType), map[string]interface{}{}},
		{"empty map", tfMap(map[string]string{}), map[string]interface{}{}},
		{
			"populated map keeps values",
			tfMap(map[string]string{"env": "demo", "team": "infra"}),
			map[string]interface{}{"env": "demo", "team": "infra"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := expandMapForPatch(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestFlattenMetadata(t *testing.T) {
	t.Parallel()

	const internalLabel = "kubernetes.io/metadata.name"
	const lastApplied = "kubectl.kubernetes.io/last-applied-configuration"

	cases := []struct {
		name              string
		obj               metav1.ObjectMeta
		prior             []MetadataModel
		ignoreAnnotations []string
		ignoreLabels      []string
		wantLabels        types.Map
		wantAnnotations   types.Map
	}{
		{
			// findings §4 case 3
			name:            "no annotations and no labels",
			obj:             metav1.ObjectMeta{Name: "ns"},
			prior:           []MetadataModel{{Labels: tfMap(nil), Annotations: tfMap(nil)}},
			wantLabels:      tfMap(nil),
			wantAnnotations: tfMap(nil),
		},
		{
			// findings §4 cases 1 and 12 — the whole reason filtering exists
			name: "internal annotations and internal labels are dropped",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{internalLabel: "ns"},
				Annotations: map[string]string{lastApplied: "{}"},
			},
			prior:           []MetadataModel{{Labels: tfMap(nil), Annotations: tfMap(nil)}},
			wantLabels:      tfMap(nil),
			wantAnnotations: tfMap(nil),
		},
		{
			// findings §4 case 5 — declaring an internal key opts back into managing it
			name: "internal keys declared in prior state are kept",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{internalLabel: "ns", "other.kubernetes.io/x": "y"},
				Annotations: map[string]string{lastApplied: "{}", "extra.kubernetes.io/a": "b"},
			},
			prior: []MetadataModel{{
				Labels:      tfMap(map[string]string{internalLabel: "ns"}),
				Annotations: tfMap(map[string]string{lastApplied: "{}"}),
			}},
			wantLabels:      tfMap(map[string]string{internalLabel: "ns"}),
			wantAnnotations: tfMap(map[string]string{lastApplied: "{}"}),
		},
		{
			// findings §4 cases 2 and 10 — non-internal keys always survive, whether
			// the practitioner declared them or an external controller added them
			name: "mixed internal and non-internal keeps only non-internal",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{"env": "demo", internalLabel: "ns", "owner": "platform"},
				Annotations: map[string]string{"team": "infra", lastApplied: "{}"},
			},
			prior: []MetadataModel{{
				Labels:      tfMap(map[string]string{"env": "demo"}),
				Annotations: tfMap(map[string]string{"team": "infra"}),
			}},
			wantLabels:      tfMap(map[string]string{"env": "demo", "owner": "platform"}),
			wantAnnotations: tfMap(map[string]string{"team": "infra"}),
		},
		{
			// findings §4 case 8 — the two carve-outs inside *.kubernetes.io
			name: "app.kubernetes.io and service.beta.kubernetes.io are exempt",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{"app.kubernetes.io/name": "web", internalLabel: "ns"},
				Annotations: map[string]string{"service.beta.kubernetes.io/aws-load-balancer-type": "nlb"},
			},
			prior:           []MetadataModel{{Labels: tfMap(nil), Annotations: tfMap(nil)}},
			wantLabels:      tfMap(map[string]string{"app.kubernetes.io/name": "web"}),
			wantAnnotations: tfMap(map[string]string{"service.beta.kubernetes.io/aws-load-balancer-type": "nlb"}),
		},
		{
			// findings §4 case 6
			name: "ignore list drops matching keys",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{"env": "demo", "cost-center": "x"},
				Annotations: map[string]string{"keep": "1", "drop-me": "2"},
			},
			prior: []MetadataModel{{
				Labels:      tfMap(map[string]string{"env": "demo"}),
				Annotations: tfMap(map[string]string{"keep": "1"}),
			}},
			ignoreLabels:      []string{"cost-center"},
			ignoreAnnotations: []string{"drop-me"},
			wantLabels:        tfMap(map[string]string{"env": "demo"}),
			wantAnnotations:   tfMap(map[string]string{"keep": "1"}),
		},
		{
			// findings §4 case 7 — declaring a key beats the ignore list too
			name: "ignore list does not drop keys declared in prior state",
			obj: metav1.ObjectMeta{
				Name:   "ns",
				Labels: map[string]string{"env": "demo", "cost-center": "x"},
			},
			prior: []MetadataModel{{
				Labels:      tfMap(map[string]string{"env": "demo", "cost-center": "x"}),
				Annotations: tfMap(nil),
			}},
			ignoreLabels:    []string{"cost-center"},
			wantLabels:      tfMap(map[string]string{"env": "demo", "cost-center": "x"}),
			wantAnnotations: tfMap(nil),
		},
		{
			// findings §4 case 9 — regexp.MatchString is UNANCHORED. "env" matches
			// "environment". Pinned so nobody "fixes" it into an anchored match.
			name: "ignore patterns are unanchored and over-match",
			obj: metav1.ObjectMeta{
				Name:   "ns",
				Labels: map[string]string{"env": "demo", "environment": "prod"},
			},
			prior: []MetadataModel{{
				Labels:      tfMap(map[string]string{"env": "demo"}),
				Annotations: tfMap(nil),
			}},
			ignoreLabels:    []string{"env"},
			wantLabels:      tfMap(map[string]string{"env": "demo"}),
			wantAnnotations: tfMap(nil),
		},
		{
			// findings §4 case 10 — out-of-band drift must survive, or Read would
			// silently hide changes made outside Terraform
			name: "out-of-band non-internal label survives a null prior",
			obj: metav1.ObjectMeta{
				Name:   "ns",
				Labels: map[string]string{"owner": "platform", internalLabel: "ns"},
			},
			prior:           []MetadataModel{{Labels: tfMap(nil), Annotations: tfMap(nil)}},
			wantLabels:      tfMap(map[string]string{"owner": "platform"}),
			wantAnnotations: tfMap(nil),
		},
		{
			// findings §4 case 11 — an explicit `labels = {}` stays empty, it does
			// not collapse to null
			name: "explicit empty prior stays empty rather than becoming null",
			obj: metav1.ObjectMeta{
				Name:   "ns",
				Labels: map[string]string{internalLabel: "ns"},
			},
			prior: []MetadataModel{{
				Labels:      tfMap(map[string]string{}),
				Annotations: tfMap(nil),
			}},
			wantLabels:      tfMap(map[string]string{}),
			wantAnnotations: tfMap(nil),
		},
		{
			// import: prior state is empty, so nothing is exempt from filtering
			name: "empty prior slice filters everything not exempt",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{internalLabel: "ns", "env": "demo"},
				Annotations: map[string]string{lastApplied: "{}"},
			},
			prior:           nil,
			wantLabels:      tfMap(map[string]string{"env": "demo"}),
			wantAnnotations: tfMap(nil),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, diags := flattenMetadata(context.Background(), tc.obj, tc.prior,
				tc.ignoreAnnotations, tc.ignoreLabels)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if len(got) != 1 {
				t.Fatalf("expected exactly one metadata element, got %d", len(got))
			}

			if !got[0].Labels.Equal(tc.wantLabels) {
				t.Errorf("labels mismatch\n got: %s\nwant: %s", got[0].Labels, tc.wantLabels)
			}
			if !got[0].Annotations.Equal(tc.wantAnnotations) {
				t.Errorf("annotations mismatch\n got: %s\nwant: %s", got[0].Annotations, tc.wantAnnotations)
			}
		})
	}
}

// TestFlattenMetadataScalarFields covers the non-map fields, including the rule that
// an absent generate_name stays null rather than becoming "".
func TestFlattenMetadataScalarFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		obj              metav1.ObjectMeta
		wantGenerateName types.String
	}{
		{
			name:             "generate_name absent stays null",
			obj:              metav1.ObjectMeta{Name: "ns", Generation: 3, ResourceVersion: "42", UID: "abc"},
			wantGenerateName: types.StringNull(),
		},
		{
			name: "generate_name present is carried through",
			obj: metav1.ObjectMeta{
				Name: "ns-x1y2", GenerateName: "ns-", Generation: 3, ResourceVersion: "42", UID: "abc",
			},
			wantGenerateName: types.StringValue("ns-"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, diags := flattenMetadata(context.Background(), tc.obj, nil, nil, nil)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			if !got[0].GenerateName.Equal(tc.wantGenerateName) {
				t.Errorf("generate_name = %s, want %s", got[0].GenerateName, tc.wantGenerateName)
			}
			if got[0].Name.ValueString() != tc.obj.Name {
				t.Errorf("name = %s, want %s", got[0].Name, tc.obj.Name)
			}
			if got[0].Generation.ValueInt64() != tc.obj.Generation {
				t.Errorf("generation = %s, want %d", got[0].Generation, tc.obj.Generation)
			}
			if got[0].ResourceVersion.ValueString() != tc.obj.ResourceVersion {
				t.Errorf("resource_version = %s, want %s", got[0].ResourceVersion, tc.obj.ResourceVersion)
			}
			if got[0].UID.ValueString() != string(tc.obj.UID) {
				t.Errorf("uid = %s, want %s", got[0].UID, tc.obj.UID)
			}
		})
	}
}
