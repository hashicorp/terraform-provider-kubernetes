// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	framework "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	coreapi "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

var serviceAccountTypes = []struct {
	name, alias string
	adopt       bool
}{
	{"kubernetes_service_account_v1", "kubernetes_service_account", false},
	{"kubernetes_default_service_account_v1", "kubernetes_default_service_account", true},
}

// Exercise the actual production mux, including the SDK resources used by live fixtures.
var serviceAccountMuxFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "service-account-test")
	},
}

func serviceAccountServer(t *testing.T) (tfprotov6.ProviderServer, *tfprotov6.GetProviderSchemaResponse) {
	t.Helper()
	server, err := serviceAccountMuxFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schema, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil || serviceAccountDiagnosticError(schema.Diagnostics) {
		t.Fatalf("GetProviderSchema: %v, %v", err, schema.Diagnostics)
	}
	return server, schema
}

func serviceAccountDiagnosticError(diags []*tfprotov6.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			return true
		}
	}
	return false
}

func serviceAccountResource(t *testing.T, name string) fwresource.Resource {
	t.Helper()
	for _, constructor := range framework.New("test", nil).Resources(context.Background()) {
		r := constructor()
		var metadata fwresource.MetadataResponse
		r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "kubernetes"}, &metadata)
		if metadata.TypeName == name {
			return r
		}
	}
	t.Fatalf("%s is not registered in the production Framework provider", name)
	return nil
}

func TestServiceAccountMuxSchema(t *testing.T) {
	server, schemas := serviceAccountServer(t)
	identities, err := server.GetResourceIdentitySchemas(context.Background(), &tfprotov6.GetResourceIdentitySchemasRequest{})
	if err != nil || serviceAccountDiagnosticError(identities.Diagnostics) {
		t.Fatalf("identity schemas: %v, %v", err, identities.Diagnostics)
	}
	for _, kind := range serviceAccountTypes {
		t.Run(kind.name, func(t *testing.T) {
			serviceAccountResource(t, kind.name)
			sdk := kubernetes.Provider()
			if sdk.ResourcesMap[kind.name] != nil || sdk.ResourcesMap[kind.alias] == nil {
				t.Fatal("versioned type must move off SDKv2 while its deprecated alias remains")
			}
			current, legacy := schemas.ResourceSchemas[kind.name], schemas.ResourceSchemas[kind.alias]
			if current == nil || legacy == nil {
				t.Fatal("production mux lost a resource type")
			}
			if current.Version != 0 || !current.ValueType().Equal(legacy.ValueType()) {
				t.Fatalf("schema0 and complete SDK wire type must survive:\nnew=%s\nold=%s", current.ValueType(), legacy.ValueType())
			}
			for name, nesting := range map[string]tfprotov6.SchemaNestedBlockNestingMode{
				"metadata": tfprotov6.SchemaNestedBlockNestingModeList,
				"secret":   tfprotov6.SchemaNestedBlockNestingModeSet, "image_pull_secret": tfprotov6.SchemaNestedBlockNestingModeSet,
			} {
				found := false
				for _, block := range current.Block.BlockTypes {
					if block.TypeName == name {
						found = block.Nesting == nesting
					}
				}
				if !found {
					t.Errorf("%s must remain a %v nested block", name, nesting)
				}
			}
			deprecated := false
			for _, attr := range current.Block.Attributes {
				if attr.Name == "default_secret_name" {
					deprecated = attr.Deprecated && attr.Computed
				}
			}
			if !deprecated {
				t.Error("default_secret_name must remain computed and deprecated")
			}
			identity := identities.IdentitySchemas[kind.name]
			if identity == nil || identity.Version != 1 ||
				!identity.ValueType().Equal(identities.IdentitySchemas[kind.alias].ValueType()) {
				t.Fatal("namespaced identity1 must match the retained SDK alias")
			}
		})
	}
}

func TestServiceAccountConfigure(t *testing.T) {
	for _, kind := range serviceAccountTypes {
		for _, tc := range []struct {
			name string
			data any
			bad  bool
		}{
			{"unconfigured", nil, false},
			{"wrong-provider-data", "not a callback", true},
			{"lazy-sdk-callback", func() any { panic("Configure must not eagerly resolve SDK meta") }, false},
		} {
			t.Run(kind.name+"/"+tc.name, func(t *testing.T) {
				r := serviceAccountResource(t, kind.name).(fwresource.ResourceWithConfigure)
				var response fwresource.ConfigureResponse
				r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: tc.data}, &response)
				if response.Diagnostics.HasError() != tc.bad {
					t.Fatalf("Configure diagnostics: %v; want error=%t", response.Diagnostics, tc.bad)
				}
			})
		}
	}
}

func serviceAccountValue(typ tftypes.Type, overrides map[string]tftypes.Value) tftypes.Value {
	values := map[string]tftypes.Value{}
	for name, fieldType := range typ.(tftypes.Object).AttributeTypes {
		values[name] = tftypes.NewValue(fieldType, nil)
	}
	for name, value := range overrides {
		values[name] = value
	}
	return tftypes.NewValue(typ, values)
}

func TestServiceAccountImportEmptyMetadata(t *testing.T) {
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "KUBE") {
			t.Setenv(name, "")
		}
	}
	for _, kind := range serviceAccountTypes {
		t.Run(kind.name, func(t *testing.T) {
			name := "account"
			if kind.adopt {
				name = "default"
			}
			api := newServiceAccountAPI(t, kind.adopt)
			api.object = &coreapi.ServiceAccount{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
				ObjectMeta: metav1.ObjectMeta{
					Name: name, Namespace: serviceAccountCLINamespace, UID: "import-uid", ResourceVersion: "1",
				},
				AutomountServiceAccountToken: ptr.To(true),
			}
			api.setPhase("import")
			server, schemas := serviceAccountServer(t)
			providerType := schemas.Provider.ValueType()
			config, err := tfprotov6.NewDynamicValue(providerType, serviceAccountValue(providerType, map[string]tftypes.Value{
				"host": tftypes.NewValue(tftypes.String, api.server.URL),
			}))
			if err != nil {
				t.Fatal(err)
			}
			configured, err := server.ConfigureProvider(context.Background(), &tfprotov6.ConfigureProviderRequest{Config: &config})
			if err != nil {
				t.Fatal(err)
			}
			if serviceAccountDiagnosticError(configured.Diagnostics) {
				t.Fatalf("configure: %v", configured.Diagnostics)
			}
			imported, err := server.ImportResourceState(context.Background(), &tfprotov6.ImportResourceStateRequest{
				TypeName: kind.name, ID: serviceAccountCLINamespace + "/" + name,
			})
			if err != nil || serviceAccountDiagnosticError(imported.Diagnostics) || len(imported.ImportedResources) != 1 {
				t.Fatalf("import: %v, %#v", err, imported)
			}
			value, err := imported.ImportedResources[0].State.Unmarshal(schemas.ResourceSchemas[kind.name].ValueType())
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]tftypes.Value
			if err := value.As(&fields); err != nil {
				t.Fatal(err)
			}
			var metadata []tftypes.Value
			if err := fields["metadata"].As(&metadata); err != nil || len(metadata) != 1 {
				t.Fatalf("metadata: %v, %v", err, metadata)
			}
			if err := metadata[0].As(&fields); err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"annotations", "labels"} {
				want := tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{})
				if !fields[field].Equal(want) {
					t.Errorf("imported %s = %s; want empty map for no-op empty/omitted configuration", field, fields[field])
				}
			}
			api.setPhase("cleanup")
			typ := schemas.ResourceSchemas[kind.name].ValueType()
			destroy, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, nil))
			if err != nil {
				t.Fatal(err)
			}
			deleted, err := server.ApplyResourceChange(context.Background(), &tfprotov6.ApplyResourceChangeRequest{
				TypeName: kind.name, PriorState: imported.ImportedResources[0].State, PlannedState: &destroy, Config: &destroy,
			})
			if err != nil || serviceAccountDiagnosticError(deleted.Diagnostics) {
				t.Fatalf("delete: %v, %#v", err, deleted)
			}
		})
	}
}

func TestServiceAccountValidateConfig(t *testing.T) {
	server, schemas := serviceAccountServer(t)
	for _, kind := range serviceAccountTypes {
		t.Run(kind.name, func(t *testing.T) {
			serviceAccountResource(t, kind.name)
			typ := schemas.ResourceSchemas[kind.name].ValueType()
			fields := typ.(tftypes.Object).AttributeTypes
			metadataType := fields["metadata"].(tftypes.List)
			valid := serviceAccountValue(metadataType.ElementType, map[string]tftypes.Value{
				"name":      tftypes.NewValue(tftypes.String, "default"),
				"namespace": tftypes.NewValue(tftypes.String, "test-owned"),
			})
			invalidMetadata := func(field string, value tftypes.Value) []tftypes.Value {
				var attrs map[string]tftypes.Value
				if err := valid.As(&attrs); err != nil {
					t.Fatal(err)
				}
				attrs = maps.Clone(attrs)
				attrs[field] = value
				return []tftypes.Value{tftypes.NewValue(metadataType.ElementType, attrs)}
			}
			for _, tc := range []struct {
				name     string
				metadata any
				bad      bool
			}{
				{"omitted", nil, true},
				{"empty", []tftypes.Value{}, true},
				{"duplicate", []tftypes.Value{valid, valid}, true},
				{"known", []tftypes.Value{valid}, false},
				{"unknown-collection", tftypes.UnknownValue, false},
				{"unknown-object", []tftypes.Value{tftypes.NewValue(metadataType.ElementType, tftypes.UnknownValue)}, false},
				{"unknown-name", invalidMetadata("name", tftypes.NewValue(tftypes.String, tftypes.UnknownValue)), false},
				{"invalid-name", invalidMetadata("name", tftypes.NewValue(tftypes.String, "INVALID NAME")), true},
				// SDKv2 leaves namespace validation to the API.
				{"namespace-validation-remains-api-owned", invalidMetadata("namespace", tftypes.NewValue(tftypes.String, "INVALID NAMESPACE")), false},
				{"invalid-label", invalidMetadata("labels", tftypes.NewValue(tftypes.Map{ElementType: tftypes.String},
					map[string]tftypes.Value{"app": tftypes.NewValue(tftypes.String, "invalid value")})), true},
				{"invalid-annotation", invalidMetadata("annotations", tftypes.NewValue(tftypes.Map{ElementType: tftypes.String},
					map[string]tftypes.Value{"invalid key": tftypes.NewValue(tftypes.String, "value")})), true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					value := serviceAccountValue(typ, map[string]tftypes.Value{
						"metadata":                        tftypes.NewValue(metadataType, tc.metadata),
						"secret":                          tftypes.NewValue(fields["secret"], tftypes.UnknownValue),
						"image_pull_secret":               tftypes.NewValue(fields["image_pull_secret"], tftypes.UnknownValue),
						"automount_service_account_token": tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
					})
					dynamic, err := tfprotov6.NewDynamicValue(typ, value)
					if err != nil {
						t.Fatal(err)
					}
					response, err := server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{
						TypeName: kind.name, Config: &dynamic,
					})
					if err != nil || serviceAccountDiagnosticError(response.Diagnostics) != tc.bad {
						t.Fatalf("ValidateResourceConfig: %v, %v; want error=%t", err, response.Diagnostics, tc.bad)
					}
				})
			}
		})
	}
}

func TestServiceAccountStateProtocol(t *testing.T) {
	server, schemas := serviceAccountServer(t)
	ctx := context.Background()
	identities, err := server.GetResourceIdentitySchemas(ctx, &tfprotov6.GetResourceIdentitySchemasRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range serviceAccountTypes {
		t.Run(kind.name, func(t *testing.T) {
			serviceAccountResource(t, kind.name)
			name := "account"
			if kind.adopt {
				name = "default"
			}
			metadata := map[string]any{
				"name": name, "namespace": serviceAccountCLINamespace, "uid": "original-uid",
				"resource_version": "100", "generation": 1,
				"annotations": map[string]string{"annotation": "value"}, "labels": map[string]string{"label": "value"},
			}
			if !kind.adopt {
				metadata["generate_name"] = nil
			}
			raw, err := json.Marshal(map[string]any{
				"id": serviceAccountCLINamespace + "/" + name, "metadata": []any{metadata},
				"automount_service_account_token": false, "default_secret_name": name + "-token-old",
				"secret":            []any{map[string]string{"name": "user-secret"}},
				"image_pull_secret": []any{map[string]string{"name": "pull-secret"}},
				"timeouts":          map[string]string{"create": "45s"},
			})
			if err != nil {
				t.Fatal(err)
			}
			rawState := &tfprotov6.RawState{JSON: raw}
			typ := schemas.ResourceSchemas[kind.name].ValueType()
			want, err := rawState.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
				TypeName: kind.name, Version: 0, RawState: rawState,
			})
			if err != nil || serviceAccountDiagnosticError(upgraded.Diagnostics) || upgraded.UpgradedState == nil {
				t.Fatalf("schema0 upgrade failed: %v, %v", err, upgraded.Diagnostics)
			}
			got, err := upgraded.UpgradedState.Unmarshal(typ)
			if err != nil || !got.Equal(want) {
				t.Fatalf("same-type upgrade lost stored fields: %v\n%s\nwant %s", err, got, want)
			}
			identityRaw := &tfprotov6.RawState{JSON: fmt.Appendf(nil,
				`{"api_version":"v1","kind":"ServiceAccount","namespace":%q,"name":%q}`, serviceAccountCLINamespace, name)}
			identityType := identities.IdentitySchemas[kind.name].ValueType()
			wantIdentity, err := identityRaw.Unmarshal(identityType)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				name, provider, sourceType string
				version, identityVersion   int64
				state, identity            *tfprotov6.RawState
				bad                        bool
			}{
				{"alias-with-identity", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 1, rawState, identityRaw, false},
				{"alias-pre-identity", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 0, rawState, nil, false},
				{"alias-null-legacy-identity", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 0, rawState, &tfprotov6.RawState{JSON: []byte("null")}, false},
				{"alias-namespaced-legacy-identity", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 0, rawState,
					&tfprotov6.RawState{JSON: fmt.Appendf(nil, `{"namespace":%q,"name":%q}`, serviceAccountCLINamespace, name)}, false},
				{"wrong-provider", "registry.terraform.io/other/kubernetes", kind.alias, 0, 1, rawState, identityRaw, true},
				{"wrong-type", "registry.terraform.io/hashicorp/kubernetes", "kubernetes_secret", 0, 1, rawState, identityRaw, true},
				{"future-schema", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 99, 1, rawState, identityRaw, true},
				{"future-identity", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 99, rawState, identityRaw, true},
				{"malformed-identity", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 1, rawState, &tfprotov6.RawState{JSON: []byte(`{`)}, true},
				{"mismatched-identity-kind", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 1, rawState,
					&tfprotov6.RawState{JSON: []byte(strings.ReplaceAll(string(identityRaw.JSON), "ServiceAccount", "Secret"))}, true},
				{"mismatched-identity-namespace", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 1, rawState,
					&tfprotov6.RawState{JSON: []byte(strings.ReplaceAll(string(identityRaw.JSON), serviceAccountCLINamespace, "different-namespace"))}, true},
				{"mismatched-identity-name", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 1, rawState,
					&tfprotov6.RawState{JSON: []byte(strings.ReplaceAll(string(identityRaw.JSON), `"name":"`+name+`"`, `"name":"different-name"`))}, true},
				{"mismatched-identity-api-version", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 1, rawState,
					&tfprotov6.RawState{JSON: []byte(strings.ReplaceAll(string(identityRaw.JSON), `"api_version":"v1"`, `"api_version":"v2"`))}, true},
				{"missing-state", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 1, nil, identityRaw, true},
				{"malformed-state", "registry.terraform.io/hashicorp/kubernetes", kind.alias, 0, 1, &tfprotov6.RawState{JSON: []byte(`{`)}, identityRaw, true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					response, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{
						SourceProviderAddress: tc.provider, SourceTypeName: tc.sourceType, TargetTypeName: kind.name,
						SourceSchemaVersion: tc.version, SourceState: tc.state,
						SourceIdentitySchemaVersion: tc.identityVersion, SourceIdentity: tc.identity,
					})
					if tc.bad {
						if err == nil && !serviceAccountDiagnosticError(response.Diagnostics) && response.TargetState != nil {
							t.Fatal("unsupported/malformed move returned successful target state")
						}
						return
					}
					if err != nil || serviceAccountDiagnosticError(response.Diagnostics) || response.TargetState == nil || response.TargetIdentity == nil {
						t.Fatalf("supported alias move failed: %v, %v", err, response.Diagnostics)
					}
					got, err := response.TargetState.Unmarshal(typ)
					if err != nil || !got.Equal(want) {
						t.Fatalf("alias move lost complete stored state: %v\n%s\nwant %s", err, got, want)
					}
					gotIdentity, err := response.TargetIdentity.IdentityData.Unmarshal(identityType)
					if err != nil || !gotIdentity.Equal(wantIdentity) {
						t.Fatalf("alias move lost identity: %v\n%s\nwant %s", err, gotIdentity, wantIdentity)
					}
				})
			}
			for _, tc := range []struct {
				name string
				raw  *tfprotov6.RawState
				bad  bool
			}{
				{"absent", nil, false},
				{"empty-envelope", &tfprotov6.RawState{}, false},
				{"stored", identityRaw, false},
				{"malformed", &tfprotov6.RawState{JSON: []byte(`{`)}, true},
			} {
				t.Run("upgrade-identity/"+tc.name, func(t *testing.T) {
					response, err := server.UpgradeResourceIdentity(ctx, &tfprotov6.UpgradeResourceIdentityRequest{
						TypeName: kind.name, Version: 0, RawIdentity: tc.raw,
					})
					if err != nil || serviceAccountDiagnosticError(response.Diagnostics) != tc.bad {
						t.Fatalf("identity upgrade: %v, %v", err, response.Diagnostics)
					}
					if tc.bad {
						return
					}
					// Framework v1.16.1 short-circuits a nil RawIdentity before
					// invoking the resource upgrader. Read must initialize it.
					if tc.raw == nil {
						if response.UpgradedIdentity != nil {
							t.Fatal("absent identity unexpectedly materialized before Read")
						}
						return
					}
					if response.UpgradedIdentity == nil || response.UpgradedIdentity.IdentityData == nil {
						t.Fatal("identity upgrader returned no identity")
					}
					value, err := response.UpgradedIdentity.IdentityData.Unmarshal(identityType)
					expected := wantIdentity
					if len(tc.raw.JSON) == 0 {
						expected = serviceAccountValue(identityType, nil)
					}
					if err != nil || !value.Equal(expected) {
						t.Fatalf("identity upgrade values: %v\n%s\nwant %s", err, value, expected)
					}
				})
			}
		})
	}
}

type serviceAccountScenario struct {
	name        string
	annotations map[string]string
	labels      map[string]string
	secrets     []string
	pullSecrets []string
	automount   string
	generate    bool
	timeout     bool
}

func serviceAccountScenarios() []serviceAccountScenario {
	return []serviceAccountScenario{
		{name: "minimum"},
		{name: "non-default",
			annotations: map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"},
			labels:      map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"},
			secrets:     []string{"one", "two"}, pullSecrets: []string{"three", "four"}, automount: "false", timeout: true},
		{name: "explicit-empty", annotations: map[string]string{}, labels: map[string]string{}, automount: "true"},
	}
}

func serviceAccountConfig(kind, label, namespaceExpr, name string, scenario serviceAccountScenario, live bool) string {
	var config strings.Builder
	fmt.Fprintf(&config, "resource %q %q {\n  metadata {\n    namespace = %s\n", kind, label, namespaceExpr)
	if scenario.generate {
		fmt.Fprintf(&config, "    generate_name = %q\n", name)
	} else if kind != "kubernetes_default_service_account_v1" && kind != "kubernetes_default_service_account" {
		fmt.Fprintf(&config, "    name = %q\n", name)
	}
	for _, field := range []struct {
		name  string
		value map[string]string
	}{{"annotations", scenario.annotations}, {"labels", scenario.labels}} {
		if field.value != nil {
			value, _ := json.Marshal(field.value)
			fmt.Fprintf(&config, "    %s = %s\n", field.name, value)
		}
	}
	config.WriteString("  }\n")
	for _, refs := range []struct {
		block string
		names []string
	}{{"secret", scenario.secrets}, {"image_pull_secret", scenario.pullSecrets}} {
		for _, ref := range refs.names {
			value := strconv.Quote("sa-" + ref)
			if live {
				value = fmt.Sprintf("kubernetes_secret.%s.metadata[0].name", ref)
			}
			fmt.Fprintf(&config, "  %s {\n    name = %s\n  }\n", refs.block, value)
		}
	}
	if scenario.automount != "" {
		fmt.Fprintf(&config, "  automount_service_account_token = %s\n", scenario.automount)
	}
	if scenario.timeout {
		config.WriteString("  timeouts {\n    create = \"45s\"\n  }\n")
	}
	config.WriteString("}\n")
	return config.String()
}

func serviceAccountLiveFixture(namespace string) string {
	var config strings.Builder
	fmt.Fprintf(&config, `
resource "kubernetes_namespace" "test" {
  metadata {
    name = %q
  }
  wait_for_default_service_account = true
}
`, namespace)
	// Explicit instances also work with plugin-testing's legacy terraform.State converter.
	for _, name := range []string{"one", "two", "three", "four"} {
		fmt.Fprintf(&config, `
resource "kubernetes_secret" %q {
  metadata {
    namespace = kubernetes_namespace.test.metadata[0].name
    name      = %q
  }
}
`, name, "sa-"+name)
	}
	return config.String()
}

func TestServiceAccountLiveFixture(t *testing.T) {
	_, schema := serviceAccountServer(t)
	for _, name := range []string{"kubernetes_namespace", "kubernetes_secret"} {
		if schema.ResourceSchemas[name] == nil {
			t.Fatalf("retained SDK fixture resource is missing: %s", name)
		}
	}
	for _, kind := range serviceAccountTypes {
		for _, scenario := range serviceAccountScenarios() {
			t.Run(kind.name+"/"+scenario.name, func(t *testing.T) {
				config := serviceAccountLiveFixture("owned-namespace") + serviceAccountConfig(kind.name, "test",
					"kubernetes_namespace.test.metadata[0].name", "account", scenario, true)
				file, diags := hclsyntax.ParseConfig([]byte(config), "fixture.tf", hcl.InitialPos)
				if diags.HasErrors() {
					t.Fatal(diags)
				}
				resources := make(map[string]*hclsyntax.Block)
				for _, block := range file.Body.(*hclsyntax.Body).Blocks {
					if block.Type != "resource" || len(block.Labels) != 2 {
						t.Fatalf("unexpected fixture block: %#v", block.Labels)
					}
					address := strings.Join(block.Labels, ".")
					if block.Body.Attributes["for_each"] != nil || block.Body.Attributes["count"] != nil {
						t.Fatalf("fixture must use explicit resources for the legacy state converter: %s", address)
					}
					resources[address] = block
				}
				if len(resources) != 6 || resources["kubernetes_namespace.test"] == nil || resources[kind.name+".test"] == nil {
					t.Fatalf("expected one SDK namespace, four SDK secrets, and the versioned SA: %v", resources)
				}
				for _, name := range []string{"one", "two", "three", "four"} {
					block := resources["kubernetes_secret."+name]
					if block == nil {
						t.Fatalf("missing explicit SDK secret fixture: %s", name)
					}
					metadata := block.Body.Blocks[0]
					value, diags := metadata.Body.Attributes["name"].Expr.Value(nil)
					if diags.HasErrors() || value.AsString() != "sa-"+name {
						t.Fatalf("secret fixture name changed: %s", name)
					}
				}
				for _, refs := range [][]string{scenario.secrets, scenario.pullSecrets} {
					for _, ref := range refs {
						if !strings.Contains(config, "name = kubernetes_secret."+ref+".metadata[0].name") {
							t.Fatalf("missing retained secret reference: %s", ref)
						}
					}
				}
			})
		}
	}
}

func serviceAccountIdentity(address, namespace, name string) statecheck.StateCheck {
	return statecheck.ExpectIdentity(address, map[string]knownvalue.Check{
		"api_version": knownvalue.StringExact("v1"), "kind": knownvalue.StringExact("ServiceAccount"),
		"namespace": knownvalue.StringExact(namespace), "name": knownvalue.StringExact(name),
	})
}

// The SDK scenarios are shared between both resources, with stronger exact API
// reference checks (the old helper accepted any one matching secret).
func TestAccKubernetesServiceAccountV1_lifecycle(t *testing.T) {
	for _, kind := range serviceAccountTypes {
		t.Run(kind.name, func(t *testing.T) {
			namespace := "tf-acc-sa-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			name := "account"
			if kind.adopt {
				name = "default"
			}
			address := kind.name + ".test"
			fixture := serviceAccountLiveFixture(namespace)
			full := serviceAccountScenarios()[1]
			full.automount, full.timeout = "", false
			updated := serviceAccountScenario{
				name: "updated", annotations: map[string]string{"TestAnnotationOne": "one", "Different": "1234"},
				labels:  map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"},
				secrets: []string{"one"}, pullSecrets: []string{"two", "three", "four"}, automount: "false",
			}
			var uid k8stypes.UID
			steps := []resource.TestStep{{
				Config: fixture,
				Check: func(_ *terraform.State) error {
					if !kind.adopt {
						return nil
					}
					client, err := mainClientset()
					if err != nil {
						return err
					}
					sa, err := client.CoreV1().ServiceAccounts(namespace).Get(context.Background(), name, metav1.GetOptions{})
					if err == nil {
						uid = sa.UID
					}
					return err
				},
			}}
			fullFalse := full
			fullFalse.automount = "false"
			scenarios := []serviceAccountScenario{full, fullFalse, updated, serviceAccountScenarios()[0], serviceAccountScenarios()[2]}
			if kind.adopt {
				basic := full
				basic.secrets, basic.pullSecrets = nil, nil
				scenarios = append([]serviceAccountScenario{
					basic,
					{name: "secrets", secrets: []string{"one"}, pullSecrets: []string{"two"}},
					{name: "automount", automount: "false"},
				}, scenarios...)
			}
			for _, scenario := range scenarios {
				config := fixture + serviceAccountConfig(kind.name, "test", "kubernetes_namespace.test.metadata[0].name", name, scenario, true) +
					serviceAccountSDKConsumer(address)
				steps = append(steps, resource.TestStep{
					Config: config,
					Check: func(state *terraform.State) error {
						client, err := mainClientset()
						if err != nil {
							return err
						}
						sa, err := client.CoreV1().ServiceAccounts(namespace).Get(context.Background(), name, metav1.GetOptions{})
						if err != nil {
							return err
						}
						if uid != "" && sa.UID != uid {
							return fmt.Errorf("adoption/update replaced ServiceAccount UID %s with %s", uid, sa.UID)
						}
						uid = sa.UID
						if err := serviceAccountCheckSDKConsumer(state, address, sa); err != nil {
							return err
						}
						return serviceAccountCheckValues(state, address, sa, scenario)
					},
					ConfigStateChecks: []statecheck.StateCheck{serviceAccountIdentity(address, namespace, name)},
				}, resource.TestStep{
					ResourceName: address, ImportState: true, ImportStateVerify: true,
				}, resource.TestStep{
					Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				})
			}
			steps = append(steps, resource.TestStep{
				ResourceName: address, ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity,
			})
			resource.ParallelTest(t, resource.TestCase{
				PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: serviceAccountMuxFactories,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
				CheckDestroy:           serviceAccountLiveDestroy, Steps: steps,
			})
		})
	}
}

func serviceAccountLiveDestroy(state *terraform.State) error {
	client, err := mainClientset()
	if err != nil {
		return err
	}
	for _, rs := range state.RootModule().Resources {
		managed := false
		for _, kind := range serviceAccountTypes {
			managed = managed || rs.Type == kind.name || rs.Type == kind.alias
		}
		if !managed {
			continue
		}
		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}
		_, err = client.CoreV1().ServiceAccounts(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("ServiceAccount still exists: %s", rs.Primary.ID)
		}
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func serviceAccountCheckValues(state *terraform.State, address string, sa *coreapi.ServiceAccount, scenario serviceAccountScenario) error {
	automount := scenario.automount != "false"
	if sa.AutomountServiceAccountToken == nil || *sa.AutomountServiceAccountToken != automount {
		return fmt.Errorf("API automount = %v, want %t", sa.AutomountServiceAccountToken, automount)
	}
	rs := state.RootModule().Resources[address]
	if rs == nil {
		return fmt.Errorf("missing state address %s", address)
	}
	token := rs.Primary.Attributes["default_secret_name"]
	for _, refs := range []struct {
		field string
		want  []string
		got   []string
	}{
		{"secret", scenario.secrets, serviceAccountSecretNames(sa.Secrets, token)},
		{"image_pull_secret", scenario.pullSecrets, serviceAccountPullNames(sa.ImagePullSecrets)},
	} {
		want := make([]string, len(refs.want))
		for i, name := range refs.want {
			want[i] = "sa-" + name
		}
		slices.Sort(want)
		slices.Sort(refs.got)
		if !slices.Equal(want, refs.got) {
			return fmt.Errorf("API %s = %v, want %v", refs.field, refs.got, want)
		}
		if err := resource.TestCheckResourceAttr(address, refs.field+".#", strconv.Itoa(len(want)))(state); err != nil {
			return err
		}
		for _, name := range want {
			if err := resource.TestCheckTypeSetElemNestedAttrs(address, refs.field+".*", map[string]string{"name": name})(state); err != nil {
				return err
			}
		}
	}
	for _, field := range []struct {
		name string
		want map[string]string
		got  map[string]string
	}{{"annotations", scenario.annotations, sa.Annotations}, {"labels", scenario.labels, sa.Labels}} {
		count := rs.Primary.Attributes["metadata.0."+field.name+".%"]
		if count != strconv.Itoa(len(field.want)) && !(count == "" && len(field.want) == 0) {
			return fmt.Errorf("metadata.%s count = %q, want %d", field.name, count, len(field.want))
		}
		for key, value := range field.want {
			if field.got[key] != value {
				return fmt.Errorf("API metadata.%s[%s] = %q, want %q", field.name, key, field.got[key], value)
			}
			if err := resource.TestCheckResourceAttr(address, "metadata.0."+field.name+"."+key, value)(state); err != nil {
				return err
			}
		}
		for key := range field.got {
			if !strings.Contains(key, "kubernetes.io/") {
				if _, ok := field.want[key]; !ok {
					return fmt.Errorf("removed metadata.%s[%s] remains in API", field.name, key)
				}
			}
		}
	}
	for key, value := range map[string]string{
		"id": sa.Namespace + "/" + sa.Name, "metadata.0.name": sa.Name, "metadata.0.namespace": sa.Namespace,
		"metadata.0.uid": string(sa.UID), "metadata.0.resource_version": sa.ResourceVersion,
		"metadata.0.generation": strconv.FormatInt(sa.Generation, 10), "automount_service_account_token": strconv.FormatBool(automount),
	} {
		if err := resource.TestCheckResourceAttr(address, key, value)(state); err != nil {
			return err
		}
	}
	return nil
}

func serviceAccountSecretNames(refs []coreapi.ObjectReference, token string) []string {
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Name != token {
			names = append(names, ref.Name)
		}
	}
	return names
}

func serviceAccountPullNames(refs []coreapi.LocalObjectReference) []string {
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, ref.Name)
	}
	return names
}

func TestAccKubernetesServiceAccountV1_generateName_disappears(t *testing.T) {
	namespace := "tf-acc-sa-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	prefix := "generated-" + acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum) + "-"
	address := "kubernetes_service_account_v1.test"
	scenario := serviceAccountScenario{generate: true}
	config := serviceAccountLiveFixture(namespace) + serviceAccountConfig(serviceAccountTypes[0].name, "test",
		"kubernetes_namespace.test.metadata[0].name", prefix, scenario, true)
	scenario.labels, scenario.automount = map[string]string{"mutable": "updated"}, "false"
	updated := serviceAccountLiveFixture(namespace) + serviceAccountConfig(serviceAccountTypes[0].name, "test",
		"kubernetes_namespace.test.metadata[0].name", prefix, scenario, true)
	var generatedName string
	var uid k8stypes.UID
	checkUnchangedIdentity := func(state *terraform.State) error {
		name := state.RootModule().Resources[address].Primary.Attributes["metadata.0.name"]
		object, err := serviceAccountLiveGet(namespace, name)
		if err != nil {
			return err
		}
		if name != generatedName || object.UID != uid || object.GenerateName != prefix {
			return fmt.Errorf("unrelated mutable edit changed generated name, prefix, or UID")
		}
		return serviceAccountCheckValues(state, address, object, scenario)
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: serviceAccountMuxFactories,
		CheckDestroy: serviceAccountLiveDestroy,
		Steps: []resource.TestStep{
			{Config: config, Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(address, "metadata.0.generate_name", prefix),
				resource.TestMatchResourceAttr(address, "metadata.0.name", regexp.MustCompile("^"+prefix)),
				func(state *terraform.State) error {
					generatedName = state.RootModule().Resources[address].Primary.Attributes["metadata.0.name"]
					object, err := serviceAccountLiveGet(namespace, generatedName)
					if err == nil {
						uid = object.UID
					}
					return err
				},
			)},
			{ResourceName: address, ImportState: true, ImportStateVerify: true},
			{
				Config: updated, Check: checkUnchangedIdentity,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
				}},
			},
			{
				Config: updated, Check: checkUnchangedIdentity,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{Config: updated, ExpectNonEmptyPlan: true, Check: func(state *terraform.State) error {
				client, err := mainClientset()
				if err != nil {
					return err
				}
				_, name, err := kubernetes.IdParts(state.RootModule().Resources[address].Primary.ID)
				if err != nil {
					return err
				}
				return client.CoreV1().ServiceAccounts(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
			}},
			{Config: updated, Check: resource.TestMatchResourceAttr(address, "metadata.0.name", regexp.MustCompile("^"+prefix))},
		},
	})
}

func serviceAccountLiveGet(namespace, name string) (*coreapi.ServiceAccount, error) {
	client, err := mainClientset()
	if err != nil {
		return nil, err
	}
	return client.CoreV1().ServiceAccounts(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func serviceAccountLiveMigrationEnvironment(t *testing.T) string {
	t.Helper()
	// Keep HOME and all KUBE settings: live tests must use the explicitly chosen cluster.
	if os.Getenv("TF_ACC") == "" {
		t.Skip("live migration requires TF_ACC")
	}
	cli, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal(err)
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository-local test workspace")
	}
	t.Setenv("TMPDIR", filepath.Clean(filepath.Join(filepath.Dir(file), "../../../..")))
	dir := t.TempDir()
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "TF_CLI_ARGS") || strings.HasPrefix(name, "TF_VAR_") ||
			name == "TF_REATTACH_PROVIDERS" || name == "TF_PLUGIN_CACHE_DIR" || name == "TF_ACC_TERRAFORM_VERSION" ||
			name == "TF_DATA_DIR" || name == "TF_WORKSPACE" {
			t.Setenv(name, "")
		}
	}
	rc := filepath.Join(dir, "terraformrc")
	if err := os.WriteFile(rc, []byte("disable_checkpoint = true\nprovider_installation {\n  direct {}\n}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TF_CLI_CONFIG_FILE", rc)
	t.Setenv("TF_ACC_TERRAFORM_PATH", cli)
	t.Setenv("TF_ACC_TEMP_DIR", dir)
	t.Setenv("CHECKPOINT_DISABLE", "1")
	return dir
}

const serviceAccountControllerAnnotation = "controller.kubernetes.io/service-account-migration"

type serviceAccountLiveSnapshot struct {
	read     func() (*coreapi.ServiceAccount, error)
	object   *coreapi.ServiceAccount
	resource *tfjson.StateResource
	outputs  map[string]*tfjson.StateOutput
}

func serviceAccountStateResource(values *tfjson.StateValues, address string) *tfjson.StateResource {
	if values == nil || values.RootModule == nil {
		return nil
	}
	for _, r := range values.RootModule.Resources {
		if r.Address == address {
			return r
		}
	}
	return nil
}

func (snapshot *serviceAccountLiveSnapshot) check(address string, capture bool) statecheck.StateCheck {
	return serviceAccountStateCheck(func(req statecheck.CheckStateRequest) error {
		r := serviceAccountStateResource(req.State.Values, address)
		if r == nil || r.SchemaVersion != 0 || r.Tainted || r.DeposedKey != "" {
			return fmt.Errorf("missing or invalid ServiceAccount schema0 state at %s", address)
		}
		object, err := snapshot.read()
		if err != nil {
			return err
		}
		if object.Annotations[serviceAccountControllerAnnotation] != "preserve" {
			return fmt.Errorf("controller annotation disappeared")
		}
		if capture {
			snapshot.object, snapshot.resource, snapshot.outputs = object.DeepCopy(), r, req.State.Values.Outputs
			return nil
		}
		normalized := *r
		normalized.Address, normalized.Type, normalized.Name = snapshot.resource.Address, snapshot.resource.Type, snapshot.resource.Name
		// Pre-identity releases legitimately gain identity1, checked independently below.
		if snapshot.resource.IdentitySchemaVersion == nil || *snapshot.resource.IdentitySchemaVersion == 0 {
			normalized.IdentitySchemaVersion, normalized.IdentityValues = snapshot.resource.IdentitySchemaVersion, snapshot.resource.IdentityValues
		}
		if !reflect.DeepEqual(snapshot.resource, &normalized) || !reflect.DeepEqual(snapshot.outputs, req.State.Values.Outputs) ||
			!reflect.DeepEqual(snapshot.object, object) {
			return fmt.Errorf("migration changed complete SA state, outputs, or API object (including UID/resourceVersion)")
		}
		return nil
	})
}

type serviceAccountLivePlanGuard struct {
	snapshot *serviceAccountLiveSnapshot
	address  string
}

func (guard serviceAccountLivePlanGuard) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if guard.snapshot == nil || guard.snapshot.resource == nil || guard.snapshot.object == nil {
		resp.Error = fmt.Errorf("live migration has no protected baseline")
		return
	}
	if req.Plan == nil || (req.Plan.Complete != nil && !*req.Plan.Complete) || len(req.Plan.DeferredChanges) != 0 {
		resp.Error = fmt.Errorf("live migration plan is incomplete")
		return
	}
	r := serviceAccountStateResource(req.Plan.PlannedValues, guard.address)
	if r == nil || r.SchemaVersion != 0 || !reflect.DeepEqual(r.AttributeValues, guard.snapshot.resource.AttributeValues) ||
		!reflect.DeepEqual(req.Plan.PlannedValues.Outputs, guard.snapshot.outputs) {
		resp.Error = fmt.Errorf("live migration changed protected planned SA state or outputs before apply")
		return
	}
	for _, drift := range req.Plan.ResourceDrift {
		if drift.Address == guard.address || drift.Address == guard.snapshot.resource.Address {
			if drift.Change == nil || !drift.Change.Actions.NoOp() || !reflect.DeepEqual(drift.Change.Before, drift.Change.After) {
				resp.Error = fmt.Errorf("live migration normalized protected SA state during refresh")
				return
			}
		}
	}
	object, err := guard.snapshot.read()
	if err != nil {
		resp.Error = err
	} else if !reflect.DeepEqual(object, guard.snapshot.object) {
		resp.Error = fmt.Errorf("live migration wrote or replaced the SA before apply")
	}
}

func TestServiceAccountLiveMigrationGuards(t *testing.T) {
	const address = "kubernetes_service_account_v1.test"
	for _, tc := range []string{"unchanged", "missing-account", "changed-state", "changed-output", "fixture-update", "changed-uid", "changed-resource-version"} {
		t.Run(tc, func(t *testing.T) {
			object := &coreapi.ServiceAccount{ObjectMeta: metav1.ObjectMeta{UID: "original", ResourceVersion: "1"}}
			current := object.DeepCopy()
			baseline := &tfjson.StateResource{Address: address, AttributeValues: map[string]any{"id": "owned/account"}}
			planned := &tfjson.StateResource{Address: address, AttributeValues: map[string]any{"id": "owned/account"}}
			snapshot := &serviceAccountLiveSnapshot{
				object: object, resource: baseline,
				read: func() (*coreapi.ServiceAccount, error) { return current, nil },
			}
			plan := &tfjson.Plan{
				PlannedValues: &tfjson.StateValues{RootModule: &tfjson.StateModule{Resources: []*tfjson.StateResource{planned}}},
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: address, Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionNoop}}},
					{Address: "kubernetes_namespace.test", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionNoop}}},
				},
			}
			switch tc {
			case "missing-account":
				plan.PlannedValues.RootModule.Resources = nil
			case "changed-state":
				planned.AttributeValues["id"] = "owned/different"
			case "changed-output":
				plan.PlannedValues.Outputs = map[string]*tfjson.StateOutput{"account": {Value: "different"}}
			case "fixture-update":
				plan.ResourceChanges[1].Change.Actions = tfjson.Actions{tfjson.ActionUpdate}
			case "changed-uid":
				current.UID = "replacement"
			case "changed-resource-version":
				current.ResourceVersion = "2"
			}
			failed := false
			for _, check := range []plancheck.PlanCheck{plancheck.ExpectEmptyPlan(), serviceAccountLivePlanGuard{snapshot: snapshot, address: address}} {
				var response plancheck.CheckPlanResponse
				check.CheckPlan(context.Background(), plancheck.CheckPlanRequest{Plan: plan}, &response)
				failed = failed || response.Error != nil
			}
			if failed != (tc != "unchanged") {
				t.Fatalf("guard rejected plan=%t, want %t", failed, tc != "unchanged")
			}
		})
	}
}

func TestAccKubernetesServiceAccountV1_migration(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("live exact-release migrations require TF_ACC")
	}
	// Serial execution is required by the per-case CLI configuration environment.
	for _, kind := range serviceAccountTypes {
		for _, version := range []string{"3.2.1", "2.37.1"} {
			for _, move := range []string{"same-type", "alias-move", "address-move"} {
				t.Run(kind.name+"/"+version+"/"+move, func(t *testing.T) {
					testAccPreCheck(t)
					dir := serviceAccountLiveMigrationEnvironment(t)
					namespace := "tf-acc-sa-migrate-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
					name := "account"
					if kind.adopt {
						name = "default"
					}
					sourceType, sourceLabel := kind.name, "test"
					if move == "alias-move" {
						sourceType = kind.alias
					} else if move == "address-move" {
						sourceLabel = "old"
					}
					source, target := sourceType+"."+sourceLabel, kind.name+".test"
					// Fixtures stay on SDKv2 throughout; this matrix migrates only the SA.
					fixture := serviceAccountLiveFixture(namespace)
					scenario := serviceAccountScenarios()[1]
					build := func(kind, label string) string {
						return fixture + serviceAccountConfig(kind, label, "kubernetes_namespace.test.metadata[0].name", name, scenario, true) +
							serviceAccountOutputs(kind+"."+label)
					}
					before, after := build(sourceType, sourceLabel), build(kind.name, "test")
					if move != "same-type" {
						after += fmt.Sprintf("\nmoved {\n  from = %s\n  to = %s\n}\n", source, target)
					} else if before != after {
						t.Fatal("same-type upgrade HCL must be byte-identical")
					}
					snapshot := &serviceAccountLiveSnapshot{read: func() (*coreapi.ServiceAccount, error) {
						return serviceAccountLiveGet(namespace, name)
					}}
					released := map[string]resource.ExternalProvider{"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= " + version}}
					check := func(address string) resource.TestCheckFunc {
						return func(state *terraform.State) error {
							object, err := snapshot.read()
							if err != nil {
								return err
							}
							return serviceAccountCheckValues(state, address, object, scenario)
						}
					}
					guards := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(), serviceAccountLivePlanGuard{snapshot: snapshot, address: target},
					}}
					resource.Test(t, resource.TestCase{
						PreCheck:               func() { testAccPreCheck(t) },
						TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
						CheckDestroy: func(state *terraform.State) error {
							if err := serviceAccountLiveDestroy(state); err != nil {
								return err
							}
							client, err := mainClientset()
							if err != nil {
								return err
							}
							_, err = client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
							if apierrors.IsNotFound(err) {
								return nil
							}
							if err == nil {
								return fmt.Errorf("test-owned namespace still exists: %s", namespace)
							}
							return err
						},
						Steps: []resource.TestStep{
							{ExternalProviders: released, Config: before, Check: resource.ComposeAggregateTestCheckFunc(serviceAccountReleasedVersion(dir, version), check(source))},
							{
								PreConfig: func() {
									client, err := mainClientset()
									if err != nil {
										t.Fatal(err)
									}
									patch, _ := json.Marshal(map[string]any{"metadata": map[string]any{"annotations": map[string]string{serviceAccountControllerAnnotation: "preserve"}}})
									if _, err := client.CoreV1().ServiceAccounts(namespace).Patch(context.Background(), name, k8stypes.MergePatchType, patch,
										metav1.PatchOptions{FieldManager: "service-account-migration-controller"}); err != nil {
										t.Fatal(err)
									}
								},
								ExternalProviders: released, Config: before, Check: check(source),
								ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
								ConfigStateChecks: []statecheck.StateCheck{snapshot.check(source, true)},
							},
							{
								ProtoV6ProviderFactories: serviceAccountMuxFactories, Config: after, Check: check(target),
								ConfigPlanChecks:  guards,
								ConfigStateChecks: []statecheck.StateCheck{snapshot.check(target, false), serviceAccountIdentity(target, namespace, name)},
							},
							{
								ProtoV6ProviderFactories: serviceAccountMuxFactories, Config: after, Check: check(target),
								ConfigPlanChecks:  guards,
								ConfigStateChecks: []statecheck.StateCheck{snapshot.check(target, false), serviceAccountIdentity(target, namespace, name)},
							},
						},
					})
				})
			}
		}
	}
}

func serviceAccountCLIEnvironment(t *testing.T) string {
	t.Helper()
	if os.Getenv("TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS") != "1" {
		t.Skip("set TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS=1 for scoped loopback Terraform tests")
	}
	cli, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("installed Terraform required:", err)
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository-local CLI workspace")
	}
	// A short repository-local path also avoids macOS's provider socket path limit.
	t.Setenv("TMPDIR", filepath.Clean(filepath.Join(filepath.Dir(file), "../../../..")))
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "KUBE") || strings.HasPrefix(name, "TF_") {
			t.Setenv(name, "")
		}
	}
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("TF_ACC_TEMP_DIR", dir)
	t.Setenv("TF_ACC_TERRAFORM_PATH", cli)
	t.Setenv("CHECKPOINT_DISABLE", "1")
	t.Setenv("TF_IN_AUTOMATION", "1")
	rc := filepath.Join(dir, "terraformrc")
	if err := os.WriteFile(rc, []byte("disable_checkpoint = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TF_CLI_CONFIG_FILE", rc)
	return dir
}

// This fixture serves only one ServiceAccount in a non-ambient namespace. All
// unexpected endpoints and writes fail the test, not just the HTTP request.
type serviceAccountAPI struct {
	t                 *testing.T
	server            *httptest.Server
	mu                sync.Mutex
	object            *coreapi.ServiceAccount
	adopt             bool
	phase             string
	calls             map[string]map[string]int
	baseline          *coreapi.ServiceAccount
	values            map[string]any
	outputs           map[string]*tfjson.StateOutput
	protectReferences bool
	version           string
	token             *coreapi.Secret
	tokenReference    coreapi.ObjectReference
}

const serviceAccountCLINamespace = "sa-loopback-owned"

func newServiceAccountAPI(t *testing.T, adopt bool) *serviceAccountAPI {
	t.Helper()
	api := &serviceAccountAPI{t: t, adopt: adopt, phase: "baseline", version: "v1.33.6", calls: map[string]map[string]int{}}
	if adopt {
		api.object = &coreapi.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: serviceAccountCLINamespace}}
		api.decorate(api.object)
	}
	api.server = httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	t.Cleanup(func() {
		api.server.Close()
		api.mu.Lock()
		defer api.mu.Unlock()
		t.Logf("ServiceAccount loopback API audit: %v", api.calls)
		if api.object != nil {
			t.Error("ServiceAccount fixture leaked: destroy did not DELETE the object")
		}
	})
	return api
}

func (api *serviceAccountAPI) decorate(object *coreapi.ServiceAccount) {
	object.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"}
	object.UID = "sa-loopback-uid"
	object.ResourceVersion = "1"
	object.CreationTimestamp = metav1.NewTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if object.Annotations == nil {
		object.Annotations = map[string]string{}
	}
	object.Annotations["controller.kubernetes.io/observed"] = "preserve"
	object.OwnerReferences = []metav1.OwnerReference{{APIVersion: "v1", Kind: "Namespace", Name: serviceAccountCLINamespace, UID: "namespace-owner"}}
	object.ManagedFields = []metav1.ManagedFieldsEntry{{Manager: "fixture-controller", Operation: metav1.ManagedFieldsOperationUpdate, APIVersion: "v1"}}
}

func (api *serviceAccountAPI) setPhase(phase string) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.phase = phase
}

func (api *serviceAccountAPI) checkDestroy(_ *terraform.State) error {
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.object != nil {
		return fmt.Errorf("ServiceAccount still exists after cleanup")
	}
	return nil
}

func serviceAccountHTTPJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func (api *serviceAccountAPI) serveHTTP(w http.ResponseWriter, r *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.calls[api.phase] == nil {
		api.calls[api.phase] = map[string]int{}
	}
	api.calls[api.phase][r.Method+" "+r.URL.Path]++
	deny := func(message string) {
		api.t.Errorf("%s: %s %s during %s", message, r.Method, r.URL.Path, api.phase)
		serviceAccountHTTPJSON(w, http.StatusForbidden, &metav1.Status{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: metav1.StatusFailure,
			Reason: metav1.StatusReasonForbidden, Code: http.StatusForbidden, Message: message,
		})
	}
	if r.Method == http.MethodGet && r.URL.Path == "/version" {
		version := strings.Split(strings.TrimPrefix(api.version, "v"), ".")
		serviceAccountHTTPJSON(w, http.StatusOK, map[string]string{"major": version[0], "minor": version[1], "gitVersion": api.version})
		return
	}
	if api.token != nil && r.Method == http.MethodGet &&
		r.URL.Path == "/api/v1/namespaces/"+serviceAccountCLINamespace+"/secrets/"+api.token.Name {
		serviceAccountHTTPJSON(w, http.StatusOK, api.token)
		return
	}
	prefix := "/api/v1/namespaces/" + serviceAccountCLINamespace + "/serviceaccounts"
	name := "account"
	if api.adopt {
		name = "default"
	}
	if r.URL.Path != prefix && r.URL.Path != prefix+"/"+name {
		deny("unexpected API endpoint")
		return
	}
	write := r.Method != http.MethodGet
	if write && api.phase != "baseline" && api.phase != "cleanup" && api.phase != "update" {
		deny("unchanged migration/import must not write")
		return
	}
	if write && ((api.phase == "cleanup" && r.Method != http.MethodDelete) ||
		(api.phase == "update" && r.Method != http.MethodPatch) ||
		(api.phase == "baseline" && ((api.adopt && r.Method != http.MethodPatch) || (!api.adopt && r.Method != http.MethodPost)))) {
		deny("unexpected lifecycle write; default Create must adopt with PATCH and Delete is cleanup-only")
		return
	}
	if r.Method != http.MethodPost && (api.object == nil || r.URL.Path == prefix) {
		serviceAccountHTTPJSON(w, http.StatusNotFound, &metav1.Status{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: metav1.StatusFailure,
			Reason: metav1.StatusReasonNotFound, Code: http.StatusNotFound, Message: "ServiceAccount not found",
		})
		return
	}
	switch r.Method {
	case http.MethodGet:
		serviceAccountHTTPJSON(w, http.StatusOK, api.object)
	case http.MethodDelete:
		api.object = nil
		serviceAccountHTTPJSON(w, http.StatusOK, &metav1.Status{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: metav1.StatusSuccess})
	case http.MethodPost, http.MethodPatch:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			deny(err.Error())
			return
		}
		if r.Method == http.MethodPatch {
			if r.Header.Get("Content-Type") != string(k8stypes.JSONPatchType) {
				deny("expected surgical JSON Patch")
				return
			}
			patch, err := jsonpatch.DecodePatch(body)
			if err != nil {
				deny(err.Error())
				return
			}
			if api.protectReferences {
				var operations []struct {
					Path string `json:"path"`
				}
				if err := json.Unmarshal(body, &operations); err != nil {
					deny(err.Error())
					return
				}
				for _, operation := range operations {
					if operation.Path == "/secrets" || strings.HasPrefix(operation.Path, "/secrets/") ||
						operation.Path == "/imagePullSecrets" || strings.HasPrefix(operation.Path, "/imagePullSecrets/") {
						deny("omitted reference blocks must not mutate controller references")
						return
					}
				}
			}
			old, _ := json.Marshal(api.object)
			body, err = patch.Apply(old)
			if err != nil {
				deny(err.Error())
				return
			}
		} else if api.object != nil || r.URL.Path != prefix {
			deny("duplicate or noncollection POST")
			return
		}
		object := &coreapi.ServiceAccount{}
		if _, _, err := clientgoscheme.Codecs.UniversalDeserializer().Decode(body, nil, object); err != nil {
			deny(err.Error())
			return
		}
		if object.Name != name || object.Namespace != serviceAccountCLINamespace {
			deny("wrong ServiceAccount identity")
			return
		}
		if api.token != nil && !slices.Contains(object.Secrets, api.tokenReference) {
			deny("update must retain the complete imported token reference")
			return
		}
		if r.Method == http.MethodPost {
			api.decorate(object)
		} else {
			object.ResourceVersion = "2"
		}
		api.object = object
		serviceAccountHTTPJSON(w, http.StatusOK, object)
	default:
		deny("unsupported method")
	}
}

func serviceAccountCLIConfig(api *serviceAccountAPI, kind, label string, scenario serviceAccountScenario) string {
	address := kind + "." + label
	return fmt.Sprintf("provider \"kubernetes\" {\n  host = %q\n}\n", api.server.URL) +
		serviceAccountConfig(kind, label, strconv.Quote(serviceAccountCLINamespace), "account", scenario, false) +
		serviceAccountOutputs(address)
}

func serviceAccountOutputs(address string) string {
	return strings.ReplaceAll(`
output "account" {
  value = {
    id             = ACCOUNT_ADDRESS.id
    uid            = ACCOUNT_ADDRESS.metadata[0].uid
    name           = ACCOUNT_ADDRESS.metadata[0].name
    namespace      = ACCOUNT_ADDRESS.metadata[0].namespace
    annotations    = ACCOUNT_ADDRESS.metadata[0].annotations
    labels         = ACCOUNT_ADDRESS.metadata[0].labels
    secret_names   = sort([for ref in ACCOUNT_ADDRESS.secret : ref.name])
    pull_names     = sort([for ref in ACCOUNT_ADDRESS.image_pull_secret : ref.name])
    automount      = ACCOUNT_ADDRESS.automount_service_account_token
    default_secret = ACCOUNT_ADDRESS.default_secret_name
  }
}
`, "ACCOUNT_ADDRESS", address)
}

type serviceAccountStateCheck func(statecheck.CheckStateRequest) error

func (check serviceAccountStateCheck) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	resp.Error = check(req)
}

func (api *serviceAccountAPI) checkState(address string, baseline bool) statecheck.StateCheck {
	return serviceAccountStateCheck(func(req statecheck.CheckStateRequest) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		if api.object == nil {
			return fmt.Errorf("ServiceAccount vanished from API")
		}
		root := req.State.Values.RootModule
		if len(root.Resources) != 1 || root.Resources[0].Address != address {
			return fmt.Errorf("expected exactly the protected address %s, got %+v", address, root.Resources)
		}
		res := root.Resources[0]
		if res.SchemaVersion != 0 {
			return fmt.Errorf("resource schema version changed: %d", res.SchemaVersion)
		}
		if baseline {
			api.baseline, api.values, api.outputs = api.object.DeepCopy(), res.AttributeValues, req.State.Values.Outputs
		} else if !reflect.DeepEqual(api.baseline, api.object) || !reflect.DeepEqual(api.values, res.AttributeValues) ||
			!reflect.DeepEqual(api.outputs, req.State.Values.Outputs) {
			return fmt.Errorf("migration changed complete remote object, typed state, or outputs:\nstate before=%#v\nstate after=%#v", api.values, res.AttributeValues)
		}
		meta := res.AttributeValues["metadata"].([]any)[0].(map[string]any)
		if meta["uid"] != string(api.object.UID) || meta["name"] != api.object.Name || meta["namespace"] != serviceAccountCLINamespace ||
			res.AttributeValues["id"] != serviceAccountCLINamespace+"/"+api.object.Name || res.AttributeValues["default_secret_name"] != "" {
			return fmt.Errorf("state identity/default token output does not match API")
		}
		output := req.State.Values.Outputs["account"]
		if output == nil {
			return fmt.Errorf("ServiceAccount consumer output is missing")
		}
		values, ok := output.Value.(map[string]any)
		if !ok || values["uid"] != string(api.object.UID) || values["id"] != serviceAccountCLINamespace+"/"+api.object.Name ||
			values["name"] != api.object.Name || values["namespace"] != serviceAccountCLINamespace || values["default_secret"] != "" ||
			api.object.AutomountServiceAccountToken == nil || values["automount"] != *api.object.AutomountServiceAccountToken {
			return fmt.Errorf("ServiceAccount consumer identity/automount/token output differs from API")
		}
		for key, want := range map[string][]string{
			"secret_names": serviceAccountSecretNames(api.object.Secrets, ""), "pull_names": serviceAccountPullNames(api.object.ImagePullSecrets),
		} {
			slices.Sort(want)
			expected, _ := json.Marshal(want)
			actual, _ := json.Marshal(values[key])
			if string(expected) != string(actual) {
				return fmt.Errorf("consumer %s = %s, want %s", key, actual, expected)
			}
		}
		if api.object.Annotations["controller.kubernetes.io/observed"] != "preserve" ||
			len(api.object.OwnerReferences) != 1 || len(api.object.ManagedFields) != 1 {
			return fmt.Errorf("controller-owned metadata was removed")
		}
		return nil
	})
}

type serviceAccountPlanGuard struct {
	address string
	api     *serviceAccountAPI
}

func (guard serviceAccountPlanGuard) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if req.Plan == nil || (req.Plan.Complete != nil && !*req.Plan.Complete) || len(req.Plan.DeferredChanges) != 0 {
		resp.Error = fmt.Errorf("migration plan is missing or incomplete")
		return
	}
	if len(req.Plan.ResourceChanges) != 1 || req.Plan.ResourceChanges[0] == nil || req.Plan.ResourceChanges[0].Address != guard.address {
		resp.Error = fmt.Errorf("migration lost/added a protected address: expected %s", guard.address)
		return
	}
	for _, change := range append(slices.Clone(req.Plan.ResourceChanges), req.Plan.ResourceDrift...) {
		if change == nil || change.Change == nil {
			resp.Error = fmt.Errorf("migration returned a missing resource change")
			return
		}
		if !change.Change.Actions.NoOp() ||
			!reflect.DeepEqual(change.Change.Before, change.Change.After) {
			resp.Error = fmt.Errorf("migration must not change %s actions=%v or refreshed values:\nbefore=%#v\nafter=%#v",
				change.Address, change.Change.Actions, change.Change.Before, change.Change.After)
			return
		}
	}
	for name, change := range req.Plan.OutputChanges {
		if change == nil || !change.Actions.NoOp() || !reflect.DeepEqual(change.Before, change.After) {
			resp.Error = fmt.Errorf("migration changed output %s", name)
			return
		}
	}
	guard.api.mu.Lock()
	defer guard.api.mu.Unlock()
	if guard.api.object == nil || !reflect.DeepEqual(guard.api.object, guard.api.baseline) {
		resp.Error = fmt.Errorf("migration changed the API object before apply")
		return
	}
	if guard.api.values != nil {
		planned := req.Plan.PlannedValues
		if planned == nil || planned.RootModule == nil || len(planned.RootModule.Resources) != 1 ||
			planned.RootModule.Resources[0].Address != guard.address {
			resp.Error = fmt.Errorf("migration lost the protected resource from planned values")
			return
		}
		if !reflect.DeepEqual(planned.RootModule.Resources[0].AttributeValues, guard.api.values) ||
			!reflect.DeepEqual(planned.Outputs, guard.api.outputs) {
			resp.Error = fmt.Errorf("migration changed snapshotted typed state or outputs before apply")
		}
	}
}

func TestServiceAccountPlanGuard(t *testing.T) {
	const address = "kubernetes_service_account_v1.test"
	for _, tc := range []struct {
		name    string
		actions tfjson.Actions
		mutate  func(*tfjson.Plan)
		bad     bool
	}{
		{"no-op", tfjson.Actions{tfjson.ActionNoop}, nil, false},
		{"update", tfjson.Actions{tfjson.ActionUpdate}, nil, true},
		{"delete", tfjson.Actions{tfjson.ActionDelete}, nil, true},
		{"replacement", tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}, nil, true},
		{"create", tfjson.Actions{tfjson.ActionCreate}, nil, true},
		{"missing-address", tfjson.Actions{tfjson.ActionNoop}, func(p *tfjson.Plan) { p.ResourceChanges = nil }, true},
		{"drift", tfjson.Actions{tfjson.ActionNoop}, func(p *tfjson.Plan) {
			p.ResourceDrift = []*tfjson.ResourceChange{{Address: address, Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionUpdate}}}}
		}, true},
		{"changed-output", tfjson.Actions{tfjson.ActionNoop}, func(p *tfjson.Plan) {
			p.OutputChanges = map[string]*tfjson.Change{"account": {Actions: tfjson.Actions{tfjson.ActionUpdate}}}
		}, true},
		{"incomplete", tfjson.Actions{tfjson.ActionNoop}, func(p *tfjson.Plan) { p.Complete = ptr.To(false) }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			object := &coreapi.ServiceAccount{ObjectMeta: metav1.ObjectMeta{UID: "original"}}
			api := &serviceAccountAPI{object: object, baseline: object.DeepCopy()}
			plan := &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{{Address: address, Change: &tfjson.Change{
				Actions: tc.actions, Before: map[string]any{"id": "owned/account"}, After: map[string]any{"id": "owned/account"},
			}}}}
			if tc.mutate != nil {
				tc.mutate(plan)
			}
			var response plancheck.CheckPlanResponse
			serviceAccountPlanGuard{address: address, api: api}.CheckPlan(context.Background(), plancheck.CheckPlanRequest{Plan: plan}, &response)
			if (response.Error != nil) != tc.bad {
				t.Fatalf("plan guard error=%v, want error=%t", response.Error, tc.bad)
			}
		})
	}
}

func serviceAccountReleasedVersion(dir, version string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		found, installed := false, false
		installation := "/registry.terraform.io/hashicorp/kubernetes/" + version + "/" + runtime.GOOS + "_" + runtime.GOARCH
		pattern := regexp.MustCompile(`(?s)provider "registry.terraform.io/hashicorp/kubernetes" \{\s*version\s*=\s*"` + regexp.QuoteMeta(version) + `"`)
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(filepath.ToSlash(path), installation) {
				// ReadDir follows a platform-directory symlink when Terraform uses a cache.
				binaries, err := os.ReadDir(path)
				if err != nil {
					return err
				}
				for _, binary := range binaries {
					prefix := "terraform-provider-kubernetes_v" + version
					if binary.Name() == prefix || binary.Name() == prefix+".exe" || strings.HasPrefix(binary.Name(), prefix+"_") {
						info, err := os.Stat(filepath.Join(path, binary.Name()))
						if err != nil {
							return err
						}
						installed = installed || info.Mode().IsRegular()
					}
				}
			}
			if entry.Name() != ".terraform.lock.hcl" {
				return nil
			}
			lock, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !pattern.Match(lock) {
				return fmt.Errorf("external provider is not exactly %s: %s", version, path)
			}
			found = true
			return nil
		})
		if err != nil {
			return err
		}
		if !found || !installed {
			return fmt.Errorf("released provider %s not verified: lock=%t, installed binary=%t", version, found, installed)
		}
		return nil
	}
}

// Released SDKv2 providers expose null metadata maps on Create, then empty maps
// on refresh. Settle that baseline-only output change before measuring migration;
// no such exception is granted to the local provider's migration plan.
type serviceAccountBaselineGuard struct{}

func (serviceAccountBaselineGuard) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	normalize := func(value any, metadata bool) any {
		data, _ := json.Marshal(value)
		var result map[string]any
		_ = json.Unmarshal(data, &result)
		target := result
		if metadata {
			blocks, ok := result["metadata"].([]any)
			if !ok || len(blocks) != 1 {
				return result
			}
			target, _ = blocks[0].(map[string]any)
		}
		for _, field := range []string{"annotations", "labels"} {
			if target[field] == nil {
				target[field] = map[string]any{}
			}
		}
		return result
	}
	for _, change := range req.Plan.ResourceChanges {
		if change.Change == nil || !change.Change.Actions.NoOp() {
			resp.Error = fmt.Errorf("released baseline refresh attempted a resource action: %+v", change)
			return
		}
	}
	for _, change := range req.Plan.ResourceDrift {
		if !reflect.DeepEqual(normalize(change.Change.Before, true), normalize(change.Change.After, true)) {
			resp.Error = fmt.Errorf("unexpected released baseline drift: %+v", change.Change)
			return
		}
	}
	for name, change := range req.Plan.OutputChanges {
		if name != "account" || !reflect.DeepEqual(normalize(change.Before, false), normalize(change.After, false)) {
			resp.Error = fmt.Errorf("unexpected released baseline output change: %s", name)
			return
		}
	}
}

func TestServiceAccountCLIMigration(t *testing.T) {
	// Gate before changing any global environment; these tests must never parallelize.
	if os.Getenv("TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS") != "1" {
		t.Skip("set TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS=1 for loopback Terraform migrations")
	}
	for _, kind := range serviceAccountTypes {
		for _, version := range []string{"3.2.1", "2.37.1"} {
			for _, scenario := range serviceAccountScenarios() {
				for _, migration := range []string{"same-type", "alias-move", "address-move"} {
					t.Run(kind.name+"/"+version+"/"+scenario.name+"/"+migration, func(t *testing.T) {
						dir := serviceAccountCLIEnvironment(t)
						api := newServiceAccountAPI(t, kind.adopt)
						sourceType, sourceLabel := kind.name, "test"
						if migration == "alias-move" {
							sourceType = kind.alias
						}
						if migration == "address-move" {
							sourceLabel = "old"
						}
						sourceAddress, targetAddress := sourceType+"."+sourceLabel, kind.name+".test"
						before := serviceAccountCLIConfig(api, sourceType, sourceLabel, scenario)
						after := serviceAccountCLIConfig(api, kind.name, "test", scenario)
						if migration != "same-type" {
							after += fmt.Sprintf("\nmoved {\n  from = %s\n  to = %s\n}\n", sourceAddress, targetAddress)
						} else if before != after {
							t.Fatal("same-type migration must use byte-identical HCL")
						}
						guards := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(), serviceAccountPlanGuard{address: targetAddress, api: api},
						}}
						name := "account"
						if kind.adopt {
							name = "default"
						}
						released := map[string]resource.ExternalProvider{"kubernetes": {
							Source: "hashicorp/kubernetes", VersionConstraint: "= " + version,
						}}
						resource.Test(t, resource.TestCase{
							IsUnitTest:             true,
							TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
							ErrorCheck:             func(err error) error { api.setPhase("cleanup"); return err },
							CheckDestroy: func(_ *terraform.State) error {
								api.mu.Lock()
								defer api.mu.Unlock()
								if api.object != nil {
									return fmt.Errorf("destroy did not delete ServiceAccount")
								}
								return nil
							},
							Steps: []resource.TestStep{
								{
									ExternalProviders: released,
									Config:            before, Check: serviceAccountReleasedVersion(dir, version),
									ExpectNonEmptyPlan: scenario.name != "non-default",
									ConfigPlanChecks:   resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{serviceAccountBaselineGuard{}}},
								},
								{
									PreConfig: func() {
										api.mu.Lock()
										defer api.mu.Unlock()
										// Model a controller reconciliation after adoption: the
										// released SDK replaces the annotations map on adoption.
										api.object.Annotations["controller.kubernetes.io/observed"] = "preserve"
										api.phase = "baseline-refresh"
									},
									ExternalProviders: released, Config: before,
									Check: func(state *terraform.State) error {
										api.mu.Lock()
										defer api.mu.Unlock()
										return serviceAccountCheckValues(state, sourceAddress, api.object, scenario)
									},
									ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{serviceAccountBaselineGuard{}}},
									ConfigStateChecks: []statecheck.StateCheck{api.checkState(sourceAddress, true)},
								},
								{
									PreConfig: func() { api.setPhase("migration") }, ProtoV6ProviderFactories: serviceAccountMuxFactories,
									Config: after, ConfigPlanChecks: guards,
									ConfigStateChecks: []statecheck.StateCheck{api.checkState(targetAddress, false), serviceAccountIdentity(targetAddress, serviceAccountCLINamespace, name)},
								},
								{
									PreConfig: func() { api.setPhase("follow-up") }, ProtoV6ProviderFactories: serviceAccountMuxFactories,
									Config: after, ConfigPlanChecks: guards,
									ConfigStateChecks: []statecheck.StateCheck{api.checkState(targetAddress, false)},
								},
								{
									PreConfig: func() { api.setPhase("cleanup") }, ProtoV6ProviderFactories: serviceAccountMuxFactories,
									Config: after, Destroy: true,
								},
							},
						})
					})
				}
			}

		}
	}
}

func serviceAccountSDKConsumer(address string) string {
	return strings.ReplaceAll(`
data "kubernetes_service_account_v1" "consumer" {
  metadata {
    name      = ACCOUNT_ADDRESS.metadata[0].name
    namespace = ACCOUNT_ADDRESS.metadata[0].namespace
  }
  depends_on = [ACCOUNT_ADDRESS]
}
output "sdk_account" {
  value = {
    uid            = data.kubernetes_service_account_v1.consumer.metadata[0].uid
    name           = data.kubernetes_service_account_v1.consumer.metadata[0].name
    namespace      = data.kubernetes_service_account_v1.consumer.metadata[0].namespace
    automount      = data.kubernetes_service_account_v1.consumer.automount_service_account_token
    secret_names   = sort([for ref in data.kubernetes_service_account_v1.consumer.secret : ref.name])
    pull_names     = sort([for ref in data.kubernetes_service_account_v1.consumer.image_pull_secret : ref.name])
    default_secret = data.kubernetes_service_account_v1.consumer.default_secret_name
  }
}
`, "ACCOUNT_ADDRESS", address)
}

func TestServiceAccountCLILegacyTokenImport(t *testing.T) {
	if os.Getenv("TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS") != "1" {
		t.Skip("set TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS=1 for loopback token import tests")
	}
	for _, kind := range serviceAccountTypes {
		for _, version := range []string{"v1.23.17", "v1.34.0"} {
			t.Run(kind.name+"/"+version, func(t *testing.T) {
				serviceAccountCLIEnvironment(t)
				api := newServiceAccountAPI(t, kind.adopt)
				api.version, api.phase = version, "import"
				name, address := "account", kind.name+".test"
				if kind.adopt {
					name = "default"
				}
				api.object = &coreapi.ServiceAccount{
					ObjectMeta:                   metav1.ObjectMeta{Name: name, Namespace: serviceAccountCLINamespace},
					AutomountServiceAccountToken: ptr.To(false),
					ImagePullSecrets:             []coreapi.LocalObjectReference{{Name: "sa-three"}},
				}
				api.decorate(api.object)
				api.tokenReference = coreapi.ObjectReference{
					Name: name + "-token-existing", Namespace: serviceAccountCLINamespace, UID: "existing-token-uid",
					APIVersion: "v1", Kind: "Secret",
				}
				api.object.Secrets = []coreapi.ObjectReference{api.tokenReference, {Name: "sa-one"}}
				api.token = &coreapi.Secret{
					TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
					ObjectMeta: metav1.ObjectMeta{
						Name: api.tokenReference.Name, Namespace: serviceAccountCLINamespace, UID: api.tokenReference.UID,
						Annotations: map[string]string{
							coreapi.ServiceAccountNameKey: name, coreapi.ServiceAccountUIDKey: string(api.object.UID),
						},
					},
					Type: coreapi.SecretTypeServiceAccountToken,
				}
				uid, token := api.object.UID, api.token.DeepCopy()
				initial := serviceAccountScenario{automount: "false", secrets: []string{"one"}, pullSecrets: []string{"three"}}
				removed := serviceAccountScenario{automount: "false"}
				config := serviceAccountCLIConfig(api, kind.name, "test", initial)
				after := serviceAccountCLIConfig(api, kind.name, "test", removed)
				check := func(scenario serviceAccountScenario) resource.TestCheckFunc {
					return func(state *terraform.State) error {
						api.mu.Lock()
						defer api.mu.Unlock()
						if api.object == nil || api.object.UID != uid || !reflect.DeepEqual(api.token, token) ||
							!slices.Contains(api.object.Secrets, api.tokenReference) {
							return fmt.Errorf("import/update changed the account, token, or complete token reference")
						}
						tokenPath := "GET /api/v1/namespaces/" + serviceAccountCLINamespace + "/secrets/" + token.Name
						if api.calls["import"][tokenPath] == 0 {
							return fmt.Errorf("import never inspected the existing token secret")
						}
						if err := resource.TestCheckResourceAttr(address, "default_secret_name", token.Name)(state); err != nil {
							return err
						}
						return serviceAccountCheckValues(state, address, api.object, scenario)
					}
				}
				resource.Test(t, resource.TestCase{
					IsUnitTest: true, ProtoV6ProviderFactories: serviceAccountMuxFactories,
					TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
					ErrorCheck:             func(err error) error { api.setPhase("cleanup"); return err }, CheckDestroy: api.checkDestroy,
					Steps: []resource.TestStep{
						{
							Config: config + fmt.Sprintf("\nimport {\n  to = %s\n  id = %q\n}\n", address, serviceAccountCLINamespace+"/"+name),
							Check:  check(initial),
							ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(address, plancheck.ResourceActionNoop),
							}},
							ConfigStateChecks: []statecheck.StateCheck{serviceAccountIdentity(address, serviceAccountCLINamespace, name)},
						},
						{
							Config: config, Check: check(initial),
							ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
						},
						{ResourceName: address, ImportState: true, ImportStateVerify: true},
						{ResourceName: address, ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity},
						{
							PreConfig: func() { api.setPhase("update") }, Config: after, Check: check(removed),
							ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
							}},
						},
						{
							PreConfig:    func() { api.setPhase("import") },
							ResourceName: address, ImportState: true, ImportStateVerify: true,
						},
						{
							PreConfig: func() { api.setPhase("follow-up") }, Config: after, Check: check(removed),
							ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
						},
						{PreConfig: func() { api.setPhase("cleanup") }, Config: after, Destroy: true},
					},
				})
			})
		}
	}
}

func serviceAccountCheckSDKConsumer(state *terraform.State, address string, object *coreapi.ServiceAccount) error {
	const consumer = "data.kubernetes_service_account_v1.consumer"
	for _, field := range []string{"id", "metadata.0.uid", "metadata.0.name", "metadata.0.namespace", "automount_service_account_token", "default_secret_name"} {
		if err := resource.TestCheckResourceAttrPair(consumer, field, address, field)(state); err != nil {
			return err
		}
	}
	data := state.RootModule().Resources[consumer]
	token := data.Primary.Attributes["default_secret_name"]
	for field, names := range map[string][]string{
		"secret": serviceAccountSecretNames(object.Secrets, token), "image_pull_secret": serviceAccountPullNames(object.ImagePullSecrets),
	} {
		if err := resource.TestCheckResourceAttr(consumer, field+".#", strconv.Itoa(len(names)))(state); err != nil {
			return err
		}
		for i, name := range names {
			if err := resource.TestCheckResourceAttr(consumer, fmt.Sprintf("%s.%d.name", field, i), name)(state); err != nil {
				return err
			}
		}
	}
	output := state.RootModule().Outputs["sdk_account"]
	if output == nil {
		return fmt.Errorf("SDK consumer output missing")
	}
	secrets, pulls := serviceAccountSecretNames(object.Secrets, token), serviceAccountPullNames(object.ImagePullSecrets)
	slices.Sort(secrets)
	slices.Sort(pulls)
	want, _ := json.Marshal(map[string]any{
		"uid": string(object.UID), "name": object.Name, "namespace": object.Namespace,
		"automount":    object.AutomountServiceAccountToken != nil && *object.AutomountServiceAccountToken,
		"secret_names": secrets, "pull_names": pulls, "default_secret": token,
	})
	got, _ := json.Marshal(output.Value)
	if string(got) != string(want) {
		return fmt.Errorf("SDK output differs from versioned Framework resource/API: got %s, want %s", got, want)
	}
	return nil
}

func TestServiceAccountCLISDKDataSource(t *testing.T) {
	if os.Getenv("TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS") != "1" {
		t.Skip("set TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS=1 for loopback SDK consumer tests")
	}
	for _, kind := range serviceAccountTypes {
		t.Run(kind.name, func(t *testing.T) {
			serviceAccountCLIEnvironment(t)
			api := newServiceAccountAPI(t, kind.adopt)
			scenario := serviceAccountScenarios()[1]
			config := serviceAccountCLIConfig(api, kind.name, "test", scenario)
			address := kind.name + ".test"
			withConsumer := config + serviceAccountSDKConsumer(address)
			check := func(state *terraform.State) error {
				api.mu.Lock()
				defer api.mu.Unlock()
				if !reflect.DeepEqual(api.object, api.baseline) {
					return fmt.Errorf("SDK consumer changed the Framework-managed API object")
				}
				return serviceAccountCheckSDKConsumer(state, address, api.object)
			}
			resource.Test(t, resource.TestCase{
				IsUnitTest: true, ProtoV6ProviderFactories: serviceAccountMuxFactories,
				ErrorCheck:   func(err error) error { api.setPhase("cleanup"); return err },
				CheckDestroy: api.checkDestroy,
				Steps: []resource.TestStep{
					{Config: config, ConfigStateChecks: []statecheck.StateCheck{api.checkState(address, true)}},
					{
						PreConfig: func() { api.setPhase("consumer") }, Config: withConsumer,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionNoop),
						}},
						Check: check,
					},
					{
						Config: withConsumer, Check: check,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
					},
					{PreConfig: func() { api.setPhase("cleanup") }, Config: config, Destroy: true},
				},
			})
		})
	}
}

// Both SDK releases leave controller references untouched during adoption but
// subsequently propose removing omitted blocks. Characterize that plan without
// applying it; the API fixture independently rejects any reference mutation.
type serviceAccountAdoptionPlanGuard struct{}

func (serviceAccountAdoptionPlanGuard) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if len(req.Plan.ResourceChanges) != 1 {
		resp.Error = fmt.Errorf("expected exactly the default ServiceAccount")
		return
	}
	change := req.Plan.ResourceChanges[0]
	if change.Address != "kubernetes_default_service_account_v1.test" || !reflect.DeepEqual(change.Change.Actions, tfjson.Actions{tfjson.ActionUpdate}) {
		resp.Error = fmt.Errorf("expected the known SDK reference-removal update, got %+v", change)
		return
	}
	before, beforeOK := change.Change.Before.(map[string]any)
	after, afterOK := change.Change.After.(map[string]any)
	if !beforeOK || !afterOK || before["id"] != serviceAccountCLINamespace+"/default" || after["id"] != before["id"] ||
		before["automount_service_account_token"] != true || after["automount_service_account_token"] != true {
		resp.Error = fmt.Errorf("adoption follow-up changed identity or automount")
		return
	}
	for _, field := range []string{"secret", "image_pull_secret"} {
		old, oldOK := before[field].([]any)
		new, newOK := after[field].([]any)
		if !oldOK || len(old) != 1 || !newOK || len(new) != 0 {
			resp.Error = fmt.Errorf("expected precisely one existing %s proposed for removal", field)
		}
	}
}

func TestServiceAccountCLIDefaultAdoptionReferences(t *testing.T) {
	if os.Getenv("TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS") != "1" {
		t.Skip("set TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS=1 for loopback adoption probes")
	}
	for _, version := range []string{"3.2.1", "2.37.1", "local"} {
		t.Run(version, func(t *testing.T) {
			dir := serviceAccountCLIEnvironment(t)
			api := newServiceAccountAPI(t, true)
			api.protectReferences = true
			api.object.Secrets = []coreapi.ObjectReference{{Name: "controller-secret", Namespace: serviceAccountCLINamespace, UID: "secret-owner"}}
			api.object.ImagePullSecrets = []coreapi.LocalObjectReference{{Name: "controller-pull"}}
			original := api.object.DeepCopy()
			config := fmt.Sprintf("provider \"kubernetes\" {\n  host = %q\n}\n", api.server.URL) +
				serviceAccountConfig("kubernetes_default_service_account_v1", "test", strconv.Quote(serviceAccountCLINamespace), "default", serviceAccountScenario{}, false)
			check := func(_ *terraform.State) error {
				api.mu.Lock()
				defer api.mu.Unlock()
				if api.object == nil || api.object.UID != original.UID ||
					!reflect.DeepEqual(api.object.Secrets, original.Secrets) || !reflect.DeepEqual(api.object.ImagePullSecrets, original.ImagePullSecrets) {
					return fmt.Errorf("adoption removed/replaced preexisting controller references")
				}
				return nil
			}
			steps := []resource.TestStep{
				{
					Config: config, ExpectNonEmptyPlan: true, Check: check,
					ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{serviceAccountAdoptionPlanGuard{}}},
				},
				{
					PreConfig: func() { api.setPhase("adoption-plan-only") }, Config: config, PlanOnly: true, ExpectNonEmptyPlan: true,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						PostApplyPostRefresh: []plancheck.PlanCheck{serviceAccountAdoptionPlanGuard{}},
					},
				},
				{PreConfig: func() { api.setPhase("cleanup") }, Config: config, Destroy: true},
			}
			for i := range steps {
				if version == "local" {
					steps[i].ProtoV6ProviderFactories = serviceAccountMuxFactories
				} else {
					steps[i].ExternalProviders = map[string]resource.ExternalProvider{"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= " + version}}
				}
			}
			if version != "local" {
				steps[0].Check = resource.ComposeAggregateTestCheckFunc(serviceAccountReleasedVersion(dir, version), check)
			}
			resource.Test(t, resource.TestCase{
				IsUnitTest: true, Steps: steps,
				ErrorCheck:   func(err error) error { api.setPhase("cleanup"); return err },
				CheckDestroy: api.checkDestroy,
			})
		})
	}
}

func TestServiceAccountCLILifecycle(t *testing.T) {
	if os.Getenv("TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS") != "1" {
		t.Skip("set TF_KUBERNETES_SERVICE_ACCOUNT_CLI_TESTS=1 for loopback Terraform lifecycle tests")
	}
	for _, kind := range serviceAccountTypes {
		for _, scenario := range serviceAccountScenarios()[:2] {
			t.Run(kind.name+"/"+scenario.name, func(t *testing.T) {
				serviceAccountCLIEnvironment(t)
				api := newServiceAccountAPI(t, kind.adopt)
				// Timeouts are configuration-only and cannot be verified on import.
				// Cover their exact state separately in the released migration matrix.
				scenario.timeout = false
				config := serviceAccountCLIConfig(api, kind.name, "test", scenario)
				address, name := kind.name+".test", "account"
				if kind.adopt {
					name = "default"
				}
				resource.Test(t, resource.TestCase{
					IsUnitTest: true, ProtoV6ProviderFactories: serviceAccountMuxFactories,
					TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
					ErrorCheck:             func(err error) error { api.setPhase("cleanup"); return err },
					CheckDestroy: func(_ *terraform.State) error {
						api.mu.Lock()
						defer api.mu.Unlock()
						if api.object != nil {
							return fmt.Errorf("local Delete did not remove ServiceAccount")
						}
						return nil
					},
					Steps: []resource.TestStep{
						{Config: config, ConfigStateChecks: []statecheck.StateCheck{
							api.checkState(address, true), serviceAccountIdentity(address, serviceAccountCLINamespace, name),
						}, Check: func(state *terraform.State) error {
							api.mu.Lock()
							defer api.mu.Unlock()
							return serviceAccountCheckValues(state, address, api.object, scenario)
						}},
						{
							PreConfig:    func() { api.setPhase("import") },
							ResourceName: address, ImportState: true, ImportStateVerify: true,
						},
						{
							ResourceName: address, ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity,
						},
						{
							PreConfig: func() { api.setPhase("follow-up") }, Config: config,
							ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(), serviceAccountPlanGuard{address: address, api: api},
							}},
							ConfigStateChecks: []statecheck.StateCheck{api.checkState(address, false)},
						},
						{PreConfig: func() { api.setPhase("cleanup") }, Config: config, Destroy: true},
					},
				})
			})
		}
	}
}
