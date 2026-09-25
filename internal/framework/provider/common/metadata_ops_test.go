// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// tfMap builds a types.Map for tests. A nil argument yields a null map; a non-nil
// but empty argument yields a known empty map.
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

func TestBaseMetadataConversion(t *testing.T) {
	ctx := context.Background()
	apiMetadata := metav1.ObjectMeta{
		Name: "example", Namespace: "team", GenerateName: "example-",
		Generation: 3, ResourceVersion: "42", UID: "uid-1",
		Labels:      map[string]string{"env": "test"},
		Annotations: map[string]string{"owner": "team"},
	}
	metadata, diags := FlattenBaseMetadata(ctx, apiMetadata, MetadataBase{}, nil, nil)
	if diags.HasError() {
		t.Fatal(diags)
	}
	want := MetadataBase{
		Name: types.StringValue("example"), Generation: types.Int64Value(3),
		ResourceVersion: types.StringValue("42"), UID: types.StringValue("uid-1"),
		Labels:      tfMap(apiMetadata.Labels),
		Annotations: tfMap(apiMetadata.Annotations),
	}
	if !reflect.DeepEqual(metadata, want) {
		t.Fatalf("metadata = %#v, want %#v", metadata, want)
	}
	state := tfsdk.State{Schema: schema.Schema{
		Attributes: MetadataSchema("thing", false).NestedObject.Attributes,
	}}
	if diags := state.Set(ctx, metadata); diags.HasError() {
		t.Fatalf("base schema rejected flattened metadata: %v", diags)
	}
	expanded, diags := ExpandBaseMetadata(ctx, metadata)
	if diags.HasError() {
		t.Fatal(diags)
	}
	wantExpanded := metav1.ObjectMeta{
		Name: "example", Labels: apiMetadata.Labels, Annotations: apiMetadata.Annotations,
	}
	if !reflect.DeepEqual(expanded, wantExpanded) {
		t.Errorf("ObjectMeta = %#v, want %#v", expanded, wantExpanded)
	}
}

func TestExpandMetadataStringStates(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		value types.String
		want  string
	}{
		{"null", types.StringNull(), ""},
		{"unknown", types.StringUnknown(), ""},
		{"empty", types.StringValue(""), ""},
		{"known", types.StringValue("example"), "example"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			input := []NamespacedMetadataModel{{
				MetadataModel: MetadataModel{
					MetadataBase: MetadataBase{Name: testCase.value},
					GenerateName: testCase.value,
				},
				Namespace: testCase.value,
			}}
			expanded, diags := ExpandNamespacedMetadata(context.Background(), input)
			if diags.HasError() {
				t.Fatal(diags)
			}
			payload, err := json.Marshal(expanded)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(payload, &fields); err != nil {
				t.Fatal(err)
			}
			for field, value := range map[string]types.String{
				"name": input[0].Name, "generateName": input[0].GenerateName, "namespace": input[0].Namespace,
			} {
				if !value.Equal(testCase.value) {
					t.Errorf("expansion changed Terraform %s: %v", field, value)
				}
				encoded, exists := fields[field]
				if testCase.want == "" {
					if exists {
						t.Errorf("%s must be omitted from JSON, got %s", field, encoded)
					}
				} else if string(encoded) != `"`+testCase.want+`"` {
					t.Errorf("%s = %s, want %q", field, encoded, testCase.want)
				}
			}
		})
	}
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
			in: []MetadataModel{{MetadataBase: MetadataBase{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(nil),
				Annotations: tfMap(nil),
			}}},
			want: metav1.ObjectMeta{Name: "demo"},
		},
		{
			name: "labels, no annotations",
			in: []MetadataModel{{MetadataBase: MetadataBase{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(map[string]string{"env": "demo"}),
				Annotations: tfMap(nil),
			}}},
			want: metav1.ObjectMeta{
				Name:   "demo",
				Labels: map[string]string{"env": "demo"},
			},
		},
		{
			name: "annotations, no labels",
			in: []MetadataModel{{MetadataBase: MetadataBase{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(nil),
				Annotations: tfMap(map[string]string{"owner": "platform"}),
			}}},
			want: metav1.ObjectMeta{
				Name:        "demo",
				Annotations: map[string]string{"owner": "platform"},
			},
		},
		{
			name: "labels and annotations",
			in: []MetadataModel{{MetadataBase: MetadataBase{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(map[string]string{"env": "demo", "team": "infra"}),
				Annotations: tfMap(map[string]string{"owner": "platform"}),
			}}},
			want: metav1.ObjectMeta{
				Name:        "demo",
				Labels:      map[string]string{"env": "demo", "team": "infra"},
				Annotations: map[string]string{"owner": "platform"},
			},
		},
		{
			name: "generate_name and no name",
			in: []MetadataModel{{
				MetadataBase: MetadataBase{
					Name:        types.StringNull(),
					Labels:      tfMap(nil),
					Annotations: tfMap(nil),
				},
				GenerateName: types.StringValue("demo-"),
			}},
			want: metav1.ObjectMeta{GenerateName: "demo-"},
		},
		{
			name: "unknown name is not written through",
			in: []MetadataModel{{
				MetadataBase: MetadataBase{
					Name:        types.StringUnknown(),
					Labels:      tfMap(nil),
					Annotations: tfMap(nil),
				},
				GenerateName: types.StringValue("demo-"),
			}},
			want: metav1.ObjectMeta{GenerateName: "demo-"},
		},
		{
			// Divergence from SDKv2, pinned deliberately: structures.go guards with
			// `len(v) > 0`, so an explicit empty map leaves the field nil there and
			// non-nil-but-empty here. ObjectMeta tags both as omitempty, so the two
			// serialise identically and the API cannot tell them apart.
			name: "explicit empty maps produce empty, not nil",
			in: []MetadataModel{{MetadataBase: MetadataBase{
				Name:        types.StringValue("demo"),
				Labels:      tfMap(map[string]string{}),
				Annotations: tfMap(map[string]string{}),
			}}},
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

			got, diags := ExpandMetadata(context.Background(), tc.in)
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

			got := ExpandMapForPatch(tc.in)
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
			name:            "no annotations and no labels",
			obj:             metav1.ObjectMeta{Name: "ns"},
			prior:           []MetadataModel{{MetadataBase: MetadataBase{Labels: tfMap(nil), Annotations: tfMap(nil)}}},
			wantLabels:      tfMap(nil),
			wantAnnotations: tfMap(nil),
		},
		{
			name: "internal annotations and internal labels are dropped",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{internalLabel: "ns"},
				Annotations: map[string]string{lastApplied: "{}"},
			},
			prior:           []MetadataModel{{MetadataBase: MetadataBase{Labels: tfMap(nil), Annotations: tfMap(nil)}}},
			wantLabels:      tfMap(nil),
			wantAnnotations: tfMap(nil),
		},
		{
			// Declaring an internal key opts back into managing it.
			name: "internal keys declared in prior state are kept",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{internalLabel: "ns", "other.kubernetes.io/x": "y"},
				Annotations: map[string]string{lastApplied: "{}", "extra.kubernetes.io/a": "b"},
			},
			prior: []MetadataModel{{MetadataBase: MetadataBase{
				Labels:      tfMap(map[string]string{internalLabel: "ns"}),
				Annotations: tfMap(map[string]string{lastApplied: "{}"}),
			}}},
			wantLabels:      tfMap(map[string]string{internalLabel: "ns"}),
			wantAnnotations: tfMap(map[string]string{lastApplied: "{}"}),
		},
		{
			// Non-internal keys always survive, whether
			// the practitioner declared them or an external controller added them
			name: "mixed internal and non-internal keeps only non-internal",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{"env": "demo", internalLabel: "ns", "owner": "platform"},
				Annotations: map[string]string{"team": "infra", lastApplied: "{}"},
			},
			prior: []MetadataModel{{MetadataBase: MetadataBase{
				Labels:      tfMap(map[string]string{"env": "demo"}),
				Annotations: tfMap(map[string]string{"team": "infra"}),
			}}},
			wantLabels:      tfMap(map[string]string{"env": "demo", "owner": "platform"}),
			wantAnnotations: tfMap(map[string]string{"team": "infra"}),
		},
		{
			name: "app.kubernetes.io and service.beta.kubernetes.io are exempt",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{"app.kubernetes.io/name": "web", internalLabel: "ns"},
				Annotations: map[string]string{"service.beta.kubernetes.io/aws-load-balancer-type": "nlb"},
			},
			prior:           []MetadataModel{{MetadataBase: MetadataBase{Labels: tfMap(nil), Annotations: tfMap(nil)}}},
			wantLabels:      tfMap(map[string]string{"app.kubernetes.io/name": "web"}),
			wantAnnotations: tfMap(map[string]string{"service.beta.kubernetes.io/aws-load-balancer-type": "nlb"}),
		},
		{
			name: "ignore list drops matching keys",
			obj: metav1.ObjectMeta{
				Name:        "ns",
				Labels:      map[string]string{"env": "demo", "cost-center": "x"},
				Annotations: map[string]string{"keep": "1", "drop-me": "2"},
			},
			prior: []MetadataModel{{MetadataBase: MetadataBase{
				Labels:      tfMap(map[string]string{"env": "demo"}),
				Annotations: tfMap(map[string]string{"keep": "1"}),
			}}},
			ignoreLabels:      []string{"cost-center"},
			ignoreAnnotations: []string{"drop-me"},
			wantLabels:        tfMap(map[string]string{"env": "demo"}),
			wantAnnotations:   tfMap(map[string]string{"keep": "1"}),
		},
		{
			// Declaring a key beats the ignore list too.
			name: "ignore list does not drop keys declared in prior state",
			obj: metav1.ObjectMeta{
				Name:   "ns",
				Labels: map[string]string{"env": "demo", "cost-center": "x"},
			},
			prior: []MetadataModel{{MetadataBase: MetadataBase{
				Labels:      tfMap(map[string]string{"env": "demo", "cost-center": "x"}),
				Annotations: tfMap(nil),
			}}},
			ignoreLabels:    []string{"cost-center"},
			wantLabels:      tfMap(map[string]string{"env": "demo", "cost-center": "x"}),
			wantAnnotations: tfMap(nil),
		},
		{
			// regexp.MatchString is UNANCHORED. "env" matches
			// "environment". Pinned so nobody "fixes" it into an anchored match.
			name: "ignore patterns are unanchored and over-match",
			obj: metav1.ObjectMeta{
				Name:   "ns",
				Labels: map[string]string{"env": "demo", "environment": "prod"},
			},
			prior: []MetadataModel{{MetadataBase: MetadataBase{
				Labels:      tfMap(map[string]string{"env": "demo"}),
				Annotations: tfMap(nil),
			}}},
			ignoreLabels:    []string{"env"},
			wantLabels:      tfMap(map[string]string{"env": "demo"}),
			wantAnnotations: tfMap(nil),
		},
		{
			// Out-of-band drift must survive, or Read would
			// silently hide changes made outside Terraform
			name: "out-of-band non-internal label survives a null prior",
			obj: metav1.ObjectMeta{
				Name:   "ns",
				Labels: map[string]string{"owner": "platform", internalLabel: "ns"},
			},
			prior:           []MetadataModel{{MetadataBase: MetadataBase{Labels: tfMap(nil), Annotations: tfMap(nil)}}},
			wantLabels:      tfMap(map[string]string{"owner": "platform"}),
			wantAnnotations: tfMap(nil),
		},
		{
			// An explicit `labels = {}` stays empty, it does
			// not collapse to null
			name: "explicit empty prior stays empty rather than becoming null",
			obj: metav1.ObjectMeta{
				Name:   "ns",
				Labels: map[string]string{internalLabel: "ns"},
			},
			prior: []MetadataModel{{MetadataBase: MetadataBase{
				Labels:      tfMap(map[string]string{}),
				Annotations: tfMap(nil),
			}}},
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

			got, diags := FlattenMetadata(context.Background(), tc.obj, tc.prior,
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

			got, diags := FlattenMetadata(context.Background(), tc.obj, nil, nil, nil)
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

// TestMetadataPatchOps pins the skip rule that keeps an update to one metadata map from
// destroying keys the provider does not manage in the other.
func TestMetadataPatchOps(t *testing.T) {
	populated := tfMap(map[string]string{"a": "1"})

	for _, tc := range []struct {
		name              string
		stateAnn, planAnn types.Map
		stateLbl, planLbl types.Map
		expected          string
	}{
		{
			name:     "nothing managed on either side sends no patch",
			stateAnn: tfMap(nil), planAnn: tfMap(nil),
			stateLbl: tfMap(nil), planLbl: tfMap(nil),
			expected: `[]`,
		},
		{
			// null and {} differ in Terraform but no key to change in Kubernetes.
			name:     "null to empty map sends no patch",
			stateAnn: tfMap(nil), planAnn: tfMap(map[string]string{}),
			stateLbl: tfMap(map[string]string{}), planLbl: tfMap(nil),
			expected: `[]`,
		},
		{
			// The regression: a label-only update must not touch annotations.
			name:     "label change leaves unmanaged annotations alone",
			stateAnn: tfMap(nil), planAnn: tfMap(nil),
			stateLbl: populated, planLbl: tfMap(map[string]string{"a": "2"}),
			expected: `[{"op":"replace","path":"/metadata/labels/a","value":"2"}]`,
		},
		{
			name:     "removing every managed key removes them one by one",
			stateAnn: populated, planAnn: tfMap(nil),
			stateLbl: tfMap(nil), planLbl: tfMap(nil),
			expected: `[{"op":"remove","path":"/metadata/annotations/a"}]`,
		},
		{
			// Both maps changed: each is diffed, under its own path prefix. The operation
			// kinds themselves are TestDiffStringMap's contract, not this one's.
			name:     "changes to both maps are emitted together",
			stateAnn: populated, planAnn: tfMap(map[string]string{"a": "2"}),
			stateLbl: tfMap(map[string]string{"x": "1"}), planLbl: tfMap(map[string]string{}),
			expected: `[{"op":"replace","path":"/metadata/annotations/a","value":"2"},` +
				`{"op":"remove","path":"/metadata/labels/x"}]`,
		},
		{
			// KNOWN LIMITATION, pinned so a future fix changes it deliberately: the first
			// managed key still replaces the whole object, dropping keys owned elsewhere.
			name:     "first managed key still replaces the whole map",
			stateAnn: tfMap(nil), planAnn: populated,
			stateLbl: tfMap(nil), planLbl: tfMap(nil),
			expected: `[{"op":"add","path":"/metadata/annotations","value":{"a":"1"}}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := MetadataModel{MetadataBase: MetadataBase{Annotations: tc.stateAnn, Labels: tc.stateLbl}}
			plan := MetadataModel{MetadataBase: MetadataBase{Annotations: tc.planAnn, Labels: tc.planLbl}}

			result, err := MetadataPatchOps("/metadata/", state, plan).MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}

			// Compared as decoded JSON: the operations marshal their fields in struct
			// order, which is not part of the contract.
			var resultOps, expectedOps []map[string]interface{}
			if err := json.Unmarshal(result, &resultOps); err != nil {
				t.Fatalf("decoding %s: %s", result, err)
			}
			if err := json.Unmarshal([]byte(tc.expected), &expectedOps); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(resultOps, expectedOps) {
				t.Errorf("patch = %s, expected %s", result, tc.expected)
			}
		})
	}
}

func TestNamespacedSchemaMatchesClusterScoped(t *testing.T) {
	for _, generatable := range []bool{true, false} {
		clusterAttrs := MetadataSchema("thing", generatable).NestedObject.Attributes
		nsAttrs := NamespacedMetadataSchema("thing", generatable).NestedObject.Attributes

		if _, ok := nsAttrs["namespace"]; !ok {
			t.Fatalf("generatable=%v: namespaced schema is missing the namespace attribute", generatable)
		}
		if len(nsAttrs) != len(clusterAttrs)+1 {
			t.Fatalf("generatable=%v: namespaced schema has %d attributes, want %d (cluster-scoped + namespace)",
				generatable, len(nsAttrs), len(clusterAttrs)+1)
		}
		for name := range clusterAttrs {
			if _, ok := nsAttrs[name]; !ok {
				t.Errorf("generatable=%v: namespaced schema is missing %q", generatable, name)
			}
		}

		// generate_name is a variant of both, so it must track the flag on both paths.
		_, clusterHas := clusterAttrs["generate_name"]
		_, nsHas := nsAttrs["generate_name"]
		if clusterHas != generatable || nsHas != generatable {
			t.Errorf("generatable=%v: generate_name present cluster=%v namespaced=%v, want %v on both",
				generatable, clusterHas, nsHas, generatable)
		}
	}
}

func TestMetadataModelsRoundTrip(t *testing.T) {
	ctx := context.Background()
	for _, variant := range []struct {
		name  string
		block schema.ListNestedBlock
		model any
	}{
		{"base", MetadataSchema("thing", false), new(MetadataBase)},
		{"generated", MetadataSchema("thing", true), new(MetadataModel)},
		{"namespaced", NamespacedMetadataSchema("thing", true), new(NamespacedMetadataModel)},
	} {
		for mapName, mapValue := range map[string]types.Map{
			"populated": tfMap(map[string]string{"owner": "test"}),
			"null":      tfMap(nil),
			"empty":     tfMap(map[string]string{}),
			"unknown":   types.MapUnknown(types.StringType),
		} {
			t.Run(variant.name+"/"+mapName, func(t *testing.T) {
				values := map[string]attr.Value{
					"annotations":      mapValue,
					"generation":       types.Int64Value(7),
					"labels":           mapValue,
					"name":             types.StringValue("thing"),
					"resource_version": types.StringValue("123"),
					"uid":              types.StringValue("uid-1"),
				}
				attributes := variant.block.NestedObject.Attributes
				if _, exists := attributes["generate_name"]; exists {
					values["generate_name"] = types.StringValue("prefix-")
				}
				if _, exists := attributes["namespace"]; exists {
					values["namespace"] = types.StringValue("team-a")
				}
				attributeTypes := make(map[string]attr.Type, len(attributes))
				for name, attribute := range attributes {
					attributeTypes[name] = attribute.GetType()
				}
				object, diags := types.ObjectValue(attributeTypes, values)
				if diags.HasError() {
					t.Fatal(diags)
				}
				raw, err := object.ToTerraformValue(ctx)
				if err != nil {
					t.Fatal(err)
				}
				state := tfsdk.State{Schema: schema.Schema{Attributes: attributes}, Raw: raw}
				if diags := state.Get(ctx, variant.model); diags.HasError() {
					t.Fatalf("decode: %v", diags)
				}
				if diags := state.Set(ctx, variant.model); diags.HasError() {
					t.Fatalf("encode: %v", diags)
				}
				if !state.Raw.Equal(raw) {
					t.Errorf("round trip = %s, want %s", state.Raw, raw)
				}
			})
		}
	}
}

func TestExpandNamespacedMetadata(t *testing.T) {
	testCases := []struct {
		name  string
		in    []NamespacedMetadataModel
		want  string
		wantN string
	}{
		{"empty input", nil, "", ""},
		{
			"namespace set",
			[]NamespacedMetadataModel{{MetadataModel: MetadataModel{MetadataBase: MetadataBase{Name: types.StringValue("r")}}, Namespace: types.StringValue("team-a")}},
			"r", "team-a",
		},
		{
			"namespace null leaves ObjectMeta.Namespace empty",
			[]NamespacedMetadataModel{{MetadataModel: MetadataModel{MetadataBase: MetadataBase{Name: types.StringValue("r")}}, Namespace: types.StringNull()}},
			"r", "",
		},
		{
			"namespace unknown leaves ObjectMeta.Namespace empty",
			[]NamespacedMetadataModel{{MetadataModel: MetadataModel{MetadataBase: MetadataBase{Name: types.StringValue("r")}}, Namespace: types.StringUnknown()}},
			"r", "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := ExpandNamespacedMetadata(context.Background(), tc.in)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got.Name != tc.want {
				t.Errorf("Name = %q, want %q", got.Name, tc.want)
			}
			if got.Namespace != tc.wantN {
				t.Errorf("Namespace = %q, want %q", got.Namespace, tc.wantN)
			}
		})
	}
}

func TestFlattenNamespacedMetadata(t *testing.T) {
	objMeta := metav1.ObjectMeta{
		Name:      "r",
		Namespace: "team-a",
		Labels: map[string]string{
			"env":                         "demo",
			"kubernetes.io/metadata.name": "team-a", // internal: filtered unless declared
		},
	}

	got, diags := FlattenNamespacedMetadata(context.Background(), objMeta, nil, nil, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(got) != 1 {
		t.Fatalf("got %d elements, want 1", len(got))
	}
	if got[0].Namespace != types.StringValue("team-a") {
		t.Errorf("Namespace = %v, want %q", got[0].Namespace, "team-a")
	}
	// Filtering is FlattenMetadata's job; this asserts the delegation actually happened.
	if !got[0].Labels.Equal(tfMap(map[string]string{"env": "demo"})) {
		t.Errorf("Labels = %v, want the internal key filtered out", got[0].Labels)
	}

	prior := []NamespacedMetadataModel{{MetadataModel: MetadataModel{MetadataBase: MetadataBase{
		Labels:      tfMap(map[string]string{"kubernetes.io/metadata.name": "team-a"}),
		Annotations: tfMap(map[string]string{}),
	}}}}
	objMeta.Annotations = map[string]string{"ignored": "external"}
	got, diags = FlattenNamespacedMetadata(context.Background(), objMeta, prior, []string{"ignored"}, []string{"env"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !got[0].Labels.Equal(prior[0].Labels) || !got[0].Annotations.Equal(prior[0].Annotations) {
		t.Errorf("prior metadata or ignore filters were not preserved: %v", got[0])
	}
}
