// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podtemplate

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	legacyschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// The retained SDKv2 aliases share the original versioned resource schemas.
var templateOwners = map[string]Options{
	"kubernetes_deployment":   {RestartPolicyAlways: true},
	"kubernetes_daemonset":    {},
	"kubernetes_stateful_set": {},
}

// The only nesting changes: SDKv2 blocks that are now assignment attributes.
var assignmentSyntaxPaths = []string{
	"spec.container.resources",
	"spec.image_pull_secrets",
	"spec.init_container.resources",
	"spec.readiness_gate",
}

var sdkTemplateForceNewPaths = []string{
	"spec.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference.match_fields",
	"spec.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference.match_fields.key",
	"spec.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference.match_fields.operator",
	"spec.affinity.node_affinity.preferred_during_scheduling_ignored_during_execution.preference.match_fields.values",
	"spec.affinity.node_affinity.required_during_scheduling_ignored_during_execution.node_selector_term.match_fields",
	"spec.affinity.node_affinity.required_during_scheduling_ignored_during_execution.node_selector_term.match_fields.key",
	"spec.affinity.node_affinity.required_during_scheduling_ignored_during_execution.node_selector_term.match_fields.operator",
	"spec.affinity.node_affinity.required_during_scheduling_ignored_during_execution.node_selector_term.match_fields.values",
	"spec.volume.azure_file.secret_namespace",
	"spec.volume.ephemeral.volume_claim_template.spec.access_modes",
	"spec.volume.ephemeral.volume_claim_template.spec.resources.limits",
	"spec.volume.ephemeral.volume_claim_template.spec.selector",
	"spec.volume.ephemeral.volume_claim_template.spec.selector.match_expressions",
	"spec.volume.ephemeral.volume_claim_template.spec.selector.match_expressions.key",
	"spec.volume.ephemeral.volume_claim_template.spec.selector.match_expressions.operator",
	"spec.volume.ephemeral.volume_claim_template.spec.selector.match_expressions.values",
	"spec.volume.ephemeral.volume_claim_template.spec.selector.match_labels",
	"spec.volume.ephemeral.volume_claim_template.spec.storage_class_name",
	"spec.volume.ephemeral.volume_claim_template.spec.volume_mode",
	"spec.volume.ephemeral.volume_claim_template.spec.volume_name",
}

func sdkTemplateSpec(t *testing.T, name string) *legacyschema.Schema {
	t.Helper()
	r, ok := kubernetes.Provider().ResourcesMap[name]
	if !ok {
		t.Fatalf("SDKv2 resource %s is not registered", name)
	}
	return r.Schema["spec"].Elem.(*legacyschema.Resource).Schema["template"].Elem.(*legacyschema.Resource).Schema["spec"]
}

func TestSpecBlockValidateImplementation(t *testing.T) {
	ctx := context.Background()
	for _, options := range []Options{{}, {RestartPolicyAlways: true}} {
		s := schema.Schema{Blocks: map[string]schema.Block{"spec": SpecBlock(options)}}
		if diagnostics := s.ValidateImplementation(ctx); diagnostics.HasError() {
			t.Fatalf("%+v: %v", options, diagnostics)
		}
	}
}

// Protocol-level persisted type equality with every SDKv2 name, through the
// real Framework and SDKv2 GetProviderSchema implementations.
func TestSpecBlockProtocolPersistedTypeMatchesSDK(t *testing.T) {
	ctx := context.Background()
	sdkResponse, err := legacyschema.NewGRPCProviderServer(kubernetes.Provider()).GetProviderSchema(ctx, &tfprotov5.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for name, options := range templateOwners {
		t.Run(name, func(t *testing.T) {
			sdkSpec := protocol5Block(t, sdkResponse.ResourceSchemas[name].Block, "spec", "template", "spec")
			frameworkSpec := protocol6Block(t, frameworkProtocolSchema(t, options), "spec")
			if !sdkSpec.ValueType().Equal(frameworkSpec.ValueType()) {
				t.Fatalf("persisted type differs\nSDKv2:     %s\nFramework: %s", sdkSpec.ValueType(), frameworkSpec.ValueType())
			}
			if sdkSpec.Nesting != tfprotov5.SchemaNestedBlockNestingModeList || frameworkSpec.Nesting != tfprotov6.SchemaNestedBlockNestingModeList {
				t.Fatalf("spec nesting changed: SDKv2 %v, Framework %v", sdkSpec.Nesting, frameworkSpec.Nesting)
			}
			sdkBlocks, frameworkBlocks := map[string]bool{}, map[string]bool{}
			protocol5BlockPaths(sdkSpec.Block, "spec", sdkBlocks)
			protocol6BlockPaths(frameworkSpec.Block, "spec", frameworkBlocks)
			var converted []string
			for p := range sdkBlocks {
				if !frameworkBlocks[p] {
					converted = append(converted, p)
				}
			}
			for p := range frameworkBlocks {
				if !sdkBlocks[p] {
					t.Errorf("Framework introduced block %s", p)
				}
			}
			sort.Strings(converted)
			if strings.Join(converted, ",") != strings.Join(assignmentSyntaxPaths, ",") {
				t.Fatalf("block-to-attribute conversions = %v, want %v", converted, assignmentSyntaxPaths)
			}
		})
	}
}

type schemaPathInfo struct {
	kind                         string
	required, optional, computed bool
	defaultValue                 attr.Value
	forceNew, validated, inline  bool
	block                        bool
	listValidators               []validator.List
	description                  string
}

func TestSpecBlockSchemaMatchesSDK(t *testing.T) {
	ctx := context.Background()
	for name, options := range templateOwners {
		t.Run(name, func(t *testing.T) {
			sdk := map[string]*legacyschema.Schema{}
			collectSDKPaths("spec", sdkTemplateSpec(t, name), sdk)
			native := map[string]schemaPathInfo{}
			block := SpecBlock(options)
			native["spec"] = schemaPathInfo{kind: "List", block: true, listValidators: block.Validators, description: block.Description}
			collectNativeObject(ctx, "spec", block.NestedObject.Attributes, block.NestedObject.Blocks, native)

			var required, optional, computed, defaulted int
			var forceNew []string
			for p, legacy := range sdk {
				info, ok := native[p]
				if !ok {
					t.Errorf("%s: missing from native schema", p)
					continue
				}
				if p == "spec" {
					continue
				}
				if want := sdkKind(legacy); info.kind != want {
					t.Errorf("%s: kind %s, want %s", p, info.kind, want)
				}
				if legacy.Required {
					required++
				} else {
					optional++
				}
				if legacy.Computed {
					computed++
				}
				if legacy.Default != nil {
					defaulted++
				}
				if legacy.ForceNew {
					forceNew = append(forceNew, p)
				}
				if info.forceNew != legacy.ForceNew {
					t.Errorf("%s: ForceNew %t, want %t", p, info.forceNew, legacy.ForceNew)
				}
				if info.inline {
					continue
				}
				if info.kind == "List" {
					checkListCardinality(ctx, t, p, info, legacy)
				}
				if info.kind == "List" || info.kind == "Set" || info.kind == "Map" {
					if legacy.Computed != (info.computed && info.defaultValue == nil) {
						t.Errorf("%s: SDK computed ownership %t not preserved (computed %t, default %v)", p, legacy.Computed, info.computed, info.defaultValue)
					}
					if info.defaultValue != nil && !info.defaultValue.IsNull() {
						t.Errorf("%s: compatibility collection default must remain null, got %v", p, info.defaultValue)
					}
					if !info.block && info.required != legacy.Required {
						t.Errorf("%s: required %t, want %t", p, info.required, legacy.Required)
					}
					continue
				}
				if info.required != legacy.Required {
					t.Errorf("%s: required %t, want %t", p, info.required, legacy.Required)
				}
				if legacy.Computed != (info.computed && info.defaultValue == nil) {
					t.Errorf("%s: SDK computed ownership %t not preserved (computed %t, default %v)", p, legacy.Computed, info.computed, info.defaultValue)
				}
				if (legacy.ValidateFunc != nil || legacy.ValidateDiagFunc != nil) && !info.validated {
					t.Errorf("%s: SDKv2 validation has no native validator", p)
				}
				if legacy.Default != nil {
					if want := sdkDefault(legacy); info.defaultValue == nil || !info.defaultValue.Equal(want) {
						t.Errorf("%s: default %v, want %v", p, info.defaultValue, want)
					}
				} else if info.defaultValue != nil && !zeroValue(info.defaultValue) {
					t.Errorf("%s: nonzero default %v where SDKv2 used the zero value", p, info.defaultValue)
				}
			}
			for p, info := range native {
				if _, ok := sdk[p]; !ok && !strings.HasSuffix(p, ".*") {
					t.Errorf("%s: native path is not an SDKv2 path", p)
				}
				if info.forceNew {
					if legacy, ok := sdk[p]; !ok || !legacy.ForceNew {
						t.Errorf("%s: native replacement without SDKv2 ForceNew", p)
					}
				}
			}
			sort.Strings(forceNew)
			if len(sdk)-1 != 691 || required != 96 || optional != 595 || computed != 26 || defaulted != 93 {
				t.Errorf("SDKv2 inventory paths=%d required=%d optional=%d computed=%d defaults=%d, want 691/96/595/26/93",
					len(sdk)-1, required, optional, computed, defaulted)
			}
			if strings.Join(forceNew, ",") != strings.Join(sdkTemplateForceNewPaths, ",") {
				t.Errorf("SDKv2 ForceNew paths = %v", forceNew)
			}
		})
	}
}

func checkListCardinality(ctx context.Context, t *testing.T, p string, info schemaPathInfo, legacy *legacyschema.Schema) {
	t.Helper()
	minimum := legacy.MinItems
	if info.block && legacy.Required && minimum == 0 {
		minimum = 1
	}
	elementType := types.StringType
	sizes := map[int]bool{minimum: false}
	if minimum > 0 {
		sizes[minimum-1] = true
	}
	if legacy.MaxItems > 0 {
		sizes[legacy.MaxItems] = false
		sizes[legacy.MaxItems+1] = true
	}
	for size, wantError := range sizes {
		elements := make([]attr.Value, size)
		for i := range elements {
			elements[i] = types.StringValue(fmt.Sprint(i))
		}
		var response validator.ListResponse
		for _, v := range info.listValidators {
			v.ValidateList(ctx, validator.ListRequest{Path: path.Root("spec"), ConfigValue: types.ListValueMust(elementType, elements)}, &response)
		}
		// Optional+computed object lists reject an explicit empty list.
		if _, objects := legacy.Elem.(*legacyschema.Resource); size == 0 && legacy.Computed && objects {
			wantError = true
		}
		if response.Diagnostics.HasError() != wantError {
			t.Errorf("%s: %d items error=%t, want %t", p, size, response.Diagnostics.HasError(), wantError)
		}
	}
}

func TestSpecBlockRestartPolicyValidatorMatchesSDK(t *testing.T) {
	ctx := context.Background()
	for name, options := range templateOwners {
		legacy := sdkTemplateSpec(t, name).Elem.(*legacyschema.Resource).Schema["restart_policy"]
		attribute := SpecBlock(options).NestedObject.Attributes["restart_policy"].(schema.StringAttribute)
		if !strings.HasPrefix(attribute.Description, legacy.Description) {
			t.Errorf("%s: restart_policy description %q, want %q", name, attribute.Description, legacy.Description)
		}
		for _, value := range []string{"Always", "OnFailure", "Never", "always", ""} {
			_, legacyErrors := legacy.ValidateFunc(value, "restart_policy")
			var response validator.StringResponse
			for _, v := range attribute.Validators {
				v.ValidateString(ctx, validator.StringRequest{Path: path.Root("restart_policy"), ConfigValue: types.StringValue(value)}, &response)
			}
			if response.Diagnostics.HasError() != (len(legacyErrors) > 0) {
				t.Errorf("%s: restart_policy %q error=%t, SDKv2 errors=%v", name, value, response.Diagnostics.HasError(), legacyErrors)
			}
		}
	}
}

func TestSpecBlockDescriptionsMatchSDK(t *testing.T) {
	ctx := context.Background()
	for name, options := range templateOwners {
		t.Run(name, func(t *testing.T) {
			sdk := map[string]*legacyschema.Schema{}
			collectSDKPaths("spec", sdkTemplateSpec(t, name), sdk)
			native := map[string]schemaPathInfo{}
			block := SpecBlock(options)
			collectNativeObject(ctx, "spec", block.NestedObject.Attributes, block.NestedObject.Blocks, native)
			for p, legacy := range sdk {
				if p == "spec" || legacy.Description == "" {
					continue
				}
				metadata, ok := podSpecDescriptionByPath[p]
				if p != "spec.restart_policy" || !options.RestartPolicyAlways {
					if !ok || metadata.description != legacy.Description {
						t.Errorf("%s: static description %q, want %q", p, metadata.description, legacy.Description)
					}
					if metadata.minItems != legacy.MinItems || metadata.maxItems != legacy.MaxItems {
						t.Errorf("%s: static cardinality (%d,%d), want (%d,%d)", p, metadata.minItems, metadata.maxItems, legacy.MinItems, legacy.MaxItems)
					}
				}
				info, ok := native[p]
				if !ok {
					continue
				}
				parent := p[:strings.LastIndex(p, ".")]
				if info.inline && !strings.Contains(native[parent].description, legacy.Description) {
					t.Errorf("%s: inline description missing from parent", p)
				} else if !info.inline && !strings.HasPrefix(info.description, legacy.Description) {
					t.Errorf("%s: description %q, want prefix %q", p, info.description, legacy.Description)
				}
			}
		})
	}
}

func collectSDKPaths(p string, s *legacyschema.Schema, out map[string]*legacyschema.Schema) {
	out[p] = s
	if r, ok := s.Elem.(*legacyschema.Resource); ok {
		for name, child := range r.Schema {
			collectSDKPaths(p+"."+name, child, out)
		}
	}
}

func sdkKind(s *legacyschema.Schema) string {
	return strings.TrimPrefix(s.Type.String(), "Type")
}

func sdkDefault(s *legacyschema.Schema) attr.Value {
	switch v := s.Default.(type) {
	case string:
		return types.StringValue(v)
	case bool:
		return types.BoolValue(v)
	case int:
		return types.Int64Value(int64(v))
	}
	panic(fmt.Sprintf("unsupported SDKv2 default %T", s.Default))
}

func zeroValue(v attr.Value) bool {
	switch value := v.(type) {
	case types.String:
		return value.ValueString() == ""
	case types.Bool:
		return !value.ValueBool()
	case types.Int64:
		return value.ValueInt64() == 0
	}
	return false
}

func collectNativeObject(ctx context.Context, p string, attributes map[string]schema.Attribute, blocks map[string]schema.Block, out map[string]schemaPathInfo) {
	for name, a := range attributes {
		collectNativeAttribute(ctx, p+"."+name, a, out)
	}
	for name, b := range blocks {
		nested := b.(schema.ListNestedBlock)
		key := p + "." + name
		out[key] = schemaPathInfo{kind: "List", block: true, listValidators: nested.Validators, description: nested.Description,
			forceNew: podModifiersRequireReplacement(nested.PlanModifiers)}
		collectNativeObject(ctx, key, nested.NestedObject.Attributes, nested.NestedObject.Blocks, out)
	}
}

func collectNativeAttribute(ctx context.Context, p string, a schema.Attribute, out map[string]schemaPathInfo) {
	info := schemaPathInfo{required: a.IsRequired(), optional: a.IsOptional(), computed: a.IsComputed(), description: a.GetDescription()}
	switch v := a.(type) {
	case schema.StringAttribute:
		info.kind, info.forceNew, info.validated = "String", podModifiersRequireReplacement(v.PlanModifiers), len(v.Validators) > 0
		info.defaultValue = nativeDefault(ctx, v.Default)
	case schema.BoolAttribute:
		info.kind, info.validated = "Bool", len(v.Validators) > 0
		info.defaultValue = nativeDefault(ctx, v.Default)
	case schema.Int64Attribute:
		info.kind, info.validated = "Int", len(v.Validators) > 0
		info.defaultValue = nativeDefault(ctx, v.Default)
	case schema.ListAttribute:
		info.kind, info.listValidators = "List", v.Validators
		info.defaultValue = nativeDefault(ctx, v.Default)
		if object, ok := v.ElementType.(types.ObjectType); ok {
			for name, childType := range object.AttrTypes {
				out[p+"."+name] = schemaPathInfo{kind: strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%T", childType), "basetypes."), "Type"), inline: true}
			}
		}
	case schema.SetAttribute:
		info.kind, info.forceNew = "Set", podModifiersRequireReplacement(v.PlanModifiers)
	case schema.MapAttribute:
		info.kind, info.forceNew = "Map", podModifiersRequireReplacement(v.PlanModifiers)
		info.defaultValue = nativeDefault(ctx, v.Default)
	case schema.ListNestedAttribute:
		info.kind, info.listValidators = "List", v.Validators
		for name, child := range v.NestedObject.Attributes {
			collectNativeAttribute(ctx, p+"."+name, child, out)
		}
	default:
		panic(fmt.Sprintf("%s: unexpected attribute %T", p, a))
	}
	out[p] = info
}

func nativeDefault(ctx context.Context, d any) attr.Value {
	switch v := d.(type) {
	case defaults.String:
		var response defaults.StringResponse
		v.DefaultString(ctx, defaults.StringRequest{}, &response)
		return response.PlanValue
	case defaults.Bool:
		var response defaults.BoolResponse
		v.DefaultBool(ctx, defaults.BoolRequest{}, &response)
		return response.PlanValue
	case defaults.Int64:
		var response defaults.Int64Response
		v.DefaultInt64(ctx, defaults.Int64Request{}, &response)
		return response.PlanValue
	case defaults.List:
		var response defaults.ListResponse
		v.DefaultList(ctx, defaults.ListRequest{}, &response)
		return response.PlanValue
	case defaults.Map:
		var response defaults.MapResponse
		v.DefaultMap(ctx, defaults.MapRequest{}, &response)
		return response.PlanValue
	}
	return nil
}

func protocol5Block(t *testing.T, block *tfprotov5.SchemaBlock, names ...string) *tfprotov5.SchemaNestedBlock {
	t.Helper()
	var found *tfprotov5.SchemaNestedBlock
	for _, name := range names {
		found = nil
		for _, nested := range block.BlockTypes {
			if nested.TypeName == name {
				found = nested
			}
		}
		if found == nil {
			t.Fatalf("SDKv2 protocol block %s not found", name)
			return nil
		}
		block = found.Block
	}
	return found
}

func protocol6Block(t *testing.T, block *tfprotov6.SchemaBlock, name string) *tfprotov6.SchemaNestedBlock {
	t.Helper()
	for _, nested := range block.BlockTypes {
		if nested.TypeName == name {
			return nested
		}
	}
	t.Fatalf("Framework protocol block %s not found", name)
	return nil
}

func protocol5BlockPaths(block *tfprotov5.SchemaBlock, p string, out map[string]bool) {
	for _, nested := range block.BlockTypes {
		out[p+"."+nested.TypeName] = true
		protocol5BlockPaths(nested.Block, p+"."+nested.TypeName, out)
	}
}

func protocol6BlockPaths(block *tfprotov6.SchemaBlock, p string, out map[string]bool) {
	for _, nested := range block.BlockTypes {
		out[p+"."+nested.TypeName] = true
		protocol6BlockPaths(nested.Block, p+"."+nested.TypeName, out)
	}
}

func frameworkProtocolSchema(t *testing.T, options Options) *tfprotov6.SchemaBlock {
	t.Helper()
	server, err := providerserver.NewProtocol6WithError(testProvider{options: options})()
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range response.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", d.Summary, d.Detail)
		}
	}
	return response.ResourceSchemas["podtemplate_test"].Block
}

type testProvider struct{ options Options }

func (testProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "podtemplate"
}
func (testProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {}
func (testProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (testProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }
func (p testProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return testResource(p) }}
}

type testResource struct{ options Options }

func (testResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "podtemplate_test"
}
func (r testResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"spec": SpecBlock(r.options)}}
}
func (testResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {}
func (testResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse)       {}
func (testResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {}
func (testResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {}
