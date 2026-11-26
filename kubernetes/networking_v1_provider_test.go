// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func init() {
	kubernetes.NetworkingV1MuxFactory = mux.MuxServerWithProvider
}

// The retained SDKv2 constructors are test-only exports: this checks the exact
// original ingress schema rather than approximating it with its data source.
func TestNetworkingV1SDKSchemaAndIdentityEquivalence(t *testing.T) {
	ctx := context.Background()
	baseline, err := tf5to6server.UpgradeServer(ctx, kubernetes.NetworkingV1SDKProviderForTest().GRPCProvider)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	oldSchemas, err := baseline.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	newSchemas, err := actual.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	oldIdentities, err := baseline.GetResourceIdentitySchemas(ctx, &tfprotov6.GetResourceIdentitySchemasRequest{})
	if err != nil {
		t.Fatal(err)
	}
	newIdentities, err := actual.GetResourceIdentitySchemas(ctx, &tfprotov6.GetResourceIdentitySchemasRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostics := range [][]*tfprotov6.Diagnostic{
		oldSchemas.Diagnostics, newSchemas.Diagnostics, oldIdentities.Diagnostics, newIdentities.Diagnostics,
	} {
		for _, diagnostic := range diagnostics {
			if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
				t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
			}
		}
	}
	for _, name := range []string{
		"kubernetes_ingress_v1", "kubernetes_ingress_class_v1", "kubernetes_network_policy_v1",
	} {
		t.Run(name, func(t *testing.T) {
			oldSchema, newSchema := oldSchemas.ResourceSchemas[name], newSchemas.ResourceSchemas[name]
			if oldSchema == nil || newSchema == nil {
				t.Fatal("resource schema missing")
			}
			if oldSchema.Version != 0 || newSchema.Version != 1 {
				t.Fatalf("schema version changed: SDKv2=%d Framework=%d", oldSchema.Version, newSchema.Version)
			}
			if expected := networkingObjectStateType(t, name, oldSchema.ValueType()); !expected.Equal(newSchema.ValueType()) {
				t.Fatalf("state changed beyond the selected singleton conversions:\nexpected: %s\nFramework: %s", expected, newSchema.ValueType())
			}
			oldIdentity, newIdentity := oldIdentities.IdentitySchemas[name], newIdentities.IdentitySchemas[name]
			if oldIdentity == nil || newIdentity == nil {
				t.Fatal("resource identity schema missing")
			}
			if oldIdentity.Version != 1 || newIdentity.Version != oldIdentity.Version {
				t.Fatalf("identity version changed: SDKv2=%d Framework=%d", oldIdentity.Version, newIdentity.Version)
			}
			if !oldIdentity.ValueType().Equal(newIdentity.ValueType()) {
				t.Fatalf("identity type changed:\nSDKv2: %s\nFramework: %s", oldIdentity.ValueType(), newIdentity.ValueType())
			}
			if len(oldIdentity.IdentityAttributes) != len(newIdentity.IdentityAttributes) {
				t.Fatal("identity attribute count changed")
			}
			for _, oldAttribute := range oldIdentity.IdentityAttributes {
				found := false
				for _, newAttribute := range newIdentity.IdentityAttributes {
					if oldAttribute.Name == newAttribute.Name {
						found = true
						if oldAttribute.RequiredForImport != newAttribute.RequiredForImport ||
							oldAttribute.OptionalForImport != newAttribute.OptionalForImport {
							t.Errorf("identity import contract changed for %s", oldAttribute.Name)
						}
					}
				}
				if !found {
					t.Errorf("identity attribute %s missing", oldAttribute.Name)
				}
			}
		})
	}
}

// Compare every SDK field after applying only the explicitly selected object paths.
func networkingObjectStateType(t *testing.T, resource string, sdk tftypes.Type) tftypes.Type {
	t.Helper()
	objects := map[string]bool{}
	switch resource {
	case "kubernetes_ingress_v1":
		objects["spec.*.rule.*.http"] = true
		for _, base := range []string{"spec.*.default_backend", "spec.*.rule.*.http.path.*.backend"} {
			for _, suffix := range []string{"", ".resource", ".service", ".service.port"} {
				objects[base+suffix] = true
			}
		}
	case "kubernetes_ingress_class_v1":
		objects["spec.*.parameters"] = true
	case "kubernetes_network_policy_v1":
		objects["spec.*.pod_selector"] = true
		for _, peer := range []string{"spec.*.ingress.*.from.*", "spec.*.egress.*.to.*"} {
			for _, field := range []string{"ip_block", "namespace_selector", "pod_selector"} {
				objects[peer+"."+field] = true
			}
		}
	default:
		t.Fatalf("unexpected resource %s", resource)
	}
	var convert func(tftypes.Type, string) tftypes.Type
	convert = func(typ tftypes.Type, at string) tftypes.Type {
		if objects[at] {
			list, ok := typ.(tftypes.List)
			if !ok {
				t.Fatalf("SDK singleton %s is %T, want list", at, typ)
			}
			typ = list.ElementType
			delete(objects, at)
		}
		switch typ := typ.(type) {
		case tftypes.Object:
			fields := make(map[string]tftypes.Type, len(typ.AttributeTypes))
			for name, child := range typ.AttributeTypes {
				next := name
				if at != "" {
					next = at + "." + name
				}
				fields[name] = convert(child, next)
			}
			return tftypes.Object{AttributeTypes: fields}
		case tftypes.List:
			return tftypes.List{ElementType: convert(typ.ElementType, at+".*")}
		case tftypes.Set:
			return tftypes.Set{ElementType: convert(typ.ElementType, at+".*")}
		default:
			return typ
		}
	}
	result := convert(sdk, "")
	if len(objects) != 0 {
		t.Fatalf("unvisited singleton paths: %v", objects)
	}
	return result
}
