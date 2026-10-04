// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/utils/ptr"
)

type ingressClassTestProvider struct {
	meta func() any
}

// Fixture JSON keeps the released SDK shape. This independent test adapter
// expresses the same configuration in the target schema without changing the
// source fixtures passed to UpgradeResourceState and MoveResourceState.
func networkingTestFixtureJSON(t *testing.T, raw []byte, target tftypes.Type) []byte {
	t.Helper()
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	var convert func(any, tftypes.Type) any
	convert = func(value any, typ tftypes.Type) any {
		if value == nil {
			return nil
		}
		switch typ := typ.(type) {
		case tftypes.Object:
			if list, ok := value.([]any); ok {
				if len(list) == 0 {
					return nil
				}
				if len(list) != 1 {
					t.Fatal("singleton fixture has more than one object")
				}
				value = list[0]
			}
			object := value.(map[string]any)
			for name, child := range typ.AttributeTypes {
				object[name] = convert(object[name], child)
			}
		case tftypes.List:
			for i, child := range value.([]any) {
				value.([]any)[i] = convert(child, typ.ElementType)
			}
		case tftypes.Set:
			for i, child := range value.([]any) {
				value.([]any)[i] = convert(child, typ.ElementType)
			}
		}
		return value
	}
	encoded, err := json.Marshal(convert(value, target))
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func networkingTestUpgradedState(t *testing.T, server tfprotov6.ProviderServer, resourceType, raw string, target tftypes.Type) tftypes.Value {
	t.Helper()
	response, err := server.UpgradeResourceState(context.Background(), &tfprotov6.UpgradeResourceStateRequest{
		TypeName: resourceType, Version: 0, RawState: &tfprotov6.RawState{JSON: []byte(raw)},
	})
	if err != nil {
		t.Fatal(err)
	}
	ingressClassCheckDiagnostics(t, response.Diagnostics)
	if response.UpgradedState == nil {
		t.Fatal("missing upgraded state")
	}
	value, err := response.UpgradedState.Unmarshal(target)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func networkingTestRetype(t *testing.T, value tftypes.Value, target tftypes.Type) tftypes.Value {
	t.Helper()
	if value.IsNull() {
		return tftypes.NewValue(target, nil)
	}
	if !value.IsKnown() {
		return tftypes.NewValue(target, tftypes.UnknownValue)
	}
	switch target := target.(type) {
	case tftypes.Object:
		if _, ok := value.Type().(tftypes.List); ok {
			var elements []tftypes.Value
			if err := value.As(&elements); err != nil {
				t.Fatal(err)
			}
			if len(elements) == 0 {
				return tftypes.NewValue(target, nil)
			}
			if len(elements) != 1 {
				t.Fatal("singleton fixture has more than one object")
			}
			value = elements[0]
		}
		var fields map[string]tftypes.Value
		if err := value.As(&fields); err != nil {
			t.Fatal(err)
		}
		fields = maps.Clone(fields)
		for name, typ := range target.AttributeTypes {
			fields[name] = networkingTestRetype(t, fields[name], typ)
		}
		return tftypes.NewValue(target, fields)
	case tftypes.List:
		var elements []tftypes.Value
		if err := value.As(&elements); err != nil {
			t.Fatal(err)
		}
		for i := range elements {
			elements[i] = networkingTestRetype(t, elements[i], target.ElementType)
		}
		return tftypes.NewValue(target, elements)
	}
	return value
}

// NetworkingTargetTestConfig renders legacy fixture blocks using the approved
// object syntax. Collections within objects remain lists of objects.
func NetworkingTargetTestConfig(legacy string) string {
	file, diagnostics := hclsyntax.ParseConfig([]byte(legacy), "fixture.tf", hcl.InitialPos)
	if diagnostics.HasErrors() {
		panic(diagnostics.Error())
	}
	singleton := map[string]bool{"default_backend": true, "backend": true, "service": true, "port": true, "resource": true, "http": true, "parameters": true, "pod_selector": true, "namespace_selector": true, "ip_block": true}
	var render func(*hclsyntax.Body, bool) string
	render = func(body *hclsyntax.Body, object bool) string {
		var result strings.Builder
		names := make([]string, 0, len(body.Attributes))
		for name := range body.Attributes {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(&result, "%s = %s\n", name, body.Attributes[name].Expr.Range().SliceBytes([]byte(legacy)))
		}
		seen := map[string]bool{}
		for _, block := range body.Blocks {
			// Top-level resource declarations are not backend resource objects.
			selected := singleton[block.Type] && len(block.Labels) == 0
			if object && !selected {
				if seen[block.Type] {
					continue
				}
				seen[block.Type] = true
				fmt.Fprintf(&result, "%s = [\n", block.Type)
				for _, sibling := range body.Blocks {
					if sibling.Type == block.Type {
						fmt.Fprintf(&result, "{\n%s},\n", render(sibling.Body, true))
					}
				}
				result.WriteString("]\n")
			} else if selected {
				fmt.Fprintf(&result, "%s = {\n%s}\n", block.Type, render(block.Body, true))
			} else {
				result.WriteString(block.Type)
				for _, label := range block.Labels {
					fmt.Fprintf(&result, " %q", label)
				}
				fmt.Fprintf(&result, " {\n%s}\n", render(block.Body, false))
			}
		}
		return result.String()
	}
	return render(file.Body.(*hclsyntax.Body), false)
}

func NetworkingTargetTestValues(legacy map[string]interface{}) map[string]interface{} {
	var clone func(any) any
	clone = func(value any) any {
		switch value := value.(type) {
		case map[string]any:
			result := make(map[string]any, len(value))
			for key, child := range value {
				result[key] = clone(child)
			}
			return result
		case []any:
			result := make([]any, len(value))
			for i, child := range value {
				result[i] = clone(child)
			}
			return result
		default:
			return value
		}
	}
	cloned := clone(legacy).(map[string]interface{})
	singleton := map[string]bool{"default_backend": true, "backend": true, "service": true, "port": true, "resource": true, "http": true, "parameters": true, "pod_selector": true, "namespace_selector": true, "ip_block": true}
	var convert func(any)
	convert = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			for key, child := range value {
				if singleton[key] {
					if list, ok := child.([]any); ok {
						if len(list) > 1 {
							panic("singleton state fixture has multiple values")
						}
						child = nil
						if len(list) == 1 {
							child = list[0]
						}
						value[key] = child
					}
				}
				convert(child)
			}
		case []any:
			for _, child := range value {
				convert(child)
			}
		}
	}
	convert(cloned)
	return cloned
}

func NetworkingOmittedTestExpressions(values map[string]interface{}) map[string]interface{} {
	result := NetworkingTargetTestValues(values)
	var visit func(interface{})
	visit = func(value interface{}) {
		switch value := value.(type) {
		case map[string]interface{}:
			for field, child := range value {
				if field == "pod_selector" || field == "namespace_selector" {
					if selector, ok := child.(map[string]interface{}); ok {
						if expressions, ok := selector["match_expressions"].([]interface{}); ok && len(expressions) == 0 {
							selector["match_expressions"] = nil
						}
					}
				}
				visit(child)
			}
		case []interface{}:
			for _, child := range value {
				visit(child)
			}
		}
	}
	visit(result)
	return result
}

func networkingTargetFlatAttributes(legacy map[string]string) map[string]string {
	singleton := map[string]bool{"pod_selector": true, "namespace_selector": true, "ip_block": true}
	converted := make(map[string]string, len(legacy))
	for key, value := range legacy {
		if strings.HasSuffix(key, ".match_expressions.#") && value == "0" {
			continue
		}
		parts := strings.Split(key, ".")
		var target []string
		skip := false
		for i := 0; i < len(parts); i++ {
			target = append(target, parts[i])
			if singleton[parts[i]] && i+1 < len(parts) {
				if parts[i+1] == "#" {
					skip = true
					break
				}
				if parts[i+1] == "0" {
					i++
				}
			}
		}
		if !skip {
			converted[strings.Join(target, ".")] = value
		}
	}
	return converted
}

func (ingressClassTestProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
}

func (ingressClassTestProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}

func (p ingressClassTestProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.meta
}

func (ingressClassTestProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewIngressClassV1}
}

func (ingressClassTestProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}

func ingressClassTestDynamic(t *testing.T, typ tftypes.Type, raw string) *tfprotov6.DynamicValue {
	t.Helper()
	value, err := (&tfprotov6.RawState{JSON: networkingTestFixtureJSON(t, []byte(raw), typ)}).Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := tfprotov6.NewDynamicValue(typ, value)
	if err != nil {
		t.Fatal(err)
	}
	return &dynamic
}

const ingressClassTestConfig = `{
	"id":null,
	"metadata":[{"name":"example","generate_name":null,"annotations":null,"labels":null,
	"generation":null,"resource_version":null,"uid":null}],
	"spec":[{"controller":null,"parameters":[{"name":"example","kind":"IngressParameters",
	"api_group":null,"namespace":null,"scope":null}]}]
}`

const ingressClassTestState = `{
	"id":"example",
	"metadata":[{"name":"example","generate_name":"","annotations":null,"labels":null,
	"generation":1,"resource_version":"10","uid":"uid-1"}],
	"spec":[{"controller":"","parameters":[{"name":"example","kind":"IngressParameters",
	"api_group":"","namespace":"","scope":"Cluster"}]}]
}`

func TestIngressClassProtocolDefaultsAndUpgrade(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(ingressClassTestProvider{})()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	ingressClassCheckDiagnostics(t, schemas.Diagnostics)
	typ := schemas.ResourceSchemas["kubernetes_ingress_class_v1"].ValueType()
	sdk, err := tf5to6server.UpgradeServer(ctx, kubernetes.Provider().GRPCProvider)
	if err != nil {
		t.Fatal(err)
	}
	sdkSchemas, err := sdk.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	ingressClassCheckDiagnostics(t, sdkSchemas.Diagnostics)
	if typ.Equal(sdkSchemas.ResourceSchemas["kubernetes_ingress_class"].ValueType()) {
		t.Fatal("singleton conversion must change the state type")
	}
	for _, tc := range []struct {
		name, prior, proposed string
	}{
		{"create defaults", "null", ingressClassTestConfig},
		{"unchanged SDK upgrade", ingressClassTestState, ingressClassTestState},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_ingress_class_v1", Config: ingressClassTestDynamic(t, typ, ingressClassTestConfig),
				PriorState: ingressClassTestDynamic(t, typ, tc.prior), ProposedNewState: ingressClassTestDynamic(t, typ, tc.proposed),
			})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, plan.Diagnostics)
			if len(plan.RequiresReplace) != 0 {
				t.Fatalf("unexpected replacement: %v", plan.RequiresReplace)
			}
			value, err := plan.PlannedState.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			var schemaResponse resource.SchemaResponse
			NewIngressClassV1().Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			var model IngressClassV1Model
			state := tfsdk.State{Schema: schemaResponse.Schema, Raw: value}
			if diags := state.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			for name, field := range map[string]types.String{
				"generate_name": model.Metadata[0].GenerateName, "controller": model.Spec[0].Controller,
				"api_group": model.Spec[0].Parameters.APIGroup, "namespace": model.Spec[0].Parameters.Namespace,
			} {
				if !field.Equal(types.StringValue("")) {
					t.Errorf("%s = %s, want SDKv2 empty default", name, field)
				}
			}
			if tc.prior != "null" {
				old, err := ingressClassTestDynamic(t, typ, tc.prior).Unmarshal(typ)
				if err != nil {
					t.Fatal(err)
				}
				if !value.Equal(old) {
					t.Fatalf("unchanged configuration changed state:\nold %s\nnew %s", old, value)
				}
			}
		})
	}
}

func ingressClassCheckDiagnostics(t *testing.T, diags []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diags {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}

func TestIngressClassProtocolValidation(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(ingressClassTestProvider{})()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.ResourceSchemas["kubernetes_ingress_class_v1"].ValueType()
	for _, tc := range []struct {
		name, config string
		wantError    bool
	}{
		{"minimal", ingressClassTestConfig, false},
		{"invalid scope", strings.Replace(ingressClassTestConfig, `"scope":null`, `"scope":"Namespaced"`, 1), true},
		{"missing parameter name", strings.Replace(ingressClassTestConfig, `"parameters":[{"name":"example"`, `"parameters":[{"name":null`, 1), true},
		{"invalid metadata name", strings.Replace(ingressClassTestConfig, `"name":"example"`, `"name":"INVALID_NAME"`, 1), true},
		{"conflicting names", strings.Replace(ingressClassTestConfig, `"generate_name":null`, `"generate_name":"generated-"`, 1), true},
		{"required blocks", `{"id":null,"metadata":[],"spec":[]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: "kubernetes_ingress_class_v1", Config: ingressClassTestDynamic(t, typ, tc.config),
			})
			if err != nil {
				t.Fatal(err)
			}
			hasError := false
			for _, diagnostic := range response.Diagnostics {
				hasError = hasError || diagnostic.Severity == tfprotov6.DiagnosticSeverityError
			}
			if hasError != tc.wantError {
				t.Fatalf("validation error = %v, want %v: %v", hasError, tc.wantError, response.Diagnostics)
			}
		})
	}
	var schemaResponse resource.SchemaResponse
	r := &IngressClassV1{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	value, err := ingressClassTestDynamic(t, typ, ingressClassTestConfig).Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	config := tfsdk.State{Schema: schemaResponse.Schema, Raw: value}
	if diags := config.SetAttribute(ctx, path.Root("spec"), types.ListUnknown(schemaResponse.Schema.Blocks["spec"].Type().(types.ListType).ElemType)); diags.HasError() {
		t.Fatal(diags)
	}
	dynamic, err := tfprotov6.NewDynamicValue(typ, config.Raw)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
		TypeName: "kubernetes_ingress_class_v1", Config: &dynamic,
	})
	if err != nil {
		t.Fatal(err)
	}
	ingressClassCheckDiagnostics(t, response.Diagnostics)
}

func TestIngressClassReplacementAndRemovalParity(t *testing.T) {
	ctx := context.Background()
	framework := providerserver.NewProtocol6(ingressClassTestProvider{})()
	sdk, err := tf5to6server.UpgradeServer(ctx, kubernetes.Provider().GRPCProvider)
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := framework.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.ResourceSchemas["kubernetes_ingress_class_v1"].ValueType()
	sdkSchemas, err := sdk.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	sdkType := sdkSchemas.ResourceSchemas["kubernetes_ingress_class"].ValueType()
	for _, tc := range []struct {
		name, config, prior, proposed string
		replace                       bool
	}{
		{"controller edit", strings.Replace(ingressClassTestConfig, `"controller":null`, `"controller":"changed"`, 1),
			ingressClassTestState, strings.Replace(ingressClassTestState, `"controller":""`, `"controller":"changed"`, 1), false},
		{"controller removal", ingressClassTestConfig, strings.Replace(ingressClassTestState, `"controller":""`, `"controller":"changed"`, 1),
			strings.Replace(ingressClassTestState, `"controller":""`, `"controller":null`, 1), false},
		{"parameter namespace removal", ingressClassTestConfig, strings.Replace(ingressClassTestState, `"namespace":""`, `"namespace":"changed"`, 1),
			strings.Replace(ingressClassTestState, `"namespace":""`, `"namespace":null`, 1), false},
		{"name edit", strings.Replace(ingressClassTestConfig, `"name":"example"`, `"name":"changed"`, 1),
			ingressClassTestState, strings.Replace(ingressClassTestState, `"name":"example"`, `"name":"changed"`, 1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var plans []tftypes.Value
			for _, implementation := range []struct {
				server tfprotov6.ProviderServer
				name   string
			}{{sdk, "kubernetes_ingress_class"}, {framework, "kubernetes_ingress_class_v1"}} {
				wireType := typ
				if implementation.name == "kubernetes_ingress_class" {
					wireType = sdkType
				}
				response, err := implementation.server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
					TypeName: implementation.name,
					Config:   ingressClassTestDynamic(t, wireType, tc.config), PriorState: ingressClassTestDynamic(t, wireType, tc.prior),
					ProposedNewState: ingressClassTestDynamic(t, wireType, tc.proposed),
				})
				if err != nil {
					t.Fatal(err)
				}
				ingressClassCheckDiagnostics(t, response.Diagnostics)
				if (len(response.RequiresReplace) > 0) != tc.replace {
					t.Fatalf("%s replacement paths: %v, want replacement %v", implementation.name, response.RequiresReplace, tc.replace)
				}
				value, err := response.PlannedState.Unmarshal(wireType)
				if err != nil {
					t.Fatal(err)
				}
				plans = append(plans, networkingTestRetype(t, value, typ))
			}
			var schemaResponse resource.SchemaResponse
			NewIngressClassV1().Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			var models [2]IngressClassV1Model
			for i, value := range plans {
				state := tfsdk.State{Schema: schemaResponse.Schema, Raw: value}
				if diags := state.Get(ctx, &models[i]); diags.HasError() {
					t.Fatal(diags)
				}
			}
			sdkSpec, frameworkSpec := models[0].Spec[0], models[1].Spec[0]
			if sdkSpec.Controller.ValueString() != frameworkSpec.Controller.ValueString() ||
				sdkSpec.Parameters.APIGroup.ValueString() != frameworkSpec.Parameters.APIGroup.ValueString() ||
				sdkSpec.Parameters.Namespace.ValueString() != frameworkSpec.Parameters.Namespace.ValueString() ||
				!sdkSpec.Parameters.Name.Equal(frameworkSpec.Parameters.Name) ||
				!sdkSpec.Parameters.Kind.Equal(frameworkSpec.Parameters.Kind) {
				t.Fatalf("SDKv2/Framework spec plans differ:\nSDK %#v\nFramework %#v", models[0].Spec, models[1].Spec)
			}
			if frameworkSpec.Controller.IsNull() || frameworkSpec.Controller.IsUnknown() ||
				frameworkSpec.Parameters.APIGroup.IsNull() || frameworkSpec.Parameters.APIGroup.IsUnknown() ||
				frameworkSpec.Parameters.Namespace.IsNull() || frameworkSpec.Parameters.Namespace.IsUnknown() {
				t.Fatal("Framework must resolve SDKv2 zero defaults before apply")
			}
			// Framework may defer computed scope during an update; the API lifecycle test
			// separately proves that it resolves without changing the configured payload.
			if !frameworkSpec.Parameters.Scope.IsUnknown() && !frameworkSpec.Parameters.Scope.Equal(sdkSpec.Parameters.Scope) {
				t.Fatalf("unexpected known scope: %s", frameworkSpec.Parameters.Scope)
			}
		})
	}
}

type ingressClassTestClientsets struct {
	kubernetes.KubeClientsets
	client *kubeclient.Clientset
}

func (m ingressClassTestClientsets) MainClientset() (*kubeclient.Clientset, error) {
	return m.client, nil
}

func (ingressClassTestClientsets) GetIgnoreAnnotations() []string {
	return []string{"ignored.example/.*"}
}

func (ingressClassTestClientsets) GetIgnoreLabels() []string { return []string{"ignored.example/.*"} }

func ingressClassTestResource(t *testing.T, handler http.HandlerFunc) *IngressClassV1 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := kubeclient.NewForConfig(&rest.Config{
		Host: server.URL,
		ContentConfig: rest.ContentConfig{
			ContentType: "application/json", AcceptContentTypes: "application/json",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &IngressClassV1{SDKv2Meta: func() any { return ingressClassTestClientsets{client: client} }}
}

func ingressClassTestStateValue(t *testing.T, r *IngressClassV1) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	raw := networkingTestUpgradedState(t, providerserver.NewProtocol6(ingressClassTestProvider{})(), "kubernetes_ingress_class_v1", ingressClassTestState, schemaResponse.Schema.Type().TerraformType(ctx))
	var err error
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: schemaResponse.Schema, Raw: raw}
}

func TestIngressClassRejectsMultipleLegacyParameters(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(ingressClassTestProvider{})()
	var legacy map[string]any
	if err := json.Unmarshal([]byte(ingressClassTestState), &legacy); err != nil {
		t.Fatal(err)
	}
	spec := legacy["spec"].([]any)[0].(map[string]any)
	parameters := spec["parameters"].([]any)
	spec["parameters"] = append(parameters, parameters[0])
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "kubernetes_ingress_class_v1", Version: 0, RawState: &tfprotov6.RawState{JSON: raw},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError && strings.Contains(diagnostic.Detail, "parameters") && strings.Contains(diagnostic.Detail, "previous provider") {
			return
		}
	}
	t.Fatalf("multiple legacy parameters were silently truncated: %#v", response)
}

func TestIngressClassLifecycle(t *testing.T) {
	ctx := context.Background()
	var object *networking.IngressClass
	var methods []string
	r := ingressClassTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		methods = append(methods, req.Method)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case http.MethodPost:
			if err := json.NewDecoder(req.Body).Decode(&object); err != nil {
				t.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			object.APIVersion, object.Kind = "networking.k8s.io/v1", "IngressClass"
			object.UID, object.ResourceVersion, object.Generation = "uid-1", "10", 1
			scope := "Cluster"
			object.Spec.Parameters.Scope = &scope
			object.Labels = map[string]string{"external": "retained", "ignored.example/label": "hidden"}
		case http.MethodGet:
			if object == nil {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","reason":"NotFound","code":404}`)
				return
			}
		case http.MethodPatch:
			var operations json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&operations); err != nil {
				t.Error(err)
				return
			}
			patch, err := jsonpatch.DecodePatch(operations)
			if err != nil {
				t.Error(err)
				return
			}
			before, err := json.Marshal(object)
			if err != nil {
				t.Error(err)
				return
			}
			after, err := patch.Apply(before)
			if err != nil {
				t.Error(err)
				return
			}
			if err := json.Unmarshal(after, object); err != nil {
				t.Error(err)
				return
			}
			object.ResourceVersion = "11"
		case http.MethodDelete:
			if object == nil {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","reason":"NotFound","code":404}`)
				return
			}
			object = nil
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Success"}`)
			return
		default:
			t.Errorf("unexpected method %s", req.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := json.NewEncoder(w).Encode(object); err != nil {
			t.Error(err)
		}
	})
	state := ingressClassTestStateValue(t, r)
	var model IngressClassV1Model
	if diags := state.Get(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	model.Spec[0].Controller = types.StringValue("example.com/controller")
	model.Spec[0].Parameters.Scope = types.StringUnknown()
	model.ID, model.Metadata[0].UID, model.Metadata[0].ResourceVersion = types.StringUnknown(), types.StringUnknown(), types.StringUnknown()
	model.Metadata[0].Generation = types.Int64Unknown()
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	identity := &tfsdk.ResourceIdentity{Schema: common.IdentitySchema()}
	created := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}, Identity: identity}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(state)}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	if diags := created.State.Get(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	if model.ID.ValueString() != "example" || model.Metadata[0].UID.ValueString() != "uid-1" ||
		model.Spec[0].Parameters.Scope.ValueString() != "Cluster" || !model.Metadata[0].Labels.IsNull() {
		t.Fatalf("create did not preserve plan or resolve computed values: %#v", model)
	}
	var gotIdentity common.ResourceIdentity
	if diags := identity.Get(ctx, &gotIdentity); diags.HasError() {
		t.Fatal(diags)
	}
	if !reflect.DeepEqual(gotIdentity, ingressClassIdentity("example")) {
		t.Fatalf("wrong identity: %#v", gotIdentity)
	}

	model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"managed": types.StringValue("new")})
	model.Metadata[0].Generation, model.Metadata[0].ResourceVersion = types.Int64Unknown(), types.StringUnknown()
	model.Spec[0].Parameters.Name = types.StringValue("updated")
	planned := tfsdk.State{Schema: state.Schema}
	if diags := planned.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	updated := resource.UpdateResponse{State: created.State, Identity: identity}
	r.Update(ctx, resource.UpdateRequest{State: created.State, Plan: tfsdk.Plan{Schema: state.Schema, Raw: planned.Raw}}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}
	if object.UID != "uid-1" || object.Labels["external"] != "retained" ||
		object.Labels["ignored.example/label"] != "hidden" || object.Labels["managed"] != "new" ||
		object.Spec.Parameters.Name != "updated" {
		t.Fatalf("update lost identity, unmanaged metadata, or spec: %#v", object)
	}

	refreshed := resource.ReadResponse{State: updated.State, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: updated.State}, &refreshed)
	if refreshed.Diagnostics.HasError() {
		t.Fatal(refreshed.Diagnostics)
	}
	if diags := refreshed.State.Get(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	if _, exists := model.Metadata[0].Labels.Elements()["ignored.example/label"]; exists {
		t.Fatal("Read retained an ignored unowned label")
	}
	deleted := resource.DeleteResponse{State: refreshed.State}
	r.Delete(ctx, resource.DeleteRequest{State: refreshed.State}, &deleted)
	if deleted.Diagnostics.HasError() || object != nil {
		t.Fatalf("delete failed: %v", deleted.Diagnostics)
	}
	missing := resource.ReadResponse{State: refreshed.State}
	r.Read(ctx, resource.ReadRequest{State: refreshed.State}, &missing)
	if missing.Diagnostics.HasError() || !missing.State.Raw.IsNull() {
		t.Fatalf("not found did not remove state: %v", missing.Diagnostics)
	}
	r.Delete(ctx, resource.DeleteRequest{State: refreshed.State}, &deleted)
	if deleted.Diagnostics.HasError() {
		t.Fatal(deleted.Diagnostics)
	}
	if !reflect.DeepEqual(methods, []string{"POST", "GET", "PATCH", "GET", "DELETE", "GET", "GET", "DELETE"}) {
		t.Fatalf("unexpected lifecycle API actions: %v", methods)
	}
}

func TestIngressClassReadErrorPreservesState(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			r := ingressClassTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, "request failed")
			})
			state := ingressClassTestStateValue(t, r)
			response := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &response)
			if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
				t.Fatalf("failed Read must return an error and preserve state: %v", response.Diagnostics)
			}
		})
	}
}

func TestIngressClassMoveState(t *testing.T) {
	ctx := context.Background()
	r := &IngressClassV1{}
	state := ingressClassTestStateValue(t, r)
	for _, tc := range []struct {
		name, provider, source, raw string
		version                     int64
		wantState, wantError        bool
	}{
		{"alias", "registry.terraform.io/hashicorp/kubernetes", "kubernetes_ingress_class", ingressClassTestState, 0, true, false},
		{"mirror", "example.com/hashicorp/kubernetes", "kubernetes_ingress_class", ingressClassTestState, 0, true, false},
		{"different provider", "registry.terraform.io/nothashicorp/kubernetes", "kubernetes_ingress_class", ingressClassTestState, 0, false, false},
		{"different source", "registry.terraform.io/hashicorp/kubernetes", "kubernetes_ingress", ingressClassTestState, 0, false, false},
		{"future schema", "registry.terraform.io/hashicorp/kubernetes", "kubernetes_ingress_class", ingressClassTestState, 1, false, false},
		{"malformed", "registry.terraform.io/hashicorp/kubernetes", "kubernetes_ingress_class", "{", 0, false, true},
		{"empty", "registry.terraform.io/hashicorp/kubernetes", "kubernetes_ingress_class", "{}", 0, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			null := tftypes.NewValue(state.Raw.Type(), nil)
			response := resource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: state.Schema, Raw: null},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: common.IdentitySchema()},
			}
			r.MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
				SourceTypeName: tc.source, SourceProviderAddress: tc.provider, SourceSchemaVersion: tc.version,
				SourceRawState: &tfprotov6.RawState{JSON: []byte(tc.raw)},
			}, &response)
			if response.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("unexpected diagnostics: %v", response.Diagnostics)
			}
			if tc.wantState && !response.TargetState.Raw.Equal(state.Raw) {
				t.Fatalf("alias move changed state:\nold %s\nnew %s", state.Raw, response.TargetState.Raw)
			}
			if !tc.wantState && !response.TargetState.Raw.IsNull() {
				t.Fatal("unsupported source produced state")
			}
		})
	}
}

func TestIngressClassConfigure(t *testing.T) {
	for _, value := range []any{nil, 42, func() any { return nil }} {
		r := &IngressClassV1{}
		response := resource.ConfigureResponse{}
		r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: value}, &response)
		if response.Diagnostics.HasError() != (reflect.TypeOf(value) == reflect.TypeOf(42)) {
			t.Fatalf("unexpected Configure diagnostics for %T: %v", value, response.Diagnostics)
		}
		_, _, diags := networkingClient(r.SDKv2Meta)
		if !diags.HasError() {
			t.Fatalf("unusable metadata %T did not produce diagnostics", value)
		}
	}
}

func TestNetworkingMetadataPatchOwnership(t *testing.T) {
	ctx := context.Background()
	stringMap := func(values map[string]string) types.Map {
		result, diags := types.MapValueFrom(ctx, types.StringType, values)
		if diags.HasError() {
			t.Fatal(diags)
		}
		return result
	}
	for _, tc := range []struct {
		name              string
		live, prior, plan map[string]string
		want              map[string]string
	}{
		{"first owned key", map[string]string{"external": "kept"}, nil, map[string]string{"managed": "new"},
			map[string]string{"external": "kept", "managed": "new"}},
		{"remove owned key", map[string]string{"external": "kept", "managed": "old"}, map[string]string{"managed": "old"}, nil,
			map[string]string{"external": "kept"}},
		{"externally removed key", map[string]string{"external": "kept"}, map[string]string{"managed": "old"}, nil,
			map[string]string{"external": "kept"}},
		{"escaped key", map[string]string{"example.com/~key": "old"}, map[string]string{"example.com/~key": "old"}, map[string]string{"example.com/~key": "new"},
			map[string]string{"example.com/~key": "new"}},
		{"empty maps", nil, nil, map[string]string{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			live := metav1.ObjectMeta{Labels: tc.live, Annotations: tc.live}
			prior := common.MetadataBase{Labels: stringMap(tc.prior), Annotations: stringMap(tc.prior)}
			plan := common.MetadataBase{Labels: stringMap(tc.plan), Annotations: stringMap(tc.plan)}
			ops, diags := networkingMetadataPatch(ctx, live, prior, plan)
			if diags.HasError() {
				t.Fatal(diags)
			}
			encoded, err := json.Marshal(ops)
			if err != nil {
				t.Fatal(err)
			}
			patch, err := jsonpatch.DecodePatch(encoded)
			if err != nil {
				t.Fatal(err)
			}
			before, err := json.Marshal(networking.IngressClass{ObjectMeta: live})
			if err != nil {
				t.Fatal(err)
			}
			after, err := patch.Apply(before)
			if err != nil {
				t.Fatal(err)
			}
			var object networking.IngressClass
			if err := json.Unmarshal(after, &object); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(object.Labels, tc.want) || !reflect.DeepEqual(object.Annotations, tc.want) {
				t.Fatalf("patch did not preserve metadata ownership: %s", after)
			}
			if !reflect.DeepEqual(live.Labels, tc.live) {
				t.Fatal("metadata patch mutated its input")
			}
		})
	}
}

func TestIngressClassImport(t *testing.T) {
	ctx := context.Background()
	for _, useIdentity := range []bool{false, true} {
		t.Run(fmt.Sprintf("identity=%t", useIdentity), func(t *testing.T) {
			r := ingressClassTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/apis/networking.k8s.io/v1/ingressclasses/example" {
					t.Errorf("unexpected import request: %s %s", req.Method, req.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"apiVersion":"networking.k8s.io/v1","kind":"IngressClass",
					"metadata":{"name":"example","uid":"imported-uid","resourceVersion":"10","generation":1},
					"spec":{"controller":"example.com/controller"}}`)
			})
			state := ingressClassTestStateValue(t, r)
			state.Raw = tftypes.NewValue(state.Raw.Type(), nil)
			request := resource.ImportStateRequest{ID: "example"}
			response := resource.ImportStateResponse{State: state, Identity: &tfsdk.ResourceIdentity{Schema: common.IdentitySchema()}}
			if useIdentity {
				request.ID = ""
				request.Identity = &tfsdk.ResourceIdentity{Schema: common.IdentitySchema()}
				if diags := request.Identity.Set(ctx, ingressClassIdentity("example")); diags.HasError() {
					t.Fatal(diags)
				}
			}
			r.ImportState(ctx, request, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			read := resource.ReadResponse{State: response.State, Identity: response.Identity}
			r.Read(ctx, resource.ReadRequest{State: response.State}, &read)
			if read.Diagnostics.HasError() {
				t.Fatal(read.Diagnostics)
			}
			var model IngressClassV1Model
			if diags := read.State.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			if model.ID.ValueString() != "example" || model.Metadata[0].UID.ValueString() != "imported-uid" ||
				!model.Metadata[0].GenerateName.Equal(types.StringValue("")) ||
				model.Spec[0].Controller.ValueString() != "example.com/controller" || model.Spec[0].Parameters != nil {
				t.Fatalf("incomplete imported state: %#v", model)
			}
		})
	}
}

// Only the loopback fake API is reachable by either provider. This runs real Terraform
// and a released provider without TF_ACC or a Kubernetes cluster.
func TestIngressClassReleasedUpgradeOffline(t *testing.T) {
	if os.Getenv("KUBE_NETWORKING_CORE_TEST") != "1" {
		t.Skip("set KUBE_NETWORKING_CORE_TEST=1 for local fake-API Terraform CLI tests")
	}
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("Terraform CLI is not installed")
	}
	private := t.TempDir()
	rc := filepath.Join(private, "terraform.tfrc")
	if err := os.WriteFile(rc, []byte("provider_installation { direct {} }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TF_CLI_CONFIG_FILE", rc)
	t.Setenv("TF_ACC_TERRAFORM_PATH", terraformPath)
	t.Setenv("TF_ACC_TEMP_DIR", private)
	for _, key := range []string{
		"KUBECONFIG", "KUBE_CONFIG_PATH", "KUBE_CONFIG_PATHS", "KUBE_CTX", "KUBE_CTX_AUTH_INFO", "KUBE_CTX_CLUSTER",
		"KUBE_USER", "KUBE_PASSWORD", "KUBE_TOKEN", "KUBE_CLIENT_CERT_DATA", "KUBE_CLIENT_KEY_DATA",
		"KUBE_CLUSTER_CA_CERT_DATA", "KUBE_PROXY_URL", "KUBE_TLS_SERVER_NAME", "TF_REATTACH_PROVIDERS",
	} {
		t.Setenv(key, "")
	}
	for _, tc := range []struct {
		maps      string
		refreshed bool
	}{
		{"omitted", false}, {"empty", false}, {"omitted", true}, {"empty", true},
	} {
		name := tc.maps
		if tc.refreshed {
			name += "_refreshed"
		}
		t.Run(name, func(t *testing.T) {
			var mu sync.Mutex
			var object *networking.IngressClass
			creates, writes := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if !strings.HasPrefix(req.URL.Path, "/apis/networking.k8s.io/v1/ingressclasses") {
					t.Errorf("unexpected fake API request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				switch req.Method {
				case http.MethodPost:
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					decoded, _, err := kubescheme.Codecs.UniversalDeserializer().Decode(body, nil, nil)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					var ok bool
					object, ok = decoded.(*networking.IngressClass)
					if !ok {
						t.Errorf("unexpected object type %T", decoded)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					creates++
					object.APIVersion, object.Kind = "networking.k8s.io/v1", "IngressClass"
					object.UID, object.ResourceVersion, object.Generation = "offline-upgrade-uid", "1", 1
					if object.Spec.Parameters != nil && object.Spec.Parameters.Scope == nil {
						scope := "Cluster"
						object.Spec.Parameters.Scope = &scope
					}
				case http.MethodGet:
					if object == nil {
						w.WriteHeader(http.StatusNotFound)
						fmt.Fprint(w, `{"apiVersion":"v1","kind":"Status","reason":"NotFound","code":404}`)
						return
					}
				case http.MethodDelete:
					object = nil
					fmt.Fprint(w, `{"apiVersion":"v1","kind":"Status","status":"Success"}`)
					return
				default:
					writes++
					t.Errorf("unchanged upgrade unexpectedly wrote the object: %s", req.Method)
					w.WriteHeader(http.StatusConflict)
					return
				}
				if err := json.NewEncoder(w).Encode(object); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			t.Setenv("KUBE_HOST", server.URL)
			t.Setenv("KUBE_INSECURE", "false")
			client, err := kubeclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			meta := func() any { return ingressClassTestClientsets{client: client} }
			metadata := ""
			if tc.maps == "empty" {
				metadata = "labels = {}\nannotations = {}"
			}
			config := fmt.Sprintf(`
				resource "kubernetes_ingress_class_v1" "test" {
				  metadata {
				    name = "offline-upgrade"
				    %s
				  }
				  spec {
				    controller = "example.com/controller"
				    parameters {
				      name = "parameters"
				      kind = "IngressParameters"
				    }
				  }
				}
				`, metadata)
			targetConfig := NetworkingTargetTestConfig(config)
			var before *networking.IngressClass
			checkPreserved := func(*terraform.State) error {
				mu.Lock()
				defer mu.Unlock()
				if !reflect.DeepEqual(object, before) || creates != 1 || writes != 0 {
					return fmt.Errorf("unchanged upgrade altered the object, UID, or payload")
				}
				return nil
			}
			upgradeChecks := []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}
			if tc.maps == "empty" && !tc.refreshed {
				upgradeChecks = []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("kubernetes_ingress_class_v1.test", plancheck.ResourceActionUpdate),
					plancheck.ExpectKnownValue("kubernetes_ingress_class_v1.test",
						tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("labels"), knownvalue.MapExact(map[string]knownvalue.Check{})),
					plancheck.ExpectKnownValue("kubernetes_ingress_class_v1.test",
						tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("annotations"), knownvalue.MapExact(map[string]knownvalue.Check{})),
				}
			}
			testCase := tfresource.TestCase{
				IsUnitTest: true,
				CheckDestroy: func(*terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if object != nil {
						return fmt.Errorf("fake IngressClass still exists")
					}
					return nil
				},
				Steps: []tfresource.TestStep{
					{
						ExternalProviders: map[string]tfresource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "3.2.1"},
						},
						Config: config,
						Check: func(*terraform.State) error {
							mu.Lock()
							defer mu.Unlock()
							if object == nil || creates != 1 {
								return fmt.Errorf("released provider did not create exactly one object")
							}
							before = object.DeepCopy()
							return nil
						},
					},
					{
						ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
							"kubernetes": providerserver.NewProtocol6WithError(ingressClassTestProvider{meta: meta}),
						},
						Config:           targetConfig,
						ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: upgradeChecks},
						Check:            checkPreserved,
					},
					{
						ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
							"kubernetes": providerserver.NewProtocol6WithError(ingressClassTestProvider{meta: meta}),
						},
						Config: targetConfig,
						ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						}},
						Check: checkPreserved,
					},
				},
			}
			if tc.refreshed {
				refreshed := testCase.Steps[0]
				refreshed.ConfigPlanChecks = tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}
				testCase.Steps = append([]tfresource.TestStep{testCase.Steps[0], refreshed}, testCase.Steps[1:]...)
			}
			tfresource.Test(t, testCase)
		})
	}
}

type ingressInternalProvider struct {
	meta func() any
}

func (ingressInternalProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
}

func (ingressInternalProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}

func (p ingressInternalProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.meta
}

func (ingressInternalProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewIngressV1}
}

func (ingressInternalProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}

const ingressInternalConfigJSON = `{
	"id":null,"wait_for_load_balancer":null,"status":null,"timeouts":null,
	"metadata":[{"name":"test","namespace":null,"generate_name":null,"annotations":null,"labels":null,"generation":null,"resource_version":null,"uid":null}],
	"spec":[{
		"ingress_class_name":null,
		"default_backend":[],
		"rule":[{
			"host":null,
			"http":[{
				"path":[{
					"path":null,
					"path_type":null,
					"backend":[{
						"resource":[],
						"service":[{"name":"app","port":[{"name":null,"number":80}]}]
					}]
				}]
			}]
		}],
		"tls":[{"hosts":null,"secret_name":null}]
	}]
}`

func ingressInternalSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	NewIngressV1().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if diags := response.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatal(diags)
	}
	return response.Schema
}

func ingressInternalState(t *testing.T, rawJSON string) tfsdk.State {
	t.Helper()
	s := ingressInternalSchema(t)
	raw := tfprotov6.RawState{JSON: networkingTestFixtureJSON(t, []byte(rawJSON), s.Type().TerraformType(context.Background()))}
	value, err := raw.Unmarshal(s.Type().TerraformType(context.Background()))
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: s, Raw: value}
}

func ingressInternalDynamic(t *testing.T, value tftypes.Value) *tfprotov6.DynamicValue {
	t.Helper()
	result, err := tfprotov6.NewDynamicValue(value.Type(), value)
	if err != nil {
		t.Fatal(err)
	}
	return &result
}

func ingressInternalPlan(t *testing.T, change func(*IngressV1Model)) (tfsdk.Plan, IngressV1Model) {
	t.Helper()
	ctx := context.Background()
	config := ingressInternalState(t, ingressInternalConfigJSON)
	var model IngressV1Model
	if d := config.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	if change != nil {
		change(&model)
	}
	if d := config.Set(ctx, model); d.HasError() {
		t.Fatal(d)
	}
	server := providerserver.NewProtocol6(ingressInternalProvider{})()
	response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "kubernetes_ingress_v1",
		Config:           ingressInternalDynamic(t, config.Raw),
		PriorState:       ingressInternalDynamic(t, tftypes.NewValue(config.Raw.Type(), nil)),
		ProposedNewState: ingressInternalDynamic(t, config.Raw),
	})
	if err != nil {
		t.Fatal(err)
	}
	ingressInternalProtocolDiagnostics(t, response.Diagnostics)
	if len(response.RequiresReplace) != 0 {
		t.Fatalf("unexpected replacement: %v", response.RequiresReplace)
	}
	value, err := response.PlannedState.Unmarshal(config.Raw.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := tfsdk.Plan{Schema: config.Schema, Raw: value}
	if d := plan.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	return plan, model
}

func ingressInternalProtocolDiagnostics(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}

func TestIngressV1InternalNativeSchemaAndDefaults(t *testing.T) {
	ctx := context.Background()
	s := ingressInternalSchema(t)
	if s.Version != 1 {
		t.Fatalf("schema version = %d", s.Version)
	}
	if _, ok := NewIngressV1().(resource.ResourceWithMoveState); ok {
		t.Fatal("v1beta1 ingress alias must not have a v1 state mover")
	}
	response, err := providerserver.NewProtocol6(ingressInternalProvider{})().GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	ingressInternalProtocolDiagnostics(t, response.Diagnostics)
	wire := response.ResourceSchemas["kubernetes_ingress_v1"]
	for _, block := range wire.Block.BlockTypes {
		want := tfprotov6.SchemaNestedBlockNestingModeList
		if block.TypeName == "timeouts" {
			want = tfprotov6.SchemaNestedBlockNestingModeSingle
		}
		if block.Nesting != want {
			t.Fatalf("%s nesting = %v, want %v", block.TypeName, block.Nesting, want)
		}
	}
	timeoutType := wire.ValueType().(tftypes.Object).AttributeTypes["timeouts"].(tftypes.Object)
	if len(timeoutType.AttributeTypes) != 2 || !timeoutType.AttributeTypes["create"].Equal(tftypes.String) || !timeoutType.AttributeTypes["delete"].Equal(tftypes.String) {
		t.Fatalf("unexpected timeout type: %v", timeoutType)
	}
	sdk := &sdkschema.Resource{Schema: map[string]*sdkschema.Schema{
		"metadata": {Type: sdkschema.TypeList, Required: true, Elem: &sdkschema.Resource{
			Schema: map[string]*sdkschema.Schema{"name": {Type: sdkschema.TypeString, Required: true}},
		}},
	}, Timeouts: &sdkschema.ResourceTimeout{
		Create: sdkschema.DefaultTimeout(20 * time.Minute), Delete: sdkschema.DefaultTimeout(20 * time.Minute),
	}}
	if sdkTimeout := sdk.CoreConfigSchema().ImpliedType().AttributeType("timeouts"); !sdkTimeout.IsObjectType() || len(sdkTimeout.AttributeTypes()) != 2 {
		t.Fatalf("SDK timeout wire contract changed: %s", sdkTimeout.FriendlyName())
	}
	_, model := ingressInternalPlan(t, nil)
	if model.Metadata[0].GenerateName.ValueString() != "" || model.Metadata[0].GenerateName.IsNull() || model.Metadata[0].Namespace.ValueString() != "default" {
		t.Fatalf("metadata defaults: %#v", model.Metadata[0])
	}
	p := model.Spec[0].Rule[0].HTTP.Path[0]
	if p.Path.IsNull() || p.Path.ValueString() != "" || p.PathType.ValueString() != "ImplementationSpecific" {
		t.Fatalf("path defaults: %#v", p)
	}
	if p.Backend.Service.Port.Name.IsNull() || model.Spec[0].Rule[0].Host.IsNull() || model.Spec[0].TLS[0].SecretName.IsNull() {
		t.Fatal("implicit SDK string zeros became null")
	}
	if !model.WaitForLoadBalancer.IsNull() || !model.Spec[0].IngressClassName.IsUnknown() || !model.Status.IsUnknown() {
		t.Fatalf("computed/default fields: %#v", model)
	}
	if !model.Spec[0].TLS[0].Hosts.IsNull() {
		t.Fatalf("omitted TLS hosts on create must remain null: %s", model.Spec[0].TLS[0].Hosts)
	}
	_, unknown := ingressInternalPlan(t, func(m *IngressV1Model) {
		m.Spec[0].Rule[0].Host = types.StringUnknown()
		m.Spec[0].Rule[0].HTTP.Path[0].Backend.Service.Port.Number = types.Int64Unknown()
		m.Spec[0].TLS[0].Hosts = types.ListUnknown(types.StringType)
	})
	if !unknown.Spec[0].Rule[0].Host.IsUnknown() || !unknown.Spec[0].Rule[0].HTTP.Path[0].Backend.Service.Port.Number.IsUnknown() || !unknown.Spec[0].TLS[0].Hosts.IsUnknown() {
		t.Fatal("unknown configured values were coerced")
	}
	_, namedPort := ingressInternalPlan(t, func(m *IngressV1Model) {
		port := m.Spec[0].Rule[0].HTTP.Path[0].Backend.Service.Port
		port.Name, port.Number = types.StringValue("http"), types.Int64Null()
	})
	port := namedPort.Spec[0].Rule[0].HTTP.Path[0].Backend.Service.Port
	if port.Number.IsNull() || port.Number.IsUnknown() || port.Number.ValueInt64() != 0 || port.Name.ValueString() != "http" {
		t.Fatalf("named backend port lost SDK numerical zero: %#v", port)
	}
}

func TestIngressV1InternalConversion(t *testing.T) {
	ctx := context.Background()
	_, model := ingressInternalPlan(t, nil)
	model.Spec[0].TLS = append(model.Spec[0].TLS, IngressV1TLSModel{
		Hosts: types.ListValueMust(types.StringType, []attr.Value{}), SecretName: types.StringValue(""),
	})
	model.Spec[0].DefaultBackend = &IngressV1BackendModel{
		Resource: &IngressV1ResourceBackendModel{
			APIGroup: types.StringValue("example.com"), Kind: types.StringValue("StorageBucket"), Name: types.StringValue("bucket"),
		},
		Service: nil,
	}
	expanded, diags := ingressExpandSpec(ctx, model.Spec)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if expanded.IngressClassName != nil || expanded.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number != 80 || *expanded.DefaultBackend.Resource.APIGroup != "example.com" {
		t.Fatalf("unexpected expanded spec: %#v", expanded)
	}
	flat, diags := ingressFlattenSpec(ctx, expanded, model.Spec)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if flat[0].IngressClassName.IsNull() || flat[0].IngressClassName.ValueString() != "" || !flat[0].TLS[0].Hosts.IsNull() || flat[0].TLS[1].Hosts.IsNull() {
		t.Fatalf("zero/null/empty regression: %#v", flat[0])
	}
	expandedAgain, diags := ingressExpandSpec(ctx, flat)
	if diags.HasError() || !reflect.DeepEqual(expanded, expandedAgain) {
		t.Fatalf("roundtrip changed payload: %v\n%#v\n%#v", diags, expanded, expandedAgain)
	}
	model.Spec[0].Rule[0].Host = types.StringUnknown()
	if _, d := ingressExpandSpec(ctx, model.Spec); !d.HasError() {
		t.Fatal("apply silently converted unknown host into empty string")
	}
	flat, diags = ingressFlattenSpec(ctx, networking.IngressSpec{Rules: []networking.IngressRule{{Host: "host-only"}}}, nil)
	if diags.HasError() || flat[0].Rule[0].HTTP != nil {
		t.Fatalf("host-only rule conversion failed: %#v %v", flat, diags)
	}
}

type ingressInternalClientsets struct {
	kubernetes.KubeClientsets
	client *kubeclient.Clientset
}

func (m ingressInternalClientsets) MainClientset() (*kubeclient.Clientset, error) {
	return m.client, nil
}

func (ingressInternalClientsets) GetIgnoreAnnotations() []string { return []string{"^ignored/"} }

func (ingressInternalClientsets) GetIgnoreLabels() []string { return []string{"^ignored/"} }

func ingressInternalResource(t *testing.T, handler http.HandlerFunc) *IngressV1 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := kubeclient.NewForConfig(&rest.Config{
		Host:          server.URL,
		ContentConfig: rest.ContentConfig{ContentType: "application/json", AcceptContentTypes: "application/json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &IngressV1{SDKv2Meta: func() any { return ingressInternalClientsets{client: client} }}
}

func TestIngressV1InternalCreateRetainsGeneratedIdentity(t *testing.T) {
	for _, failWait := range []bool{false, true} {
		t.Run(fmt.Sprintf("waitError=%t", failWait), func(t *testing.T) {
			ctx := context.Background()
			plan, _ := ingressInternalPlan(t, func(m *IngressV1Model) {
				m.Metadata[0].Name = types.StringNull()
				m.Metadata[0].GenerateName = types.StringValue("generated-")
				m.WaitForLoadBalancer = types.BoolValue(true)
			})
			var created networking.Ingress
			gets := 0
			r := ingressInternalResource(t, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch req.Method {
				case http.MethodPost:
					if err := json.NewDecoder(req.Body).Decode(&created); err != nil {
						t.Error(err)
					}
					if created.Name != "" || created.GenerateName != "generated-" {
						t.Errorf("unexpected generated-name request: %#v", created.ObjectMeta)
					}
					created.Name, created.Namespace, created.UID = "generated-123", "default", "uid-123"
					created.ResourceVersion, created.Generation = "1", 1
					created.TypeMeta = metav1.TypeMeta{APIVersion: ingressAPIVersion, Kind: ingressKind}
				case http.MethodGet:
					gets++
					if req.URL.Path != "/apis/networking.k8s.io/v1/namespaces/default/ingresses/generated-123" {
						t.Errorf("waiter used non-server-assigned name: %s", req.URL.Path)
					}
					if failWait {
						http.Error(w, "forbidden", http.StatusForbidden)
						return
					}
					created.Status.LoadBalancer.Ingress = []networking.IngressLoadBalancerIngress{{Hostname: "lb.example.com"}}
				default:
					t.Errorf("unexpected method: %s", req.Method)
				}
				if err := json.NewEncoder(w).Encode(created); err != nil {
					t.Error(err)
				}
			})
			resp := resource.CreateResponse{
				State:    tfsdk.State{Schema: plan.Schema},
				Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
			}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
			if resp.Diagnostics.HasError() != failWait {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			var model IngressV1Model
			if d := resp.State.Get(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			if model.ID.ValueString() != "default/generated-123" || model.Metadata[0].Name.ValueString() != "generated-123" || model.Status.IsUnknown() || gets == 0 {
				t.Fatalf("created state not preserved: %#v", model)
			}
			var identity common.NamespacedResourceIdentity
			if d := resp.Identity.Get(ctx, &identity); d.HasError() || identity.Name.ValueString() != "generated-123" || identity.Namespace.ValueString() != "default" {
				t.Fatalf("identity not preserved: %#v %v", identity, d)
			}
		})
	}
}

func ingressInternalExisting(t *testing.T) (tfsdk.State, IngressV1Model) {
	t.Helper()
	plan, model := ingressInternalPlan(t, nil)
	ing := &networking.Ingress{ObjectMeta: metav1.ObjectMeta{
		Name: "test", Namespace: "default", UID: "uid-1", ResourceVersion: "1", Generation: 1,
	}}
	if d := ingressApplyResult(context.Background(), &model, ing); d.HasError() {
		t.Fatal(d)
	}
	state := tfsdk.State{Schema: plan.Schema}
	if d := state.Set(context.Background(), model); d.HasError() {
		t.Fatal(d)
	}
	return state, model
}

type ingressInternalRoundTripper func(*http.Request) (*http.Response, error)

func (f ingressInternalRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestIngressV1InternalReadUpdateTimeouts(t *testing.T) {
	for _, operation := range []string{"Read", "Update"} {
		for _, caller := range []string{"default", "earlier-deadline", "cancelled"} {
			t.Run(operation+"/"+caller, func(t *testing.T) {
				state, _ := ingressInternalExisting(t)
				ctx := context.Background()
				started := time.Now()
				expectedDeadline := started.Add(ingressDefaultTimeout)
				switch caller {
				case "earlier-deadline":
					expectedDeadline = started.Add(time.Minute)
					var cancel context.CancelFunc
					ctx, cancel = context.WithDeadline(ctx, expectedDeadline)
					defer cancel()
				case "cancelled":
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				}
				var observed context.Context
				client, err := kubeclient.NewForConfig(&rest.Config{
					Host: "https://ingress.invalid",
					Transport: ingressInternalRoundTripper(func(req *http.Request) (*http.Response, error) {
						observed = req.Context()
						if caller == "cancelled" {
							return nil, req.Context().Err()
						}
						deadline, ok := req.Context().Deadline()
						if !ok || deadline.Before(expectedDeadline) || deadline.After(expectedDeadline.Add(time.Second)) {
							t.Errorf("request deadline = %s (exists=%t), expected %s", deadline, ok, expectedDeadline)
						}
						return nil, fmt.Errorf("transport stopped before network access")
					}),
				})
				if err != nil {
					t.Fatal(err)
				}
				r := &IngressV1{SDKv2Meta: func() any { return ingressInternalClientsets{client: client} }}
				var d diag.Diagnostics
				switch operation {
				case "Read":
					resp := resource.ReadResponse{State: state}
					r.Read(ctx, resource.ReadRequest{State: state}, &resp)
					d = resp.Diagnostics
				case "Update":
					resp := resource.UpdateResponse{State: state}
					r.Update(ctx, resource.UpdateRequest{
						State: state, Plan: tfsdk.Plan(state),
					}, &resp)
					d = resp.Diagnostics
				}
				if !d.HasError() {
					t.Fatal("interrupted request silently succeeded")
				}
				if caller == "cancelled" && !strings.Contains(d.Errors()[0].Detail(), context.Canceled.Error()) {
					t.Fatalf("caller cancellation was lost: %v", d)
				}
				if caller != "cancelled" && (observed == nil || observed.Err() != context.Canceled) {
					t.Fatal("operation did not cancel its request context on return")
				}
			})
		}
	}
}

func TestIngressV1InternalUpdatePatchAndNoWait(t *testing.T) {
	ctx := context.Background()
	state, model := ingressInternalExisting(t)
	model.WaitForLoadBalancer = types.BoolValue(true)
	model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"managed": types.StringValue("new")})
	model.Spec[0].TLS = []IngressV1TLSModel{}
	model.Spec[0].IngressClassName = types.StringUnknown()
	planState := tfsdk.State{Schema: state.Schema}
	if d := planState.Set(ctx, model); d.HasError() {
		t.Fatal(d)
	}
	gets, patches := 0, 0
	r := ingressInternalResource(t, func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			gets++
		case http.MethodPatch:
			patches++
			var ops []struct {
				Op    string `json:"op"`
				Path  string `json:"path"`
				Value any    `json:"value"`
			}
			if err := json.NewDecoder(req.Body).Decode(&ops); err != nil {
				t.Error(err)
			}
			if len(ops) != 2 || ops[0].Path != "/metadata/labels/managed" || ops[1].Path != "/spec/tls" {
				t.Errorf("patch overwrote unowned fields: %#v", ops)
			}
		default:
			t.Errorf("unexpected request: %s", req.Method)
		}
		class := "admission-default"
		labels := map[string]string{"external": "retained"}
		if req.Method == http.MethodPatch {
			labels["managed"] = "new"
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(&networking.Ingress{
			TypeMeta: metav1.TypeMeta{APIVersion: ingressAPIVersion, Kind: ingressKind},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test", Namespace: "default", UID: "uid-1", ResourceVersion: "2", Generation: 2,
				Labels: labels,
			},
			Spec: networking.IngressSpec{IngressClassName: &class},
		}); err != nil {
			t.Error(err)
		}
	})
	resp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: planState.Raw}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if gets != 1 || patches != 1 {
		t.Fatalf("update waited for the empty load balancer: GET=%d PATCH=%d", gets, patches)
	}
	if d := resp.State.Get(ctx, &model); d.HasError() || model.Spec[0].IngressClassName.ValueString() != "admission-default" {
		t.Fatalf("computed class not resolved: %v %#v", d, model.Spec)
	}
	if len(model.Metadata[0].Labels.Elements()) != 1 {
		t.Fatal("unmanaged labels leaked into applied plan state")
	}
}

func TestIngressV1InternalReadDeleteErrors(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			ctx := context.Background()
			state, _ := ingressInternalExisting(t)
			r := ingressInternalResource(t, func(w http.ResponseWriter, req *http.Request) {
				http.Error(w, http.StatusText(status), status)
			})
			read := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &read)
			if status == http.StatusNotFound && (!read.State.Raw.IsNull() || read.Diagnostics.HasError()) {
				t.Fatalf("not-found read did not remove state: %v", read.Diagnostics)
			}
			if status == http.StatusForbidden && (!read.Diagnostics.HasError() || read.State.Raw.IsNull()) {
				t.Fatal("read error was hidden or state removed")
			}
			deleted := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &deleted)
			if deleted.Diagnostics.HasError() != (status == http.StatusForbidden) {
				t.Fatalf("delete diagnostics: %v", deleted.Diagnostics)
			}
		})
	}
}

func TestIngressV1InternalReadMetadataAndStatus(t *testing.T) {
	ctx := context.Background()
	state, model := ingressInternalExisting(t)
	model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{
		"ignored/managed": types.StringValue("old"), "removed": types.StringValue("old"),
	})
	if d := state.Set(ctx, model); d.HasError() {
		t.Fatal(d)
	}
	r := ingressInternalResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			t.Errorf("unexpected read method: %s", req.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(&networking.Ingress{
			TypeMeta: metav1.TypeMeta{APIVersion: ingressAPIVersion, Kind: ingressKind},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test", Namespace: "default", UID: "uid-1", ResourceVersion: "2",
				Labels: map[string]string{"ignored/managed": "new", "ignored/unmanaged": "hidden", "kubernetes.io/hidden": "hidden"},
			},
			Spec: networking.IngressSpec{Rules: []networking.IngressRule{{Host: "host-only"}}},
			Status: networking.IngressStatus{LoadBalancer: networking.IngressLoadBalancerStatus{
				Ingress: []networking.IngressLoadBalancerIngress{{IP: "192.0.2.1"}, {Hostname: "lb.example.com"}},
			}},
		}); err != nil {
			t.Error(err)
		}
	})
	response := resource.ReadResponse{State: state, Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()}}
	r.Read(ctx, resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if d := response.State.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	if labels := model.Metadata[0].Labels.Elements(); len(labels) != 1 || !labels["ignored/managed"].Equal(types.StringValue("new")) {
		t.Fatalf("metadata filtering/removal regression: %v", labels)
	}
	if model.Metadata[0].GenerateName.IsNull() || model.Spec[0].Rule[0].HTTP != nil {
		t.Fatal("read changed SDK zero/empty block semantics")
	}
	status := model.Status.Elements()[0].(types.Object).Attributes()
	loadBalancer := status["load_balancer"].(types.List).Elements()[0].(types.Object).Attributes()
	endpoints := loadBalancer["ingress"].(types.List).Elements()
	if len(endpoints) != 2 || !endpoints[0].(types.Object).Attributes()["hostname"].Equal(types.StringValue("")) {
		t.Fatalf("computed status list shape/zeros changed: %v", endpoints)
	}
}

func TestIngressV1InternalWaiter(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	gets := 0
	client.PrependReactor("get", "ingresses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gets++
		if gets == 1 {
			return true, nil, apierrors.NewNotFound(k8sschema.GroupResource{Group: "networking.k8s.io", Resource: "ingresses"}, "generated")
		}
		return true, &networking.Ingress{Status: networking.IngressStatus{LoadBalancer: networking.IngressLoadBalancerStatus{
			Ingress: []networking.IngressLoadBalancerIngress{{IP: "192.0.2.1"}},
		}}}, nil
	})
	if _, err := ingressWaitForLoadBalancer(ctx, client, "default", "generated", 3*time.Second); err != nil || gets < 2 {
		t.Fatalf("initial 404 not retried: %v GET=%d", err, gets)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := ingressWaitForLoadBalancer(ctx, client, "default", "generated", time.Second); err == nil {
		t.Fatal("cancelled waiter silently succeeded")
	}
}

func TestIngressV1InternalIdentityAndConfigure(t *testing.T) {
	ctx := context.Background()
	r := &IngressV1{}
	var configured resource.ConfigureResponse
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: func() kubernetes.KubeClientsets { return nil }}, &configured)
	if !configured.Diagnostics.HasError() {
		t.Fatal("invalid Configure callback accepted")
	}
	var identitySchema resource.IdentitySchemaResponse
	r.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identitySchema)
	if identitySchema.IdentitySchema.Version != 1 {
		t.Fatal("identity version downgraded")
	}
	identity := &tfsdk.ResourceIdentity{Schema: identitySchema.IdentitySchema}
	upgrade := resource.UpgradeIdentityResponse{Identity: identity}
	r.UpgradeIdentity(ctx)[0].IdentityUpgrader(ctx, resource.UpgradeIdentityRequest{
		RawIdentity: &tfprotov6.RawState{JSON: []byte(`{"name":"test","namespace":"default"}`)},
	}, &upgrade)
	if upgrade.Diagnostics.HasError() {
		t.Fatal(upgrade.Diagnostics)
	}
	s := ingressInternalSchema(t)
	for _, test := range []struct {
		name    string
		request resource.ImportStateRequest
	}{
		{"id", resource.ImportStateRequest{ID: "default/test"}},
		{"identity", resource.ImportStateRequest{Identity: identity}},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := resource.ImportStateResponse{
				State:    tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)},
				Identity: &tfsdk.ResourceIdentity{Schema: identitySchema.IdentitySchema},
			}
			r.ImportState(ctx, test.request, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			var imported IngressV1Model
			if d := response.State.Get(ctx, &imported); d.HasError() || imported.ID.ValueString() != "default/test" {
				t.Fatalf("import failed: %v %#v", d, imported)
			}
			if !imported.WaitForLoadBalancer.IsNull() {
				t.Fatalf("import must not invent a configured wait flag, got %s", imported.WaitForLoadBalancer)
			}
		})
	}
}

func TestIngressV1InternalValidationAndUnknownBlocks(t *testing.T) {
	ctx := context.Background()
	config := ingressInternalState(t, ingressInternalConfigJSON)
	spec := tftypes.NewAttributePath().WithAttributeName("spec")
	paths := spec.WithElementKeyInt(0).WithAttributeName("rule").WithElementKeyInt(0).
		WithAttributeName("http").WithAttributeName("path")
	port := paths.WithElementKeyInt(0).WithAttributeName("backend").
		WithAttributeName("service").WithAttributeName("port")
	for _, tc := range []struct {
		name  string
		path  *tftypes.AttributePath
		value any
		error bool
	}{
		{"missing spec", spec, nil, true},
		{"empty spec", spec, []tftypes.Value{}, true},
		{"unknown spec", spec, tftypes.UnknownValue, false},
		{"missing paths", paths, nil, true},
		{"empty paths", paths, []tftypes.Value{}, true},
		{"unknown paths", paths, tftypes.UnknownValue, false},
		{"missing port", port, nil, true},
		{"unknown port", port, tftypes.UnknownValue, false},
		{"invalid path type", paths.WithElementKeyInt(0).WithAttributeName("path_type"), "Unknown", true},
		{"unknown path type", paths.WithElementKeyInt(0).WithAttributeName("path_type"), tftypes.UnknownValue, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := tftypes.Transform(config.Raw, func(path *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
				if path.Equal(tc.path) {
					return tftypes.NewValue(value.Type(), tc.value), nil
				}
				return value, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			response, err := providerserver.NewProtocol6(ingressInternalProvider{})().ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: "kubernetes_ingress_v1", Config: ingressInternalDynamic(t, value),
			})
			if err != nil {
				t.Fatal(err)
			}
			hasError := false
			for _, d := range response.Diagnostics {
				hasError = hasError || d.Severity == tfprotov6.DiagnosticSeverityError
			}
			if hasError != tc.error {
				t.Fatalf("expected error %t, diagnostics: %v", tc.error, response.Diagnostics)
			}
		})
	}
}

func TestIngressV1InternalRemovedScalarsPlanDefaults(t *testing.T) {
	ctx := context.Background()
	state, prior := ingressInternalExisting(t)
	prior.WaitForLoadBalancer = types.BoolValue(true)
	prior.Spec[0].Rule[0].Host = types.StringValue("example.com")
	prior.Spec[0].Rule[0].HTTP.Path[0].Path = types.StringValue("/prefix")
	prior.Spec[0].Rule[0].HTTP.Path[0].PathType = types.StringValue("Prefix")
	prior.Spec[0].TLS[0].SecretName = types.StringValue("certificate")
	prior.Spec[0].TLS[0].Hosts = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("example.com")})
	if d := state.Set(ctx, prior); d.HasError() {
		t.Fatal(d)
	}
	config := ingressInternalState(t, ingressInternalConfigJSON)
	response, err := providerserver.NewProtocol6(ingressInternalProvider{})().PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "kubernetes_ingress_v1",
		Config:           ingressInternalDynamic(t, config.Raw),
		PriorState:       ingressInternalDynamic(t, state.Raw),
		ProposedNewState: ingressInternalDynamic(t, config.Raw),
	})
	if err != nil {
		t.Fatal(err)
	}
	ingressInternalProtocolDiagnostics(t, response.Diagnostics)
	if len(response.RequiresReplace) != 0 {
		t.Fatalf("mutable spec fields required replacement: %v", response.RequiresReplace)
	}
	value, err := response.PlannedState.Unmarshal(state.Raw.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := tfsdk.Plan{Schema: state.Schema, Raw: value}
	var got IngressV1Model
	if d := plan.Get(ctx, &got); d.HasError() {
		t.Fatal(d)
	}
	path := got.Spec[0].Rule[0].HTTP.Path[0]
	if got.WaitForLoadBalancer.ValueBool() || got.Spec[0].Rule[0].Host.ValueString() != "" ||
		path.Path.ValueString() != "" || path.PathType.ValueString() != "ImplementationSpecific" ||
		got.Spec[0].TLS[0].SecretName.ValueString() != "" || !got.Spec[0].TLS[0].Hosts.IsNull() {
		t.Fatalf("removed scalars retained old values: %#v", got)
	}
}

func TestIngressV1InternalDeleteWaitAndTimeout(t *testing.T) {
	for _, tc := range []struct {
		name      string
		getStatus int
		wantError bool
	}{
		{"deleted", http.StatusNotFound, false},
		{"permission error after delete", http.StatusForbidden, true},
		{"still exists", http.StatusOK, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			state, model := ingressInternalExisting(t)
			model.Timeouts.Object = types.ObjectValueMust(model.Timeouts.AttributeTypes(ctx), map[string]attr.Value{
				"create": types.StringNull(), "delete": types.StringValue("100ms"),
			})
			if d := state.Set(ctx, model); d.HasError() {
				t.Fatal(d)
			}
			gets := 0
			r := ingressInternalResource(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodDelete {
					w.WriteHeader(http.StatusOK)
					return
				}
				gets++
				if tc.getStatus != http.StatusOK {
					http.Error(w, http.StatusText(tc.getStatus), tc.getStatus)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"apiVersion":"networking.k8s.io/v1","kind":"Ingress","metadata":{"name":"test","namespace":"default"}}`)
			})
			response := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &response)
			if response.Diagnostics.HasError() != tc.wantError || gets == 0 {
				t.Fatalf("delete waiter error=%t GET=%d: %v", tc.wantError, gets, response.Diagnostics)
			}
		})
	}
}

// Both providers use only this loopback API; Registry access downloads the pinned
// released binary, and no Kubernetes credentials or cluster are used.
func TestIngressV1InternalReleasedUpgradeOffline(t *testing.T) {
	ingressInternalReleasedUpgradeOffline(t, false)
}

func TestIngressV1InternalReleasedRefreshedUpgradeOffline(t *testing.T) {
	// SDK refresh can persist empty collections where the initial apply kept null.
	ingressInternalReleasedUpgradeOffline(t, true)
}

func ingressInternalReleasedUpgradeOffline(t *testing.T, releasedRefresh bool) {
	if os.Getenv("KUBE_NETWORKING_CORE_TEST") != "1" {
		t.Skip("set KUBE_NETWORKING_CORE_TEST=1 for local fake-API Terraform CLI tests")
	}
	t.Helper()
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("Terraform CLI is not installed")
	}
	temporary := os.TempDir()
	private := t.TempDir()
	for _, variable := range os.Environ() {
		key, _, _ := strings.Cut(variable, "=")
		if strings.HasPrefix(key, "TF_") {
			t.Setenv(key, "")
		}
	}
	// The private test path is too long for go-plugin Unix sockets on Darwin.
	for _, key := range []string{"TMPDIR", "TMP", "TEMP", "GOTMPDIR"} {
		t.Setenv(key, temporary)
	}
	rc := filepath.Join(private, "terraform.tfrc")
	if err := os.WriteFile(rc, []byte("provider_installation { direct {} }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TF_CLI_CONFIG_FILE", rc)
	t.Setenv("TF_ACC_TERRAFORM_PATH", terraformPath)
	t.Setenv("TF_ACC_TEMP_DIR", private)
	for _, key := range []string{
		"KUBECONFIG", "KUBE_CONFIG_PATH", "KUBE_CONFIG_PATHS", "KUBE_CTX", "KUBE_CTX_AUTH_INFO", "KUBE_CTX_CLUSTER",
		"KUBE_USER", "KUBE_PASSWORD", "KUBE_TOKEN", "KUBE_CLIENT_CERT_DATA", "KUBE_CLIENT_KEY_DATA",
		"KUBE_CLUSTER_CA_CERT_DATA", "KUBE_PROXY_URL", "KUBE_TLS_SERVER_NAME", "TF_REATTACH_PROVIDERS",
	} {
		t.Setenv(key, "")
	}
	const backend = `
		default_backend {
		  service {
		    name = "backend"
		    port { number = 80 }
		  }
		}
	`
	for _, tc := range []struct {
		name, metadata, spec, extra string
		normalizeMetadata           bool
	}{
		{name: "minimal", spec: backend},
		{
			name: "full", metadata: `labels = { app = "ingress" }
				annotations = { "example.com/managed" = "true" }`,
			spec: backend + `
				ingress_class_name = "example"
				rule {
				  host = "app.example.com"
				  http {
				    path {
				      path = "/app"
				      path_type = "Prefix"
				      backend {
				        service {
				          name = "app"
				          port { name = "http" }
				        }
				      }
				    }
				    path {
				      path = "/bucket"
				      path_type = "Exact"
				      backend {
				        resource {
				          api_group = "example.com"
				          kind = "StorageBucket"
				          name = "bucket"
				        }
				      }
				    }
				  }
				}
				rule { host = "host-only.example.com" }
				tls {
				  hosts = ["app.example.com", "host-only.example.com"]
				  secret_name = "tls-secret"
				}
			`,
			extra: `wait_for_load_balancer = true
				timeouts {
				  create = "2m"
				  delete = "2m"
				}`,
		},
		{name: "tls-hosts-omitted", spec: backend + `tls {}`, extra: "wait_for_load_balancer = false"},
		{name: "tls-hosts-empty", spec: backend + `tls { hosts = [] }`, extra: "wait_for_load_balancer = false"},
		{
			name: "tls-hosts-mixed", spec: backend + `
				tls { secret_name = "omitted" }
				tls {
				  hosts = []
				  secret_name = "empty"
				}
				tls {
				  hosts = ["app.example.com"]
				  secret_name = "full"
				}
			`, extra: "wait_for_load_balancer = false",
		},
		{
			name: "metadata-empty", metadata: "labels = {}\nannotations = {}",
			spec: backend, extra: "wait_for_load_balancer = false", normalizeMetadata: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var object *networking.Ingress
			creates, writes, deletes := 0, 0, 0
			name := fmt.Sprintf("offline-ingress-%d", time.Now().UnixNano())
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				base := "/apis/networking.k8s.io/v1/namespaces/default/ingresses"
				if req.URL.Path != base && req.URL.Path != base+"/"+name {
					t.Errorf("unexpected fake API request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				switch req.Method {
				case http.MethodPost:
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					decoded, _, err := kubescheme.Codecs.UniversalDeserializer().Decode(body, nil, nil)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					var ok bool
					object, ok = decoded.(*networking.Ingress)
					if !ok {
						t.Errorf("unexpected object type %T", decoded)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					creates++
					object.APIVersion, object.Kind = ingressAPIVersion, ingressKind
					object.UID, object.ResourceVersion, object.Generation = "offline-ingress-uid", "1", 1
					object.Status.LoadBalancer.Ingress = []networking.IngressLoadBalancerIngress{
						{IP: "192.0.2.10"}, {Hostname: "lb.example.com"},
					}
				case http.MethodGet:
					if object == nil {
						w.WriteHeader(http.StatusNotFound)
						fmt.Fprint(w, `{"apiVersion":"v1","kind":"Status","reason":"NotFound","code":404}`)
						return
					}
				case http.MethodDelete:
					deletes++
					object = nil
					fmt.Fprint(w, `{"apiVersion":"v1","kind":"Status","status":"Success"}`)
					return
				default:
					writes++
					t.Errorf("unchanged upgrade unexpectedly wrote ingress: %s", req.Method)
					w.WriteHeader(http.StatusConflict)
					return
				}
				if err := json.NewEncoder(w).Encode(object); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			t.Setenv("KUBE_HOST", server.URL)
			t.Setenv("KUBE_INSECURE", "false")
			client, err := kubeclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			meta := func() any { return ingressInternalClientsets{client: client} }
			factories := map[string]func() (tfprotov6.ProviderServer, error){
				"kubernetes": providerserver.NewProtocol6WithError(ingressInternalProvider{meta: meta}),
			}
			config := fmt.Sprintf(`
				resource "kubernetes_ingress_v1" "test" {
				  metadata {
				    name = %q
				    %s
				  }
				  spec {
				    %s
				  }
				  %s
				}
			`, name, tc.metadata, tc.spec, tc.extra)
			targetConfig := NetworkingTargetTestConfig(config)
			var before *networking.Ingress
			var snapshot ingressInternalCoreSnapshot
			checkUnchanged := func(*terraform.State) error {
				mu.Lock()
				defer mu.Unlock()
				if !reflect.DeepEqual(object, before) || creates != 1 || writes != 0 || deletes != 0 {
					return fmt.Errorf("unchanged upgrade altered ingress UID, payload, or lifecycle: creates=%d writes=%d deletes=%d", creates, writes, deletes)
				}
				return nil
			}
			checks := []plancheck.PlanCheck{
				plancheck.ExpectEmptyPlan(), ingressInternalCorePlanCheck{t: t},
			}
			normalizeMetadata := tc.normalizeMetadata && !releasedRefresh
			if normalizeMetadata {
				checks = []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("kubernetes_ingress_v1.test", plancheck.ResourceActionUpdate),
					ingressInternalCorePlanCheck{t: t, normalizeMetadata: true},
				}
			}
			steps := []tfresource.TestStep{
				{
					ExternalProviders: map[string]tfresource.ExternalProvider{
						"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "3.2.1"},
					},
					Config: config,
					Check: func(*terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if object == nil || creates != 1 {
							return fmt.Errorf("released provider did not create exactly one ingress")
						}
						before = object.DeepCopy()
						return nil
					},
					ConfigStateChecks: []statecheck.StateCheck{
						ingressInternalCoreStateCheck{snapshot: &snapshot},
					},
				},
				{
					ProtoV6ProviderFactories: factories,
					Config:                   targetConfig,
					ConfigPlanChecks:         tfresource.ConfigPlanChecks{PreApply: checks},
					Check:                    checkUnchanged,
					ConfigStateChecks: []statecheck.StateCheck{
						ingressInternalCoreStateCheck{snapshot: &snapshot, compare: true, normalizeMetadata: normalizeMetadata},
					},
				},
				{
					ProtoV6ProviderFactories: factories,
					Config:                   targetConfig,
					ConfigPlanChecks: tfresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					Check: checkUnchanged,
					ConfigStateChecks: []statecheck.StateCheck{
						ingressInternalCoreStateCheck{snapshot: &snapshot, compare: true, normalizeMetadata: normalizeMetadata},
					},
				},
			}
			if releasedRefresh {
				refresh := tfresource.TestStep{
					ExternalProviders: steps[0].ExternalProviders,
					Config:            config,
					ConfigPlanChecks: tfresource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					Check: checkUnchanged,
					ConfigStateChecks: []statecheck.StateCheck{
						ingressInternalCoreStateCheck{snapshot: &snapshot},
					},
				}
				steps = append([]tfresource.TestStep{steps[0], refresh}, steps[1:]...)
			}
			t.Cleanup(func() {
				mu.Lock()
				defer mu.Unlock()
				if creates != 1 || deletes != 1 || writes != 0 || object != nil {
					t.Errorf("unexpected fake lifecycle: creates=%d deletes=%d writes=%d object=%#v", creates, deletes, writes, object)
				}
			})
			tfresource.Test(t, tfresource.TestCase{
				IsUnitTest: true,
				CheckDestroy: func(*terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if object != nil {
						return fmt.Errorf("fake ingress still exists")
					}
					return nil
				},
				Steps: steps,
			})
		})
	}
}

type ingressInternalCoreSnapshot struct {
	values, identity map[string]interface{}
	providerName     string
}

type ingressInternalCoreStateCheck struct {
	snapshot          *ingressInternalCoreSnapshot
	compare           bool
	normalizeMetadata bool
}

func (c ingressInternalCoreStateCheck) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	if req.State == nil || req.State.Values == nil || req.State.Values.RootModule == nil {
		resp.Error = fmt.Errorf("missing ingress state")
		return
	}
	for _, found := range req.State.Values.RootModule.Resources {
		if found.Address != "kubernetes_ingress_v1.test" {
			continue
		}
		expectedVersion := uint64(0)
		if c.compare {
			expectedVersion = 1
		}
		if found.SchemaVersion != expectedVersion || found.IdentitySchemaVersion == nil || *found.IdentitySchemaVersion != 1 {
			resp.Error = fmt.Errorf("resource/identity schema version changed: resource=%d identity=%v", found.SchemaVersion, found.IdentitySchemaVersion)
			return
		}
		got := ingressInternalCoreSnapshot{
			values: found.AttributeValues, identity: found.IdentityValues, providerName: found.ProviderName,
		}
		if !c.compare {
			*c.snapshot = got
			return
		}
		if c.normalizeMetadata {
			got.values = maps.Clone(got.values)
			before := c.snapshot.values["metadata"].([]interface{})[0].(map[string]interface{})
			after := maps.Clone(got.values["metadata"].([]interface{})[0].(map[string]interface{}))
			got.values["metadata"] = []interface{}{after}
			for _, field := range []string{"labels", "annotations"} {
				if before[field] != nil || !reflect.DeepEqual(after[field], map[string]interface{}{}) {
					resp.Error = fmt.Errorf("metadata state normalization was not exactly null to empty: %s", field)
					return
				}
				after[field] = nil
			}
		}
		expected := *c.snapshot
		expected.values = NetworkingTargetTestValues(expected.values)
		if !reflect.DeepEqual(expected, got) {
			differences := map[string]string{}
			for key, before := range c.snapshot.values {
				if after := got.values[key]; !reflect.DeepEqual(before, after) {
					differences[key] = fmt.Sprintf("%#v -> %#v", before, after)
				}
			}
			resp.Error = fmt.Errorf("upgrade altered ingress state, identity, or provider: differences=%v identity=%v -> %v provider=%q -> %q",
				differences, c.snapshot.identity, got.identity, c.snapshot.providerName, got.providerName)
		}
		return
	}
	resp.Error = fmt.Errorf("ingress is absent from Terraform state")
}

type ingressInternalCorePlanCheck struct {
	t                 *testing.T
	normalizeMetadata bool
}

func (c ingressInternalCorePlanCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, change := range req.Plan.ResourceChanges {
		if change.Change == nil || change.Change.Actions.NoOp() {
			continue
		}
		raw, err := json.Marshal(change.Change)
		if err != nil {
			resp.Error = err
			return
		}
		c.t.Logf("Core ingress change: %s", raw)
		if !c.normalizeMetadata {
			continue
		}
		if change.Address != "kubernetes_ingress_v1.test" || len(change.Change.Actions) != 1 || change.Change.Actions[0] != "update" {
			resp.Error = fmt.Errorf("metadata normalization changed unexpected resource or action: %s %s", change.Address, raw)
			return
		}
		var values struct {
			Before map[string]interface{}
			After  map[string]interface{}
		}
		if err := json.Unmarshal(raw, &values); err != nil {
			resp.Error = err
			return
		}
		before, after := values.Before, values.After
		beforeMetadata := before["metadata"].([]interface{})[0].(map[string]interface{})
		afterMetadata := after["metadata"].([]interface{})[0].(map[string]interface{})
		for _, field := range []string{"labels", "annotations"} {
			if beforeMetadata[field] != nil || !reflect.DeepEqual(afterMetadata[field], map[string]interface{}{}) {
				resp.Error = fmt.Errorf("metadata normalization was not exactly null to empty for %s: %s", field, raw)
				return
			}
			afterMetadata[field] = nil
		}
		// Framework marks computed outputs unknown on this state-only update;
		// the post-apply state comparison verifies they retain their exact values.
		for _, field := range []string{"id", "status"} {
			if after[field] == nil {
				after[field] = before[field]
			}
		}
		for _, field := range []string{"generation", "resource_version", "uid"} {
			if afterMetadata[field] == nil {
				afterMetadata[field] = beforeMetadata[field]
			}
		}
		beforeSpec := before["spec"].([]interface{})[0].(map[string]interface{})
		afterSpec := after["spec"].([]interface{})[0].(map[string]interface{})
		if afterSpec["ingress_class_name"] == nil {
			afterSpec["ingress_class_name"] = beforeSpec["ingress_class_name"]
		}
		if !reflect.DeepEqual(before, after) {
			resp.Error = fmt.Errorf("metadata normalization changed other fields: %s", raw)
			return
		}
	}
}

type networkPolicyTestProvider struct {
	meta func() any
}

func (networkPolicyTestProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
}

func (networkPolicyTestProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}

func (p networkPolicyTestProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.meta
}

func (networkPolicyTestProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewNetworkPolicyV1}
}

func (networkPolicyTestProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}

const networkPolicyTestConfig = `{
	"id":null,
	"metadata":[{"name":"example","namespace":null,"generate_name":null,"annotations":null,"labels":null,
	"generation":null,"resource_version":null,"uid":null}],
	"spec":[{"pod_selector":[{"match_labels":null,"match_expressions":[]}],
	"policy_types":["Egress","Ingress"],"egress":[],
	"ingress":[{"ports":[{"port":null,"end_port":null,"protocol":null}],
	"from":[{"ip_block":[{"cidr":null,"except":null}],"namespace_selector":[],"pod_selector":[]}]}]}]
}`

const networkPolicyTestStoredState = `{
	"id":"default/example",
	"metadata":[{"name":"example","namespace":"default","generate_name":"","annotations":null,"labels":null,
	"generation":1,"resource_version":"10","uid":"uid-1"}],
	"spec":[{"pod_selector":[{"match_labels":{},"match_expressions":[
	{"key":"tier","operator":"In","values":["web","api"]}]}],
	"policy_types":["Egress","Ingress"],"egress":[],
	"ingress":[{"ports":[{"port":"0080","end_port":90,"protocol":"TCP"},
	{"port":"dns","end_port":0,"protocol":"UDP"}],
	"from":[{"ip_block":[],"namespace_selector":[{"match_labels":null,"match_expressions":[]}],
	"pod_selector":[]},{"ip_block":[{"cidr":"10.0.0.0/8","except":[]}],
	"namespace_selector":[],"pod_selector":[]}]}]}]
}`

func networkPolicyTestState(t *testing.T, raw string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var response resource.SchemaResponse
	NewNetworkPolicyV1().Schema(ctx, resource.SchemaRequest{}, &response)
	if raw == networkPolicyTestStoredState {
		return tfsdk.State{Schema: response.Schema, Raw: networkingTestUpgradedState(t, providerserver.NewProtocol6(networkPolicyTestProvider{})(), "kubernetes_network_policy_v1", raw, response.Schema.Type().TerraformType(ctx))}
	}
	value, err := (&tfprotov6.RawState{JSON: networkingTestFixtureJSON(t, []byte(raw), response.Schema.Type().TerraformType(ctx))}).Unmarshal(response.Schema.Type().TerraformType(ctx))
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: response.Schema, Raw: value}
}

func networkPolicyTestDynamic(t *testing.T, value tftypes.Value) *tfprotov6.DynamicValue {
	t.Helper()
	dynamic, err := tfprotov6.NewDynamicValue(value.Type(), value)
	if err != nil {
		t.Fatal(err)
	}
	return &dynamic
}

func networkPolicyTestProtocolErrors(t *testing.T, diags []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", d.Summary, d.Detail)
		}
	}
}

func networkPolicyTestBlockShape(t *testing.T, location string, got, want *tfprotov6.SchemaBlock) {
	t.Helper()
	for _, expected := range want.BlockTypes {
		selected := expected.TypeName == "pod_selector" || expected.TypeName == "namespace_selector" || expected.TypeName == "ip_block"
		if selected {
			found := false
			for _, attribute := range got.Attributes {
				if attribute.Name == expected.TypeName {
					_, found = attribute.ValueType().(tftypes.Object)
				}
			}
			if !found {
				t.Fatalf("%s.%s must be a singleton object", location, expected.TypeName)
			}
			continue
		}
		var actual *tfprotov6.SchemaNestedBlock
		for _, candidate := range got.BlockTypes {
			if candidate.TypeName == expected.TypeName {
				actual = candidate
				break
			}
		}
		if actual == nil || actual.Nesting != expected.Nesting {
			t.Fatalf("%s.%s: genuine collection shape changed", location, expected.TypeName)
		}
		networkPolicyTestBlockShape(t, location+"."+expected.TypeName, actual.Block, expected.Block)
	}
}

func TestNetworkPolicySchemaAndPlan(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(networkPolicyTestProvider{})()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	networkPolicyTestProtocolErrors(t, schemas.Diagnostics)
	sdk, err := tf5to6server.UpgradeServer(ctx, kubernetes.Provider().GRPCProvider)
	if err != nil {
		t.Fatal(err)
	}
	sdkSchemas, err := sdk.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	networkPolicyTestProtocolErrors(t, sdkSchemas.Diagnostics)
	typ := schemas.ResourceSchemas["kubernetes_network_policy_v1"].ValueType()
	if typ.Equal(sdkSchemas.ResourceSchemas["kubernetes_network_policy"].ValueType()) {
		t.Fatal("singleton conversion must change the state type")
	}
	networkPolicyTestBlockShape(t, "network_policy", schemas.ResourceSchemas["kubernetes_network_policy_v1"].Block,
		sdkSchemas.ResourceSchemas["kubernetes_network_policy"].Block)
	if schemas.ResourceSchemas["kubernetes_network_policy_v1"].Version != 1 {
		t.Fatal("schema version changed")
	}
	config := networkPolicyTestState(t, networkPolicyTestConfig)
	validated, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
		TypeName: "kubernetes_network_policy_v1", Config: networkPolicyTestDynamic(t, config.Raw),
	})
	if err != nil {
		t.Fatal(err)
	}
	// The SDK does not validate the CIDR, protocol, port range, or selector
	// expression scalars. Empty peer and selector blocks must remain accepted.
	networkPolicyTestProtocolErrors(t, validated.Diagnostics)
	planned, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName: "kubernetes_network_policy_v1", Config: networkPolicyTestDynamic(t, config.Raw),
		PriorState:       networkPolicyTestDynamic(t, tftypes.NewValue(typ, nil)),
		ProposedNewState: networkPolicyTestDynamic(t, config.Raw),
	})
	if err != nil {
		t.Fatal(err)
	}
	networkPolicyTestProtocolErrors(t, planned.Diagnostics)
	raw, err := planned.PlannedState.Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	state := tfsdk.State{Schema: config.Schema, Raw: raw}
	var model NetworkPolicyV1Model
	if d := state.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	port := model.Spec[0].Ingress[0].Ports[0]
	if port.Port.ValueString() != "" || port.Port.IsNull() || port.EndPort.IsNull() ||
		port.EndPort.ValueInt64() != 0 || port.Protocol.ValueString() != "TCP" ||
		model.Metadata[0].Namespace.ValueString() != "default" || model.Metadata[0].GenerateName.IsNull() ||
		model.Spec[0].Ingress[0].From[0].IPBlock.CIDR.IsNull() {
		t.Fatalf("SDK defaults not preserved: %#v", model)
	}
	stored := networkPolicyTestState(t, networkPolicyTestStoredState)
	// Omitted computed fields are represented by null in configuration, but
	// Terraform's proposed state carries their existing values.
	var priorModel NetworkPolicyV1Model
	if d := stored.Get(ctx, &priorModel); d.HasError() {
		t.Fatal(d)
	}
	priorModel.ID = types.StringNull()
	priorModel.Metadata[0].GenerateName = types.StringNull()
	priorModel.Metadata[0].Generation = types.Int64Null()
	priorModel.Metadata[0].UID = types.StringNull()
	priorModel.Metadata[0].ResourceVersion = types.StringNull()
	unchangedConfig := tfsdk.State{Schema: stored.Schema}
	if d := unchangedConfig.Set(ctx, &priorModel); d.HasError() {
		t.Fatal(d)
	}
	unchanged, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName: "kubernetes_network_policy_v1", Config: networkPolicyTestDynamic(t, unchangedConfig.Raw),
		PriorState: networkPolicyTestDynamic(t, stored.Raw), ProposedNewState: networkPolicyTestDynamic(t, stored.Raw),
	})
	if err != nil {
		t.Fatal(err)
	}
	networkPolicyTestProtocolErrors(t, unchanged.Diagnostics)
	newState, err := unchanged.PlannedState.Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	if !newState.Equal(stored.Raw) || len(unchanged.RequiresReplace) != 0 {
		t.Fatalf("unchanged SDK state changed: %s", newState)
	}
}

func TestNetworkPolicyPlanChanges(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(networkPolicyTestProvider{})()
	for _, tc := range []struct {
		name    string
		change  func(*NetworkPolicyV1Model)
		replace bool
	}{
		{"remove optional scalars", func(m *NetworkPolicyV1Model) {
			port := &m.Spec[0].Ingress[0].Ports[0]
			port.Port, port.Protocol, port.EndPort = types.StringNull(), types.StringNull(), types.Int64Null()
		}, false},
		{"unknown port", func(m *NetworkPolicyV1Model) {
			m.Spec[0].Ingress[0].Ports[0].Port = types.StringUnknown()
		}, false},
		{"remove rules", func(m *NetworkPolicyV1Model) {
			m.Spec[0].Ingress = []networkPolicyIngressModel{}
		}, false},
		{"empty pod selector", func(m *NetworkPolicyV1Model) {
			m.Spec[0].PodSelector = &networkPolicySelectorModel{
				MatchLabels: types.MapNull(types.StringType), MatchExpressions: []networkPolicyExpressionModel{},
			}
		}, false},
		{"change name", func(m *NetworkPolicyV1Model) {
			m.Metadata[0].Name = types.StringValue("renamed")
		}, true},
		{"change namespace", func(m *NetworkPolicyV1Model) {
			m.Metadata[0].Namespace = types.StringValue("other")
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior := networkPolicyTestState(t, networkPolicyTestStoredState)
			var model NetworkPolicyV1Model
			if d := prior.Get(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			tc.change(&model)
			proposed := tfsdk.State{Schema: prior.Schema}
			if d := proposed.Set(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			model.ID = types.StringNull()
			model.Metadata[0].UID = types.StringNull()
			model.Metadata[0].ResourceVersion = types.StringNull()
			model.Metadata[0].Generation = types.Int64Null()
			model.Metadata[0].GenerateName = types.StringNull()
			config := tfsdk.State{Schema: prior.Schema}
			if d := config.Set(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_network_policy_v1", Config: networkPolicyTestDynamic(t, config.Raw),
				PriorState: networkPolicyTestDynamic(t, prior.Raw), ProposedNewState: networkPolicyTestDynamic(t, proposed.Raw),
			})
			if err != nil {
				t.Fatal(err)
			}
			networkPolicyTestProtocolErrors(t, resp.Diagnostics)
			if (len(resp.RequiresReplace) > 0) != tc.replace {
				t.Fatalf("wrong replacement paths: %v", resp.RequiresReplace)
			}
			raw, err := resp.PlannedState.Unmarshal(prior.Raw.Type())
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: prior.Schema, Raw: raw}
			if d := state.Get(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			switch tc.name {
			case "remove optional scalars":
				port := model.Spec[0].Ingress[0].Ports[0]
				if !port.Port.Equal(types.StringValue("")) || !port.EndPort.Equal(types.Int64Value(0)) ||
					!port.Protocol.Equal(types.StringValue("TCP")) {
					t.Fatalf("removal did not restore defaults: %#v", port)
				}
			case "unknown port":
				if !model.Spec[0].Ingress[0].Ports[0].Port.IsUnknown() {
					t.Fatal("planning replaced an unknown port")
				}
			}
		})
	}
}

func TestNetworkPolicySDKImplicitDefaults(t *testing.T) {
	legacy := kubernetes.Provider().ResourcesMap["kubernetes_network_policy"]
	data := sdkschema.TestResourceDataRaw(t, legacy.Schema, map[string]interface{}{
		"spec": []interface{}{map[string]interface{}{
			"pod_selector": []interface{}{map[string]interface{}{}},
			"policy_types": []interface{}{"Ingress"},
			"ingress": []interface{}{map[string]interface{}{
				"ports": []interface{}{map[string]interface{}{}},
			}},
		}},
	})
	if data.Get("spec.0.ingress.0.ports.0.protocol") != "TCP" {
		t.Fatal("SDK configuration did not default protocol to TCP")
	}
	// A Read that receives a nil API protocol omits it from d.Set. That writes
	// the zero string on import, not the configuration default.
	data = sdkschema.TestResourceDataRaw(t, legacy.Schema, map[string]interface{}{})
	if err := data.Set("spec", []interface{}{map[string]interface{}{
		"pod_selector": []interface{}{map[string]interface{}{
			"match_expressions": []interface{}{map[string]interface{}{}},
		}},
		"policy_types": []interface{}{"Ingress"},
		"ingress": []interface{}{map[string]interface{}{
			"ports": []interface{}{map[string]interface{}{}},
		}},
	}}); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]interface{}{
		"spec.0.ingress.0.ports.0.port":                      "",
		"spec.0.ingress.0.ports.0.end_port":                  0,
		"spec.0.ingress.0.ports.0.protocol":                  "",
		"spec.0.pod_selector.0.match_expressions.0.key":      "",
		"spec.0.pod_selector.0.match_expressions.0.operator": "",
	} {
		if got := data.Get(name); !reflect.DeepEqual(got, want) {
			t.Errorf("SDK %s = %#v, want %#v", name, got, want)
		}
	}
	flattened := networkPolicyFlattenPorts([]networking.NetworkPolicyPort{{}}, nil)
	if flattened[0].Protocol.ValueString() != data.Get("spec.0.ingress.0.ports.0.protocol") {
		t.Fatal("Framework nil API protocol differs from SDK Read")
	}
}

func TestNetworkPolicyConversions(t *testing.T) {
	ctx := context.Background()
	state := networkPolicyTestState(t, networkPolicyTestStoredState)
	var model NetworkPolicyV1Model
	if d := state.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	spec, diags := networkPolicyExpandSpec(ctx, model.Spec)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if len(spec.Ingress) != 1 || spec.Ingress[0].From[0].NamespaceSelector == nil ||
		spec.Ingress[0].From[0].PodSelector != nil || len(spec.PodSelector.MatchExpressions[0].Values) != 2 ||
		spec.Ingress[0].Ports[0].Port.Type != intstr.Int || spec.Ingress[0].Ports[0].Port.IntVal != 80 ||
		spec.Ingress[0].Ports[1].Port.Type != intstr.String ||
		!reflect.DeepEqual(spec.PolicyTypes, []networking.PolicyType{networking.PolicyTypeEgress, networking.PolicyTypeIngress}) {
		t.Fatalf("incorrect API conversion: %#v", spec)
	}
	flattened, diags := networkPolicyFlattenSpec(ctx, spec, model.Spec)
	if diags.HasError() {
		t.Fatal(diags)
	}
	model.Spec = flattened
	roundTrip := tfsdk.State{Schema: state.Schema}
	if d := roundTrip.Set(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	if !roundTrip.Raw.Equal(state.Raw) {
		t.Fatalf("round trip changed state:\nold %s\nnew %s", state.Raw, roundTrip.Raw)
	}
	imported, diags := networkPolicyFlattenSpec(ctx, spec, nil)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if imported[0].Ingress[0].Ports[0].Port.ValueString() != "80" ||
		!imported[0].PodSelector.MatchLabels.IsNull() {
		t.Fatalf("unexpected imported port or selector: %#v", imported)
	}
	for _, present := range []bool{false, true} {
		var selector *networkPolicySelectorModel
		if present {
			selector = &networkPolicySelectorModel{MatchLabels: types.MapNull(types.StringType)}
		}
		got, d := networkPolicyExpandSelector(ctx, selector)
		if d.HasError() || (got != nil) != present {
			t.Fatalf("selector presence %t was lost: %#v, %s", present, got, d)
		}
	}
	ports := networkPolicyExpandPorts([]networkPolicyPortModel{
		{Port: types.StringValue(""), EndPort: types.Int64Value(0), Protocol: types.StringValue("")},
	})
	if ports[0].Port != nil || ports[0].EndPort != nil || ports[0].Protocol != nil {
		t.Fatalf("explicit zero values should be omitted in the API: %#v", ports)
	}
}

func TestNetworkPolicyMoveState(t *testing.T) {
	ctx := context.Background()
	r := &NetworkPolicyV1{}
	source := networkPolicyTestState(t, networkPolicyTestStoredState)
	for _, tc := range []struct {
		name, sourceType, provider, raw string
		version                         int64
		match, fail                     bool
	}{
		{"alias", "kubernetes_network_policy", "registry.terraform.io/hashicorp/kubernetes", networkPolicyTestStoredState, 0, true, false},
		{"mirror", "kubernetes_network_policy", "mirror.example/hashicorp/kubernetes", networkPolicyTestStoredState, 0, true, false},
		{"same type", "kubernetes_network_policy_v1", "registry.terraform.io/hashicorp/kubernetes", networkPolicyTestStoredState, 0, false, false},
		{"fork", "kubernetes_network_policy", "registry.terraform.io/nothashicorp/kubernetes", networkPolicyTestStoredState, 0, false, false},
		{"future", "kubernetes_network_policy", "registry.terraform.io/hashicorp/kubernetes", networkPolicyTestStoredState, 1, false, false},
		{"malformed", "kubernetes_network_policy", "registry.terraform.io/hashicorp/kubernetes", "{", 0, true, true},
		{"empty ID", "kubernetes_network_policy", "registry.terraform.io/hashicorp/kubernetes", `{}`, 0, true, true},
		{"unknown field", "kubernetes_network_policy", "registry.terraform.io/hashicorp/kubernetes", strings.Replace(networkPolicyTestStoredState, `"id":`, `"future_field":true,"id":`, 1), 0, true, true},
		{"trailing object", "kubernetes_network_policy", "registry.terraform.io/hashicorp/kubernetes", networkPolicyTestStoredState + `{}`, 0, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := resource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: source.Schema, Raw: tftypes.NewValue(source.Raw.Type(), nil)},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
			}
			r.MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
				SourceTypeName: tc.sourceType, SourceSchemaVersion: tc.version, SourceProviderAddress: tc.provider,
				SourceRawState: &tfprotov6.RawState{JSON: []byte(tc.raw)},
			}, &resp)
			if resp.Diagnostics.HasError() != tc.fail {
				t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
			}
			if tc.match && !tc.fail {
				if !resp.TargetState.Raw.Equal(source.Raw) {
					t.Fatalf("move lost state: %s", resp.TargetState.Raw)
				}
				var identity common.NamespacedResourceIdentity
				if d := resp.TargetIdentity.Get(ctx, &identity); d.HasError() {
					t.Fatal(d)
				}
				if identity.Name.ValueString() != "example" || identity.Namespace.ValueString() != "default" {
					t.Fatalf("wrong identity: %#v", identity)
				}
			} else if !resp.TargetState.Raw.IsNull() {
				t.Fatal("nonmatching or malformed move produced state")
			}
		})
	}
}

type networkPolicyTestClients struct {
	kubernetes.KubeClientsets
	client *kubeclient.Clientset
}

func (m networkPolicyTestClients) MainClientset() (*kubeclient.Clientset, error) {
	return m.client, nil
}

func (networkPolicyTestClients) GetIgnoreAnnotations() []string {
	return []string{"ignored.example/.*"}
}

func (networkPolicyTestClients) GetIgnoreLabels() []string { return []string{"ignored.example/.*"} }

func networkPolicyTestResource(t *testing.T, handler http.HandlerFunc) *NetworkPolicyV1 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := kubeclient.NewForConfig(&rest.Config{
		Host: server.URL,
		ContentConfig: rest.ContentConfig{
			ContentType: "application/json",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &NetworkPolicyV1{SDKv2Meta: func() any { return networkPolicyTestClients{client: client} }}
}

type networkPolicyTestRoundTripper func(*http.Request) (*http.Response, error)

func (f networkPolicyTestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNetworkPolicyCRUDTimeouts(t *testing.T) {
	initial := networkPolicyTestState(t, networkPolicyTestStoredState)
	plan := tfsdk.Plan(initial)
	for _, operation := range []string{"create", "read", "update", "delete"} {
		for _, caller := range []string{"default", "earlier-deadline", "cancelled"} {
			t.Run(operation+"/"+caller, func(t *testing.T) {
				ctx := context.Background()
				if caller == "earlier-deadline" {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, time.Minute)
					defer cancel()
				}
				if caller == "cancelled" {
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				}
				var observed context.Context
				client, err := kubeclient.NewForConfig(&rest.Config{
					Host: "http://127.0.0.1",
					Transport: networkPolicyTestRoundTripper(func(req *http.Request) (*http.Response, error) {
						observed = req.Context()
						return nil, context.Canceled
					}),
				})
				if err != nil {
					t.Fatal(err)
				}
				r := &NetworkPolicyV1{SDKv2Meta: func() any { return networkPolicyTestClients{client: client} }}
				var diagnostics diag.Diagnostics
				state := initial
				started := time.Now()
				switch operation {
				case "create":
					resp := resource.CreateResponse{State: initial}
					r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
					diagnostics, state = resp.Diagnostics, resp.State
				case "read":
					resp := resource.ReadResponse{State: initial}
					r.Read(ctx, resource.ReadRequest{State: initial}, &resp)
					diagnostics, state = resp.Diagnostics, resp.State
				case "update":
					resp := resource.UpdateResponse{State: initial}
					r.Update(ctx, resource.UpdateRequest{Plan: plan, State: initial}, &resp)
					diagnostics, state = resp.Diagnostics, resp.State
				case "delete":
					resp := resource.DeleteResponse{State: initial}
					r.Delete(ctx, resource.DeleteRequest{State: initial}, &resp)
					diagnostics, state = resp.Diagnostics, resp.State
				}
				finished := time.Now()
				if !diagnostics.HasError() || !strings.Contains(diagnostics.Errors()[0].Detail(), context.Canceled.Error()) {
					t.Fatalf("cancellation diagnostic missing: %s", diagnostics)
				}
				if !state.Raw.Equal(initial.Raw) {
					t.Fatal("cancellation changed prior state")
				}
				if caller == "cancelled" {
					return
				}
				if observed == nil {
					t.Fatal("Kubernetes request was not reached")
				}
				deadline, ok := observed.Deadline()
				if !ok {
					t.Fatal("SDKv2 operation deadline was lost")
				}
				if caller == "earlier-deadline" {
					expected, _ := ctx.Deadline()
					if !deadline.Equal(expected) {
						t.Fatalf("caller deadline changed: got %s, want %s", deadline, expected)
					}
				} else if deadline.Before(started.Add(20*time.Minute)) || deadline.After(finished.Add(20*time.Minute)) {
					t.Fatalf("default deadline is not 20 minutes: %s", deadline)
				}
				if observed.Err() != context.Canceled {
					t.Fatal("operation context was not cancelled on return")
				}
			})
		}
	}
}

func TestNetworkPolicyLifecycle(t *testing.T) {
	ctx := context.Background()
	var object *networking.NetworkPolicy
	r := networkPolicyTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case http.MethodPost:
			if err := json.NewDecoder(req.Body).Decode(&object); err != nil {
				t.Error(err)
				return
			}
			object.APIVersion, object.Kind = networkPolicyAPIVersion, networkPolicyKind
			object.UID, object.ResourceVersion, object.Generation = "uid-1", "10", 1
			object.Labels = map[string]string{"external": "keep", "ignored.example/label": "hidden"}
		case http.MethodGet:
			if object == nil {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","reason":"NotFound","code":404}`)
				return
			}
		case http.MethodPatch:
			var data json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
				t.Error(err)
				return
			}
			patch, err := jsonpatch.DecodePatch(data)
			if err != nil {
				t.Error(err)
				return
			}
			before, err := json.Marshal(object)
			if err != nil {
				t.Error(err)
				return
			}
			after, err := patch.Apply(before)
			if err != nil {
				t.Error(err)
				return
			}
			object = &networking.NetworkPolicy{}
			if err := json.Unmarshal(after, object); err != nil {
				t.Error(err)
				return
			}
			object.ResourceVersion = "11"
		case http.MethodDelete:
			object = nil
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Success"}`)
			return
		}
		if err := json.NewEncoder(w).Encode(object); err != nil {
			t.Error(err)
		}
	})
	initial := networkPolicyTestState(t, networkPolicyTestStoredState)
	create := resource.CreateResponse{
		State:    tfsdk.State{Schema: initial.Schema},
		Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
	}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(initial)}, &create)
	if create.Diagnostics.HasError() {
		t.Fatal(create.Diagnostics)
	}
	if !create.State.Raw.Equal(initial.Raw) {
		t.Fatalf("create changed planned values: %s", create.State.Raw)
	}
	var planned NetworkPolicyV1Model
	if d := create.State.Get(ctx, &planned); d.HasError() {
		t.Fatal(d)
	}
	planned.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"owned": types.StringValue("one")})
	planned.Spec[0].Ingress = []networkPolicyIngressModel{}
	planned.Spec[0].Egress = []networkPolicyEgressModel{{Ports: []networkPolicyPortModel{{
		Port: types.StringValue("53"), Protocol: types.StringValue("UDP"), EndPort: types.Int64Value(0),
	}}, To: []networkPolicyPeerModel{}}}
	plan := tfsdk.Plan{Schema: initial.Schema}
	if d := plan.Set(ctx, &planned); d.HasError() {
		t.Fatal(d)
	}
	update := resource.UpdateResponse{State: create.State, Identity: create.Identity}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: create.State}, &update)
	if update.Diagnostics.HasError() {
		t.Fatal(update.Diagnostics)
	}
	if object.Labels["external"] != "keep" || object.Labels["ignored.example/label"] != "hidden" ||
		object.Labels["owned"] != "one" || len(object.Spec.Ingress) != 0 || len(object.Spec.Egress) != 1 ||
		string(object.UID) != "uid-1" {
		t.Fatalf("update lost ownership or failed rule replacement: %#v", object)
	}
	read := resource.ReadResponse{State: update.State, Identity: update.Identity}
	r.Read(ctx, resource.ReadRequest{State: update.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	var refreshed NetworkPolicyV1Model
	if d := read.State.Get(ctx, &refreshed); d.HasError() {
		t.Fatal(d)
	}
	if _, ok := refreshed.Metadata[0].Labels.Elements()["ignored.example/label"]; ok {
		t.Fatal("ignored label leaked into state")
	}
	deleted := resource.DeleteResponse{State: read.State}
	r.Delete(ctx, resource.DeleteRequest{State: read.State}, &deleted)
	if deleted.Diagnostics.HasError() || object != nil {
		t.Fatalf("delete failed: %s", deleted.Diagnostics)
	}
	r.Read(ctx, resource.ReadRequest{State: read.State}, &read)
	if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
		t.Fatalf("confirmed absence retained state: %s", read.Diagnostics)
	}
}

func TestNetworkPolicyErrorsRetainState(t *testing.T) {
	ctx := context.Background()
	initial := networkPolicyTestState(t, networkPolicyTestStoredState)
	for _, status := range []int{http.StatusForbidden, http.StatusInternalServerError, http.StatusNotFound} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			r := networkPolicyTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if req.Method == http.MethodPost {
					var object networking.NetworkPolicy
					if err := json.NewDecoder(req.Body).Decode(&object); err != nil {
						t.Error(err)
						return
					}
					object.UID = "uid-1"
					object.ResourceVersion = "10"
					object.Generation = 1
					if err := json.NewEncoder(w).Encode(object); err != nil {
						t.Error(err)
					}
					return
				}
				w.WriteHeader(status)
				reason := http.StatusText(status)
				if status == http.StatusNotFound {
					reason = "NotFound"
				}
				fmt.Fprintf(w, `{"kind":"Status","apiVersion":"v1","reason":%q,"code":%d}`, reason, status)
			})
			create := resource.CreateResponse{
				State:    tfsdk.State{Schema: initial.Schema},
				Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
			}
			r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(initial)}, &create)
			if !create.Diagnostics.HasError() || create.State.Raw.IsNull() || create.Identity.Raw.IsNull() {
				t.Fatalf("partial create lost state or identity: %s", create.Diagnostics)
			}
			read := resource.ReadResponse{State: initial}
			r.Read(ctx, resource.ReadRequest{State: initial}, &read)
			if status == http.StatusNotFound {
				if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
					t.Fatalf("not found did not clear state: %s", read.Diagnostics)
				}
			} else if !read.Diagnostics.HasError() || !read.State.Raw.Equal(initial.Raw) {
				t.Fatalf("read failure lost state: %s", read.Diagnostics)
			}
			update := resource.UpdateResponse{State: initial}
			r.Update(ctx, resource.UpdateRequest{
				State: initial, Plan: tfsdk.Plan(initial),
			}, &update)
			if !update.Diagnostics.HasError() || !update.State.Raw.Equal(initial.Raw) {
				t.Fatalf("update read failure lost state: %s", update.Diagnostics)
			}
			deleted := resource.DeleteResponse{State: initial}
			r.Delete(ctx, resource.DeleteRequest{State: initial}, &deleted)
			if deleted.Diagnostics.HasError() != (status != http.StatusNotFound) {
				t.Fatalf("unexpected delete result: %s", deleted.Diagnostics)
			}
		})
	}
}

func TestNetworkPolicyImportAndConfigure(t *testing.T) {
	ctx := context.Background()
	r := &NetworkPolicyV1{}
	var configured resource.ConfigureResponse
	r.Configure(ctx, resource.ConfigureRequest{}, &configured)
	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: "wrong"}, &configured)
	if !configured.Diagnostics.HasError() {
		t.Fatal("incorrect configure callback was accepted")
	}
	identity := &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()}
	if d := identity.Set(ctx, networkPolicyIdentity("default", "example")); d.HasError() {
		t.Fatal(d)
	}
	for _, req := range []resource.ImportStateRequest{{ID: "default/example"}, {Identity: identity}} {
		resp := resource.ImportStateResponse{
			State:    networkPolicyTestState(t, "null"),
			Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
		}
		r.ImportState(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		var model NetworkPolicyV1Model
		if d := resp.State.Get(ctx, &model); d.HasError() {
			t.Fatal(d)
		}
		if model.ID.ValueString() != "default/example" {
			t.Fatalf("wrong imported ID: %s", model.ID)
		}
	}
}

func TestNetworkPolicyPortSpelling(t *testing.T) {
	for _, spelling := range []string{"80", "0080", "+80", "dns", "2147483648"} {
		t.Run(spelling, func(t *testing.T) {
			prior := []networkPolicyPortModel{{Port: types.StringValue(spelling), Protocol: types.StringValue("TCP")}}
			api := []networking.NetworkPolicyPort{{Port: ptr.To(intstr.Parse(spelling)), Protocol: ptr.To(corev1.ProtocolTCP)}}
			got := networkPolicyFlattenPorts(api, prior)
			if got[0].Port.ValueString() != spelling {
				t.Fatalf("known spelling %q was changed to %s", spelling, got[0].Port)
			}
		})
	}
	// Read must still detect real drift, rather than blindly retaining prior values.
	got := networkPolicyFlattenPorts([]networking.NetworkPolicyPort{{Port: ptr.To(intstr.FromInt32(81))}},
		[]networkPolicyPortModel{{Port: types.StringValue("0080")}})
	if got[0].Port.ValueString() != "81" {
		t.Fatal("port drift was hidden")
	}
	wideEndPort := []networkPolicyPortModel{{
		Port: types.StringValue("80"), EndPort: types.Int64Value(1<<32 + 90), Protocol: types.StringValue("TCP"),
	}}
	apiPorts := networkPolicyExpandPorts(wideEndPort)
	if *apiPorts[0].EndPort != 90 || !networkPolicyFlattenPorts(apiPorts, wideEndPort)[0].EndPort.Equal(wideEndPort[0].EndPort) {
		t.Fatal("SDK int32 conversion changed a known end_port")
	}
	selector, diags := networkPolicyFlattenSelector(context.Background(), &metav1.LabelSelector{}, nil)
	if diags.HasError() || selector == nil || !selector.MatchLabels.IsNull() {
		t.Fatalf("empty selector was lost: %#v %s", selector, diags)
	}
}

type networkPolicyOfflinePlanCheck struct {
	normalizeMetadata         bool
	normalizeSelectorMaps     bool
	normalizeExpressionValues bool
}

func (check networkPolicyOfflinePlanCheck) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if len(req.Plan.ResourceChanges) != 1 {
		resp.Error = fmt.Errorf("expected exactly one state-only collection update, got %d changes", len(req.Plan.ResourceChanges))
		return
	}
	change := req.Plan.ResourceChanges[0]
	if change.Address != "kubernetes_network_policy_v1.test" || change.Change == nil ||
		len(change.Change.Actions) != 1 || (!change.Change.Actions.NoOp() && change.Change.Actions[0] != "update") || len(change.Change.ReplacePaths) != 0 {
		resp.Error = fmt.Errorf("expected an in-place state-only collection update: %#v", change)
		return
	}
	beforeJSON, _ := json.Marshal(change.Change.Before)
	var expected map[string]any
	if err := json.Unmarshal(beforeJSON, &expected); err != nil {
		resp.Error = err
		return
	}
	metadata, ok := expected["metadata"].([]any)
	if !ok || len(metadata) != 1 {
		resp.Error = fmt.Errorf("unexpected metadata before update: %#v", expected["metadata"])
		return
	}
	values := metadata[0].(map[string]any)
	if check.normalizeMetadata {
		for _, field := range []string{"annotations", "labels"} {
			if values[field] != nil {
				resp.Error = fmt.Errorf("expected released empty %s to be null, got %#v", field, values[field])
				return
			}
			values[field] = map[string]any{}
		}
	}
	if err := networkPolicyExpectedSelectorNormalization(expected["spec"], check.normalizeSelectorMaps, check.normalizeExpressionValues); err != nil {
		resp.Error = err
		return
	}
	expected = NetworkingOmittedTestExpressions(expected)
	values = expected["metadata"].([]any)[0].(map[string]any)
	expectedBeforeComputed, _ := json.Marshal(expected)
	normalizing := string(expectedBeforeComputed) != string(beforeJSON)
	if normalizing != (change.Change.Actions[0] == "update") {
		resp.Error = fmt.Errorf("unexpected action for exact collection normalization: %v", change.Change.Actions)
		return
	}
	if !normalizing {
		plancheck.ExpectEmptyPlan().CheckPlan(ctx, req, resp)
		return
	}

	// These two read-only values become unknown on any Framework update, but
	// the post-apply state and fake API assertions still require exact equality.
	unknown := change.Change.AfterUnknown.(map[string]any)["metadata"].([]any)[0].(map[string]any)
	for _, field := range []string{"generation", "resource_version"} {
		if unknown[field] != true {
			resp.Error = fmt.Errorf("expected %s to be unknown until the state-only update", field)
			return
		}
		delete(values, field)
	}
	afterJSON, _ := json.Marshal(change.Change.After)
	expectedJSON, _ := json.Marshal(expected)
	if string(afterJSON) != string(expectedJSON) {
		resp.Error = fmt.Errorf("update changes more than the approved explicit empty collections:\nwant %s\ngot  %s", expectedJSON, afterJSON)
	}
}

func networkPolicyExpectedSelectorNormalization(value any, normalizeMaps, normalizeValues bool) error {
	switch node := value.(type) {
	case []any:
		for _, child := range node {
			if err := networkPolicyExpectedSelectorNormalization(child, normalizeMaps, normalizeValues); err != nil {
				return err
			}
		}
	case map[string]any:
		if current, exists := node["match_labels"]; exists && normalizeMaps {
			if current != nil {
				return fmt.Errorf("expected released explicit empty match_labels to be null, got %#v", current)
			}
			node["match_labels"] = map[string]any{}
		}
		if current, exists := node["values"]; exists && normalizeValues &&
			(node["operator"] == "Exists" || node["operator"] == "DoesNotExist") {
			if current != nil {
				return fmt.Errorf("expected released explicit empty expression values to be null, got %#v", current)
			}
			node["values"] = []any{}
		}
		for _, child := range node {
			if err := networkPolicyExpectedSelectorNormalization(child, normalizeMaps, normalizeValues); err != nil {
				return err
			}
		}
	}
	return nil
}

// Both provider versions are isolated from kubeconfig and use only a loopback
// fake API; real Terraform and the pinned Registry binary exercise Core planning.
func TestNetworkPolicyReleasedUpgradeOffline(t *testing.T) {
	if os.Getenv("KUBE_NETWORKING_CORE_TEST") != "1" {
		t.Skip("set KUBE_NETWORKING_CORE_TEST=1 for local fake-API Terraform CLI tests")
	}
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("Terraform CLI is not installed")
	}
	private := t.TempDir()
	rc := filepath.Join(private, "terraform.tfrc")
	if err := os.WriteFile(rc, []byte("provider_installation { direct {} }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(private, "provider-cache")
	if err := os.Mkdir(cache, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TF_CLI_CONFIG_FILE", rc)
	t.Setenv("TF_PLUGIN_CACHE_DIR", cache)
	t.Setenv("TF_ACC_TERRAFORM_PATH", terraformPath)
	t.Setenv("TF_ACC_TEMP_DIR", private)
	for _, key := range []string{
		"KUBECONFIG", "KUBE_CONFIG_PATH", "KUBE_CONFIG_PATHS", "KUBE_CTX", "KUBE_CTX_AUTH_INFO", "KUBE_CTX_CLUSTER",
		"KUBE_USER", "KUBE_PASSWORD", "KUBE_TOKEN", "KUBE_CLIENT_CERT_DATA", "KUBE_CLIENT_KEY_DATA",
		"KUBE_CLUSTER_CA_CERT_DATA", "KUBE_PROXY_URL", "KUBE_TLS_SERVER_NAME", "TF_REATTACH_PROVIDERS",
		"TF_CLI_ARGS", "TF_CLI_ARGS_apply", "TF_CLI_ARGS_plan", "TF_CLI_ARGS_destroy", "TF_WORKSPACE",
	} {
		t.Setenv(key, "")
	}
	const minimal = `
					pod_selector {}
					policy_types = ["Ingress"]
				`
	const selectors = `
					pod_selector { %s }
					policy_types = ["Ingress", "Egress"]
					ingress {
					  from {
					    namespace_selector { %s }
					    pod_selector { %s }
					  }
					}
					egress {
					  to {
					    namespace_selector { %s }
					    pod_selector { %s }
					  }
					}
				`
	const expressions = `
					pod_selector {
					  match_expressions {
					    key = "exists"
					    operator = "Exists"
					    %s
					  }
					  match_expressions {
					    key = "absent"
					    operator = "DoesNotExist"
					    %s
					  }
					}
					policy_types = ["Ingress"]
				`
	const except = `
					pod_selector {}
					policy_types = ["Ingress", "Egress"]
					ingress {
					  from {
					    ip_block {
					      cidr = "10.0.0.0/8"
					      %s
					    }
					  }
					}
					egress {
					  to {
					    ip_block {
					      cidr = "192.168.0.0/16"
					      %s
					    }
					  }
					}
				`
	for _, tc := range []struct {
		name, metadata, spec      string
		normalizeMetadata         bool
		normalizeSelectorMaps     bool
		normalizeExpressionValues bool
		refreshReleased           bool
	}{
		{name: "minimal", spec: minimal},
		{name: "metadata_empty", metadata: "labels = {}\nannotations = {}", spec: minimal, normalizeMetadata: true},
		{
			name: "full", metadata: `labels = { managed = "yes" }
							annotations = { "example.com/managed" = "yes" }`,
			spec: `
							pod_selector {
							  match_labels = { app = "web" }
							  match_expressions {
							    key = "tier"
							    operator = "In"
							    values = ["api", "web"]
							  }
							}
							policy_types = ["Ingress", "Egress"]
							ingress {
							  ports {
							    port = "80"
							    end_port = 90
							  }
							  ports {
							    port = "dns"
							    protocol = "UDP"
							  }
							  from {
							    namespace_selector { match_labels = { team = "dev" } }
							    pod_selector {
							      match_expressions {
							        key = "blocked"
							        operator = "NotIn"
							        values = ["yes"]
							      }
							    }
							  }
							  from {
							    ip_block {
							      cidr = "10.0.0.0/8"
							      except = ["10.1.0.0/16", "10.2.0.0/16"]
							    }
							  }
							}
							egress {
							  ports { port = "443" }
							  to {
							    ip_block {
							      cidr = "192.168.0.0/16"
							      except = ["192.168.1.0/24"]
							    }
							  }
							}
						`,
		},
		{name: "selector_maps_omitted", spec: fmt.Sprintf(selectors, "", "", "", "", "")},
		{name: "selector_maps_omitted_refreshed", spec: fmt.Sprintf(selectors, "", "", "", "", ""), refreshReleased: true},
		{name: "selector_maps_empty", spec: fmt.Sprintf(selectors, "match_labels = {}", "match_labels = {}", "match_labels = {}", "match_labels = {}", "match_labels = {}"), normalizeSelectorMaps: true},
		{name: "expression_values_omitted", spec: fmt.Sprintf(expressions, "", "")},
		{name: "expression_values_omitted_refreshed", spec: fmt.Sprintf(expressions, "", ""), refreshReleased: true},
		{name: "expression_values_empty", spec: fmt.Sprintf(expressions, "values = []", "values = []"), normalizeExpressionValues: true},
		{name: "except_omitted", spec: fmt.Sprintf(except, "", "")},
		{name: "except_omitted_refreshed", spec: fmt.Sprintf(except, "", ""), refreshReleased: true},
		{name: "except_empty", spec: fmt.Sprintf(except, "except = []", "except = []")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var object *networking.NetworkPolicy
			creates, writes, deletes := 0, 0, 0
			name := "offline-policy-" + strings.ReplaceAll(tc.name, "_", "-")
			collectionPath := "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if req.URL.Path != collectionPath && req.URL.Path != collectionPath+"/"+name {
					t.Errorf("unexpected fake API request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				switch req.Method {
				case http.MethodPost:
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					decoded, _, err := kubescheme.Codecs.UniversalDeserializer().Decode(body, nil, nil)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					var ok bool
					object, ok = decoded.(*networking.NetworkPolicy)
					if !ok {
						t.Errorf("unexpected object type %T", decoded)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					creates++
					object.APIVersion, object.Kind = "networking.k8s.io/v1", "NetworkPolicy"
					object.UID, object.ResourceVersion, object.Generation = "offline-policy-uid", "1", 1
					for i := range object.Spec.Ingress {
						for j := range object.Spec.Ingress[i].Ports {
							if object.Spec.Ingress[i].Ports[j].Protocol == nil {
								object.Spec.Ingress[i].Ports[j].Protocol = ptr.To(corev1.ProtocolTCP)
							}
						}
					}
					for i := range object.Spec.Egress {
						for j := range object.Spec.Egress[i].Ports {
							if object.Spec.Egress[i].Ports[j].Protocol == nil {
								object.Spec.Egress[i].Ports[j].Protocol = ptr.To(corev1.ProtocolTCP)
							}
						}
					}
				case http.MethodGet:
					if object == nil {
						w.WriteHeader(http.StatusNotFound)
						fmt.Fprint(w, `{"apiVersion":"v1","kind":"Status","reason":"NotFound","code":404}`)
						return
					}
				case http.MethodDelete:
					deletes++
					object = nil
					fmt.Fprint(w, `{"apiVersion":"v1","kind":"Status","status":"Success"}`)
					return
				default:
					writes++
					t.Errorf("unchanged upgrade unexpectedly wrote the object: %s", req.Method)
					w.WriteHeader(http.StatusConflict)
					return
				}
				if err := json.NewEncoder(w).Encode(object); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			t.Setenv("KUBE_HOST", server.URL)
			t.Setenv("KUBE_INSECURE", "false")
			client, err := kubeclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			meta := func() any { return networkPolicyTestClients{client: client} }
			config := fmt.Sprintf(`
							resource "kubernetes_network_policy_v1" "test" {
							  metadata {
							    name = %q
							    %s
							  }
							  spec { %s }
							}
						`, name, tc.metadata, tc.spec)
			targetConfig := NetworkingTargetTestConfig(config)
			var before *networking.NetworkPolicy
			var beforeState map[string]string
			releasedChecks := 0
			assertUnchanged := func(state *terraform.State) error {
				mu.Lock()
				defer mu.Unlock()
				if !reflect.DeepEqual(object, before) || creates != 1 || writes != 0 || deletes != 0 {
					return fmt.Errorf("unchanged upgrade altered the object, UID, or payload")
				}
				after := state.RootModule().Resources["kubernetes_network_policy_v1.test"]
				if after == nil || after.Primary == nil {
					return fmt.Errorf("upgraded state has no NetworkPolicy")
				}
				expected := make(map[string]string, len(beforeState)+2)
				for key, value := range beforeState {
					expected[key] = value
				}
				if tc.normalizeMetadata {
					expected["metadata.0.labels.%"] = "0"
					expected["metadata.0.annotations.%"] = "0"
				}
				if tc.normalizeSelectorMaps {
					for _, prefix := range []string{
						"spec.0.pod_selector.0",
						"spec.0.ingress.0.from.0.namespace_selector.0", "spec.0.ingress.0.from.0.pod_selector.0",
						"spec.0.egress.0.to.0.namespace_selector.0", "spec.0.egress.0.to.0.pod_selector.0",
					} {
						expected[prefix+".match_labels.%"] = "0"
					}
				}
				if tc.normalizeExpressionValues {
					expected["spec.0.pod_selector.0.match_expressions.0.values.#"] = "0"
					expected["spec.0.pod_selector.0.match_expressions.1.values.#"] = "0"
				}
				expected = networkingTargetFlatAttributes(expected)
				if !reflect.DeepEqual(after.Primary.Attributes, expected) {
					return fmt.Errorf("upgrade changed stored attributes:\nexpected %#v\nafter %#v", expected, after.Primary.Attributes)
				}
				return nil
			}
			local := map[string]func() (tfprotov6.ProviderServer, error){
				"kubernetes": providerserver.NewProtocol6WithError(networkPolicyTestProvider{meta: meta}),
			}
			testCase := tfresource.TestCase{
				IsUnitTest: true,
				CheckDestroy: func(*terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if object != nil || creates != 1 || deletes != 1 || writes != 0 {
						return fmt.Errorf("fake NetworkPolicy was not destroyed exactly once without update writes")
					}
					return nil
				},
				Steps: []tfresource.TestStep{
					{
						ExternalProviders: map[string]tfresource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "3.2.1"},
						},
						Config: config,
						Check: func(state *terraform.State) error {
							mu.Lock()
							defer mu.Unlock()
							if object == nil || creates != 1 {
								return fmt.Errorf("released provider did not create exactly one object")
							}
							before = object.DeepCopy()
							beforeState = state.RootModule().Resources["kubernetes_network_policy_v1.test"].Primary.Attributes
							releasedChecks++
							collections := map[string]string{}
							for key, value := range beforeState {
								if strings.HasSuffix(key, ".match_labels.%") || strings.HasSuffix(key, ".values.#") || strings.HasSuffix(key, ".except.#") {
									collections[key] = value
								}
							}
							t.Logf("released baseline #%d primitive counts: %v", releasedChecks, collections)
							return nil
						},
					},
					{
						ProtoV6ProviderFactories: local,
						Config:                   targetConfig,
						ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							networkPolicyOfflinePlanCheck{
								normalizeMetadata: tc.normalizeMetadata, normalizeSelectorMaps: tc.normalizeSelectorMaps,
								normalizeExpressionValues: tc.normalizeExpressionValues,
							},
						}},
						Check: assertUnchanged,
					},
					{
						ProtoV6ProviderFactories: local,
						Config:                   targetConfig,
						ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						}},
						Check: assertUnchanged,
					},
				},
			}
			if tc.refreshReleased {
				refreshed := testCase.Steps[0]
				refreshed.ConfigPlanChecks = tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}
				testCase.Steps = append([]tfresource.TestStep{testCase.Steps[0], refreshed}, testCase.Steps[1:]...)
			}
			tfresource.Test(t, testCase)
		})
	}
}

func TestIngressClassOperationErrors(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		for _, operation := range []string{"create", "read", "update", "delete"} {
			name := operation
			if cancelled {
				name += "_cancelled"
			}
			t.Run(name, func(t *testing.T) {
				requests := 0
				r := ingressClassTestResource(t, func(w http.ResponseWriter, req *http.Request) {
					requests++
					http.Error(w, "forbidden", http.StatusForbidden)
				})
				state := ingressClassTestStateValue(t, r)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if cancelled {
					cancel()
				}
				switch operation {
				case "create":
					nullState := tfsdk.State{Schema: state.Schema, Raw: tftypes.NewValue(state.Raw.Type(), nil)}
					response := resource.CreateResponse{State: nullState}
					r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(state)}, &response)
					if !response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
						t.Fatalf("failed Create must return diagnostics without inventing state: %v", response.Diagnostics)
					}
				case "read":
					response := resource.ReadResponse{State: state}
					r.Read(ctx, resource.ReadRequest{State: state}, &response)
					if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
						t.Fatalf("failed Read must preserve state: %v", response.Diagnostics)
					}
				case "update":
					response := resource.UpdateResponse{State: state}
					r.Update(ctx, resource.UpdateRequest{State: state, Plan: tfsdk.Plan(state)}, &response)
					if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
						t.Fatalf("failed Update must preserve state: %v", response.Diagnostics)
					}
				case "delete":
					response := resource.DeleteResponse{State: state}
					r.Delete(ctx, resource.DeleteRequest{State: state}, &response)
					if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
						t.Fatalf("failed Delete must preserve state: %v", response.Diagnostics)
					}
				}
				if cancelled && requests != 0 {
					t.Fatalf("cancelled context made %d API requests", requests)
				}
			})
		}
	}
}

func TestNetworkingCollectionPlanRemoval(t *testing.T) {
	ctx := context.Background()
	modifier := networkingEmptyCollectionPlanModifier{}
	for _, collection := range []struct {
		name                           string
		null, empty, nonempty, unknown attr.Value
		modify                         func(attr.Value, attr.Value, attr.Value) attr.Value
	}{
		{
			"map", types.MapNull(types.StringType), types.MapValueMust(types.StringType, map[string]attr.Value{}),
			types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}), types.MapUnknown(types.StringType),
			func(config, state, plan attr.Value) attr.Value {
				response := planmodifier.MapResponse{PlanValue: plan.(types.Map)}
				modifier.PlanModifyMap(ctx, planmodifier.MapRequest{
					ConfigValue: config.(types.Map), StateValue: state.(types.Map), PlanValue: plan.(types.Map),
				}, &response)
				return response.PlanValue
			},
		},
		{
			"list", types.ListNull(types.StringType), types.ListValueMust(types.StringType, []attr.Value{}),
			types.ListValueMust(types.StringType, []attr.Value{types.StringValue("value")}), types.ListUnknown(types.StringType),
			func(config, state, plan attr.Value) attr.Value {
				response := planmodifier.ListResponse{PlanValue: plan.(types.List)}
				modifier.PlanModifyList(ctx, planmodifier.ListRequest{
					ConfigValue: config.(types.List), StateValue: state.(types.List), PlanValue: plan.(types.List),
				}, &response)
				return response.PlanValue
			},
		},
		{
			"set", types.SetNull(types.StringType), types.SetValueMust(types.StringType, []attr.Value{}),
			types.SetValueMust(types.StringType, []attr.Value{types.StringValue("value")}), types.SetUnknown(types.StringType),
			func(config, state, plan attr.Value) attr.Value {
				response := planmodifier.SetResponse{PlanValue: plan.(types.Set)}
				modifier.PlanModifySet(ctx, planmodifier.SetRequest{
					ConfigValue: config.(types.Set), StateValue: state.(types.Set), PlanValue: plan.(types.Set),
				}, &response)
				return response.PlanValue
			},
		},
	} {
		t.Run(collection.name, func(t *testing.T) {
			for _, tc := range []struct {
				name                          string
				config, state, plan, expected attr.Value
			}{
				{"fresh omission", collection.null, collection.null, collection.unknown, collection.null},
				{"historical empty omission", collection.null, collection.empty, collection.unknown, collection.empty},
				{"nonempty removal", collection.null, collection.nonempty, collection.unknown, collection.null},
				{"explicit empty", collection.empty, collection.null, collection.empty, collection.empty},
				{"configured value", collection.nonempty, collection.empty, collection.nonempty, collection.nonempty},
				{"unknown configuration", collection.unknown, collection.empty, collection.unknown, collection.unknown},
			} {
				t.Run(tc.name, func(t *testing.T) {
					if actual := collection.modify(tc.config, tc.state, tc.plan); !actual.Equal(tc.expected) {
						t.Fatalf("planned value = %s, want %s", actual, tc.expected)
					}
				})
			}
		})
	}
}

func networkingUpgradeFixture(t *testing.T, kind string) *tfprotov6.RawState {
	t.Helper()
	metadata := map[string]any{"name": "example", "uid": "kept-uid", "resource_version": "10", "generation": 1, "generate_name": "", "labels": nil, "annotations": nil}
	id := "example"
	if kind != "ingress_class" {
		metadata["namespace"] = "default"
		id = "default/example"
	}
	var spec map[string]any
	out := map[string]any{"id": id, "metadata": []any{metadata}}
	switch kind {
	case "ingress_class":
		spec = map[string]any{"controller": "example.com/controller", "parameters": []any{map[string]any{"name": "params", "kind": "Params", "api_group": "example.com", "namespace": "", "scope": "Cluster"}}}
	case "ingress":
		backend := func() []any {
			return []any{map[string]any{"resource": []any{}, "service": []any{map[string]any{"name": "app", "port": []any{map[string]any{"name": "", "number": 80}}}}}}
		}
		spec = map[string]any{"ingress_class_name": "example", "default_backend": backend(), "rule": []any{map[string]any{"host": "example.com", "http": []any{map[string]any{"path": []any{map[string]any{"path": "/", "path_type": "Prefix", "backend": backend()}}}}}}, "tls": []any{}}
		out["wait_for_load_balancer"] = false
		out["timeouts"] = map[string]any{"create": "17m", "delete": "19m"}
		out["status"] = []any{map[string]any{"load_balancer": []any{map[string]any{"ingress": []any{}}}}}
	case "network_policy":
		selector := func() []any { return []any{map[string]any{"match_labels": nil, "match_expressions": []any{}}} }
		peer := func() map[string]any {
			return map[string]any{"ip_block": []any{}, "namespace_selector": selector(), "pod_selector": []any{}}
		}
		spec = map[string]any{"pod_selector": selector(), "policy_types": []any{"Ingress", "Egress"}, "ingress": []any{map[string]any{"ports": []any{}, "from": []any{peer()}}}, "egress": []any{map[string]any{"ports": []any{}, "to": []any{peer()}}}}
	default:
		t.Fatal(kind)
	}
	out["spec"] = []any{spec}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	return &tfprotov6.RawState{JSON: data}
}

func networkingUpgradeProvider(kind string, meta func() any) provider.Provider {
	switch kind {
	case "ingress":
		return ingressInternalProvider{meta: meta}
	case "ingress_class":
		return ingressClassTestProvider{meta: meta}
	default:
		return networkPolicyTestProvider{meta: meta}
	}
}

func networkingUpgradeResource(kind string) resource.Resource {
	switch kind {
	case "ingress":
		return NewIngressV1()
	case "ingress_class":
		return NewIngressClassV1()
	default:
		return NewNetworkPolicyV1()
	}
}

// UpgradeResourceState and MoveResourceState must independently reach the final
// schema. Deliberately skip Read before planning so refresh cannot hide a bad conversion.
func TestNetworkingSingletonUpgradeAndMoveWithoutRead(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"ingress", "ingress_class", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			server := providerserver.NewProtocol6(networkingUpgradeProvider(kind, nil))()
			typeName := "kubernetes_" + kind + "_v1"
			schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, schemas.Diagnostics)
			schema := schemas.ResourceSchemas[typeName]
			if schema.Version != 1 {
				t.Fatalf("schema version=%d", schema.Version)
			}
			typ := schema.ValueType()
			raw := networkingUpgradeFixture(t, kind)
			upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{TypeName: typeName, Version: 0, RawState: raw})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, upgraded.Diagnostics)
			expected, err := upgraded.UpgradedState.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			routes := map[string]*tfprotov6.DynamicValue{"same_type": upgraded.UpgradedState}
			if kind != "ingress" {
				moved, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceTypeName: "kubernetes_" + kind, SourceSchemaVersion: 0, SourceState: raw, TargetTypeName: typeName})
				if err != nil {
					t.Fatal(err)
				}
				ingressClassCheckDiagnostics(t, moved.Diagnostics)
				routes["alias_move"] = moved.TargetState
			}
			for route, prior := range routes {
				t.Run(route, func(t *testing.T) {
					value, err := prior.Unmarshal(typ)
					if err != nil {
						t.Fatal(err)
					}
					if !value.Equal(expected) {
						t.Fatal("same-type and alias conversion differ")
					}
					config, err := tftypes.Transform(value, func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
						steps := at.Steps()
						if len(steps) == 0 {
							return v, nil
						}
						last := steps[len(steps)-1]
						if last == tftypes.AttributeName("id") || last == tftypes.AttributeName("status") || last == tftypes.AttributeName("generate_name") || last == tftypes.AttributeName("generation") || last == tftypes.AttributeName("resource_version") || last == tftypes.AttributeName("uid") {
							return tftypes.NewValue(v.Type(), nil), nil
						}
						return v, nil
					})
					if err != nil {
						t.Fatal(err)
					}
					plan, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: typeName, Config: ingressInternalDynamic(t, config), PriorState: prior, ProposedNewState: prior})
					if err != nil {
						t.Fatal(err)
					}
					ingressClassCheckDiagnostics(t, plan.Diagnostics)
					if len(plan.RequiresReplace) != 0 {
						t.Fatalf("unchanged migration replaces %v", plan.RequiresReplace)
					}
					planned, err := plan.PlannedState.Unmarshal(typ)
					if err != nil {
						t.Fatal(err)
					}
					if !planned.Equal(value) {
						t.Fatalf("unchanged migration changed state:\n%s\n%s", value, planned)
					}
					changed, err := tftypes.Transform(config, func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
						steps := at.Steps()
						if len(steps) == 3 && steps[0] == tftypes.AttributeName("metadata") && steps[2] == tftypes.AttributeName("name") {
							return tftypes.NewValue(tftypes.String, "changed"), nil
						}
						return v, nil
					})
					if err != nil {
						t.Fatal(err)
					}
					// A real name edit must retain its replacement contract.
					renamed, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: typeName, Config: ingressInternalDynamic(t, changed), PriorState: prior, ProposedNewState: ingressInternalDynamic(t, changed)})
					if err != nil {
						t.Fatal(err)
					}
					ingressClassCheckDiagnostics(t, renamed.Diagnostics)
					if len(renamed.RequiresReplace) == 0 {
						t.Fatal("name edit lost required replacement")
					}
				})
			}
		})
	}
}

func TestNetworkingSingletonConversionContract(t *testing.T) {
	for _, kind := range []string{"ingress", "ingress_class", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			var object map[string]any
			if err := json.Unmarshal(networkingUpgradeFixture(t, kind).JSON, &object); err != nil {
				t.Fatal(err)
			}
			beforeID := object["id"]
			beforeTimeouts, _ := json.Marshal(object["timeouts"])
			if err := upgradeNetworkingSingletons(object, kind); err != nil {
				t.Fatal(err)
			}
			first, _ := json.Marshal(object)
			if err := upgradeNetworkingSingletons(object, kind); err != nil {
				t.Fatal(err)
			}
			second, _ := json.Marshal(object)
			if string(first) != string(second) {
				t.Fatal("converter is not idempotent")
			}
			if object["id"] != beforeID {
				t.Fatal("ID changed")
			}
			afterTimeouts, _ := json.Marshal(object["timeouts"])
			if string(afterTimeouts) != string(beforeTimeouts) {
				t.Fatal("configured timeouts changed")
			}
			spec := object["spec"].([]any)[0].(map[string]any)
			switch kind {
			case "ingress_class":
				if _, ok := spec["parameters"].(map[string]any); !ok {
					t.Fatal("parameters is not an object")
				}
			case "ingress":
				backend := spec["default_backend"].(map[string]any)
				service := backend["service"].(map[string]any)
				if _, ok := service["port"].(map[string]any); !ok {
					t.Fatal("port is not an object")
				}
				if backend["resource"] != nil {
					t.Fatal("omitted resource must be null")
				}
			case "network_policy":
				peer := spec["ingress"].([]any)[0].(map[string]any)["from"].([]any)[0].(map[string]any)
				if peer["pod_selector"] != nil {
					t.Fatal("omitted selector must be null")
				}
				if _, ok := peer["namespace_selector"].(map[string]any); !ok {
					t.Fatal("present empty namespace selector must remain an object")
				}
			}
		})
	}
	for _, tc := range []struct {
		name                string
		value               any
		required, wantError bool
	}{{"absent", nil, false, false}, {"empty-list", []any{}, false, false}, {"empty-object", map[string]any{}, false, false}, {"one-empty", []any{map[string]any{}}, false, false}, {"multiple", []any{map[string]any{}, map[string]any{}}, false, true}, {"scalar", true, false, true}, {"null-element", []any{nil}, true, true}, {"optional-null-element", []any{nil}, false, true}, {"missing-required", nil, true, true}, {"empty-required", []any{}, true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			parent := map[string]any{"selector": tc.value}
			_, err := networkingSingleton(parent, "selector", "spec[0]", tc.required)
			if (err != nil) != tc.wantError {
				t.Fatalf("conversion error=%v", err)
			}
			if err != nil && !strings.Contains(err.Error(), "spec[0].selector") {
				t.Fatalf("missing path: %v", err)
			}
		})
	}
}

func TestNetworkingLegacyIdentityMissingReadProtocol(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"ingress", "ingress_class", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet {
					t.Errorf("unexpected mutation: %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"apiVersion":"v1","kind":"Status","reason":"NotFound","code":404}`)
			}))
			defer api.Close()
			client, err := kubeclient.NewForConfig(&rest.Config{Host: api.URL})
			if err != nil {
				t.Fatal(err)
			}
			server := providerserver.NewProtocol6(networkingUpgradeProvider(kind, func() any { return ingressClassTestClientsets{client: client} }))()
			typeName := "kubernetes_" + kind + "_v1"
			schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, schemas.Diagnostics)
			conf := tftypes.NewValue(schemas.Provider.ValueType(), map[string]tftypes.Value{})
			configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: ingressInternalDynamic(t, conf)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, configured.Diagnostics)
			upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{TypeName: typeName, Version: 0, RawState: networkingUpgradeFixture(t, kind)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, upgraded.Diagnostics)
			read, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: typeName, CurrentState: upgraded.UpgradedState})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, read.Diagnostics)
			state, err := read.NewState.Unmarshal(schemas.ResourceSchemas[typeName].ValueType())
			if err != nil || !state.IsNull() {
				t.Fatalf("state not removed: %s %v", state, err)
			}
			if calls != 1 {
				t.Fatalf("GET calls=%d", calls)
			}
		})
	}
}

// The API expansion proves empty namespace selectors and absent pod selectors
// retain different meanings, and the flattener must converge with upgraded state.
func TestNetworkingSingletonReadConvergence(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"ingress", "ingress_class", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			r := networkingUpgradeResource(kind)
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			value, _, _, err := decodeNetworkingState(ctx, networkingUpgradeFixture(t, kind), schema.Schema.Type().TerraformType(ctx), kind)
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: schema.Schema, Raw: value}
			switch kind {
			case "ingress_class":
				var model IngressClassV1Model
				if d := state.Get(ctx, &model); d.HasError() {
					t.Fatal(d)
				}
				flat := flattenIngressClassSpec(expandIngressClassSpec(model.Spec[0]))
				if !reflect.DeepEqual(model.Spec, flat) {
					t.Fatalf("upgrade/read differ: %#v %#v", model.Spec, flat)
				}
			case "ingress":
				var model IngressV1Model
				if d := state.Get(ctx, &model); d.HasError() {
					t.Fatal(d)
				}
				api, d := ingressExpandSpec(ctx, model.Spec)
				if d.HasError() {
					t.Fatal(d)
				}
				flat, d := ingressFlattenSpec(ctx, api, model.Spec)
				if d.HasError() {
					t.Fatal(d)
				}
				if !reflect.DeepEqual(model.Spec, flat) {
					t.Fatalf("upgrade/read differ: %#v %#v", model.Spec, flat)
				}
			case "network_policy":
				var model NetworkPolicyV1Model
				if d := state.Get(ctx, &model); d.HasError() {
					t.Fatal(d)
				}
				api, d := networkPolicyExpandSpec(ctx, model.Spec)
				if d.HasError() {
					t.Fatal(d)
				}
				peer := api.Ingress[0].From[0]
				if peer.NamespaceSelector == nil || peer.PodSelector != nil || !reflect.DeepEqual(*peer.NamespaceSelector, metav1.LabelSelector{}) {
					t.Fatalf("selector meaning changed: %#v", peer)
				}
				flat, d := networkPolicyFlattenSpec(ctx, api, model.Spec)
				if d.HasError() {
					t.Fatal(d)
				}
				if !reflect.DeepEqual(model.Spec, flat) {
					t.Fatalf("upgrade/read differ: %#v %#v", model.Spec, flat)
				}
			}
		})
	}
}

func networkingUpgradeConfig(t *testing.T, state tftypes.Value) tftypes.Value {
	t.Helper()
	config, err := tftypes.Transform(state, func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		steps := at.Steps()
		if len(steps) == 0 {
			return v, nil
		}
		switch steps[len(steps)-1] {
		case tftypes.AttributeName("id"), tftypes.AttributeName("status"), tftypes.AttributeName("generate_name"), tftypes.AttributeName("generation"), tftypes.AttributeName("resource_version"), tftypes.AttributeName("uid"):
			return tftypes.NewValue(v.Type(), nil), nil
		}
		return v, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestNetworkingSingletonUnknownPlan(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ kind, field string }{{"ingress", "default_backend"}, {"ingress", "http"}, {"ingress_class", "parameters"}, {"network_policy", "pod_selector"}, {"network_policy", "namespace_selector"}} {
		t.Run(tc.kind+"/"+tc.field, func(t *testing.T) {
			var s resource.SchemaResponse
			networkingUpgradeResource(tc.kind).Schema(ctx, resource.SchemaRequest{}, &s)
			state, _, _, err := decodeNetworkingState(ctx, networkingUpgradeFixture(t, tc.kind), s.Schema.Type().TerraformType(ctx), tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			config, err := tftypes.Transform(networkingUpgradeConfig(t, state), func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
				steps := at.Steps()
				if len(steps) > 0 && steps[len(steps)-1] == tftypes.AttributeName(tc.field) {
					return tftypes.NewValue(v.Type(), tftypes.UnknownValue), nil
				}
				return v, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			server := providerserver.NewProtocol6(networkingUpgradeProvider(tc.kind, nil))()
			typeName := "kubernetes_" + tc.kind + "_v1"
			validated, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: typeName, Config: ingressInternalDynamic(t, config)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, validated.Diagnostics)
			plan, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: typeName, Config: ingressInternalDynamic(t, config), PriorState: ingressInternalDynamic(t, tftypes.NewValue(state.Type(), nil)), ProposedNewState: ingressInternalDynamic(t, config)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, plan.Diagnostics)
		})
	}
}

func TestNetworkingNamespacedNameAndGenerateName(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"ingress", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			var s resource.SchemaResponse
			networkingUpgradeResource(kind).Schema(ctx, resource.SchemaRequest{}, &s)
			state, _, _, err := decodeNetworkingState(ctx, networkingUpgradeFixture(t, kind), s.Schema.Type().TerraformType(ctx), kind)
			if err != nil {
				t.Fatal(err)
			}
			config, err := tftypes.Transform(networkingUpgradeConfig(t, state), func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
				steps := at.Steps()
				if len(steps) == 3 && steps[0] == tftypes.AttributeName("metadata") && steps[2] == tftypes.AttributeName("generate_name") {
					return tftypes.NewValue(tftypes.String, "ignored-prefix-"), nil
				}
				return v, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			server := providerserver.NewProtocol6(networkingUpgradeProvider(kind, nil))()
			result, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: "kubernetes_" + kind + "_v1", Config: ingressInternalDynamic(t, config)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, result.Diagnostics)
			for _, d := range result.Diagnostics {
				if d.Severity == tfprotov6.DiagnosticSeverityWarning && strings.Contains(d.Summary, "generate_name is ignored") {
					return
				}
			}
			t.Fatal("missing ignored generate_name warning")
		})
	}
}

// A mutating admission policy must not change known planned values.
func TestNetworkPolicyWritePreservesPlan(t *testing.T) {
	ctx := context.Background()
	var object *networking.NetworkPolicy
	r := networkPolicyTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodPost {
			if err := json.NewDecoder(req.Body).Decode(&object); err != nil {
				t.Fatal(err)
			}
			object.APIVersion, object.Kind = networkPolicyAPIVersion, networkPolicyKind
			object.UID, object.ResourceVersion, object.Generation = "uid-1", "10", 1
			object.Spec.PodSelector.MatchLabels = map[string]string{"admission.example/tenant": "one"}
		}
		if err := json.NewEncoder(w).Encode(object); err != nil {
			t.Error(err)
		}
	})
	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)
	raw, _, _, err := decodeNetworkingState(ctx, &tfprotov6.RawState{JSON: []byte(networkPolicyTestStoredState)}, schema.Schema.Type().TerraformType(ctx), "network_policy")
	if err != nil {
		t.Fatal(err)
	}
	initial := tfsdk.State{Schema: schema.Schema, Raw: raw}
	response := resource.CreateResponse{State: tfsdk.State{Schema: initial.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(initial)}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var plan, actual NetworkPolicyV1Model
	if d := initial.Get(ctx, &plan); d.HasError() {
		t.Fatal(d)
	}
	if d := response.State.Get(ctx, &actual); d.HasError() {
		t.Fatal(d)
	}
	if !plan.Spec[0].PodSelector.MatchLabels.Equal(actual.Spec[0].PodSelector.MatchLabels) {
		t.Fatalf("known planned match_labels changed during Create: plan=%s actual=%s", plan.Spec[0].PodSelector.MatchLabels, actual.Spec[0].PodSelector.MatchLabels)
	}
	// An update response can also contain admission changes. Keep the new plan,
	// then expose the API value as drift on Read.
	prior := response.State
	plan.Spec[0].PodSelector.MatchLabels = types.MapValueMust(types.StringType, map[string]attr.Value{"configured": types.StringValue("new")})
	proposed := tfsdk.State{Schema: initial.Schema}
	if d := proposed.Set(ctx, &plan); d.HasError() {
		t.Fatal(d)
	}
	update := resource.UpdateResponse{State: prior}
	r.Update(ctx, resource.UpdateRequest{State: prior, Plan: tfsdk.Plan(proposed)}, &update)
	if update.Diagnostics.HasError() {
		t.Fatal(update.Diagnostics)
	}
	if d := update.State.Get(ctx, &actual); d.HasError() {
		t.Fatal(d)
	}
	if !actual.Spec[0].PodSelector.MatchLabels.Equal(plan.Spec[0].PodSelector.MatchLabels) {
		t.Fatal("Update changed the planned selector")
	}
	read := resource.ReadResponse{State: update.State}
	r.Read(ctx, resource.ReadRequest{State: update.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if d := read.State.Get(ctx, &actual); d.HasError() {
		t.Fatal(d)
	}
	if actual.Spec[0].PodSelector.MatchLabels.Equal(plan.Spec[0].PodSelector.MatchLabels) {
		t.Fatal("Read hid admission drift")
	}

}
