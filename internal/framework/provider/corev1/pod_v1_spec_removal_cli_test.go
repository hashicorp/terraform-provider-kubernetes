// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func TestPodV1SpecRemovalReplacementSemanticsCore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		initialSpec string
		updatedSpec string
		wantSDK     plancheck.ResourceActionType
		wantNative  plancheck.ResourceActionType
	}{
		{
			name: "oneof2regularcontainers",
			initialSpec: `
    container {
      name  = "app"
      image = "image:1"
    }
    container {
      name  = "sidecar"
      image = "sidecar:1"
    }`,
			updatedSpec: `
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionReplace,
			wantNative: plancheck.ResourceActionReplace,
		},
		{
			name: "volume_mount",
			initialSpec: `
    container {
      name  = "app"
      image = "image:1"
      volume_mount {
        name       = "data"
        mount_path = "/data"
      }
    }`,
			updatedSpec: `
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionReplace,
			wantNative: plancheck.ResourceActionReplace,
		},
		{
			name: "configuredvolume",
			initialSpec: `
    volume {
      name = "data"
      empty_dir {}
    }
    container {
      name  = "app"
      image = "image:1"
    }`,
			updatedSpec: `
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionUpdate,
			wantNative: plancheck.ResourceActionUpdate,
		},
		{
			name: "configuredvolume_immutable_medium",
			initialSpec: `
    volume {
      name = "data"
      empty_dir {
        medium = "Memory"
      }
    }
    container {
      name  = "app"
      image = "image:1"
    }`,
			updatedSpec: `
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionReplace,
			wantNative: plancheck.ResourceActionReplace,
		},
		{
			name: "configuredvolume_append_empty_dir_with_existing_secret_configmap",
			initialSpec: `
    volume {
      name = "cfg"
      config_map {
        name = "app-config"
      }
    }
    volume {
      name = "sec"
      secret {
        secret_name = "app-secret"
      }
    }
    container {
      name  = "app"
      image = "image:1"
    }`,
			updatedSpec: `
    volume {
      name = "cfg"
      config_map {
        name = "app-config"
      }
    }
    volume {
      name = "sec"
      secret {
        secret_name = "app-secret"
      }
    }
    volume {
      name = "data"
      empty_dir {}
    }
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionUpdate,
			wantNative: plancheck.ResourceActionUpdate,
		},
		{
			name: "envblock",
			initialSpec: `
    container {
      name  = "app"
      image = "image:1"
      env {
        name  = "MODE"
        value = "prod"
      }
    }`,
			updatedSpec: `
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionReplace,
			wantNative: plancheck.ResourceActionReplace,
		},
		{
			name: "env_fromblock",
			initialSpec: `
    container {
      name  = "app"
      image = "image:1"
      env_from {
        config_map_ref {
          name = "cfg"
        }
      }
    }`,
			updatedSpec: `
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionReplace,
			wantNative: plancheck.ResourceActionReplace,
		},
		{
			name: "containersecuritycontext",
			initialSpec: `
    container {
      name  = "app"
      image = "image:1"
      security_context {
        run_as_user = "1000"
      }
    }`,
			updatedSpec: `
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionReplace,
			wantNative: plancheck.ResourceActionReplace,
		},
		{
			name: "podsecuritycontext",
			initialSpec: `
    security_context {
      run_as_user = "1000"
    }
    container {
      name  = "app"
      image = "image:1"
    }`,
			updatedSpec: `
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionReplace,
			wantNative: plancheck.ResourceActionReplace,
		},
		{
			name: "podsecuritycontext_field",
			initialSpec: `
    security_context {
      run_as_user = "1000"
    }
    container {
      name  = "app"
      image = "image:1"
    }`,
			updatedSpec: `
    security_context {}
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionReplace,
			wantNative: plancheck.ResourceActionReplace,
		},
		{
			name: "init_container_remove_count",
			initialSpec: `
    init_container {
      name  = "init1"
      image = "init:1"
    }
    init_container {
      name  = "init2"
      image = "init:2"
    }
    container {
      name  = "app"
      image = "image:1"
    }`,
			updatedSpec: `
    init_container {
      name  = "init1"
      image = "init:1"
    }
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionReplace,
			wantNative: plancheck.ResourceActionReplace,
		},
		{
			name: "active_deadline_seconds_60_to_30",
			initialSpec: `
    active_deadline_seconds = 60
    container {
      name  = "app"
      image = "image:1"
    }`,
			updatedSpec: `
    active_deadline_seconds = 30
    container {
      name  = "app"
      image = "image:1"
    }`,
			wantSDK:    plancheck.ResourceActionUpdate,
			wantNative: plancheck.ResourceActionUpdate,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sdkAction := podSpecObservedAction(t,
				podSpecLegacyFactories(),
				"podlegacy_pod.test",
				podSpecLegacyConfig(tc.initialSpec),
				podSpecLegacyConfig(tc.updatedSpec),
			)
			nativeAction := podSpecObservedAction(t,
				map[string]func() (tfprotov6.ProviderServer, error){
					"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
				},
				"podquantity_native.test",
				podSpecNativeConfig(tc.initialSpec),
				podSpecNativeConfig(tc.updatedSpec),
			)

			t.Logf("observed actions: sdk=%s native=%s", sdkAction, nativeAction)

			if !podSpecActionMatches(tc.wantSDK, sdkAction) {
				t.Fatalf("sdk action mismatch: want %s, got %s", tc.wantSDK, sdkAction)
			}
			if !podSpecActionMatches(tc.wantNative, nativeAction) {
				t.Fatalf("native action mismatch: want %s, got %s", tc.wantNative, nativeAction)
			}
		})
	}
}

func podSpecObservedAction(
	t *testing.T,
	factories map[string]func() (tfprotov6.ProviderServer, error),
	resourceAddress, initialConfig, updatedConfig string,
) plancheck.ResourceActionType {
	t.Helper()

	var action plancheck.ResourceActionType
	check := podSpecCaptureResourceAction{resourceAddress: resourceAddress, action: &action, t: t}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []testresource.TestStep{
			{Config: initialConfig},
			{
				Config: updatedConfig,
				ConfigPlanChecks: testresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					check,
				}},
			},
		},
	})

	return action
}

func podSpecLegacyFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"podlegacy": func() (tfprotov6.ProviderServer, error) {
			provider := &sdkschema.Provider{ResourcesMap: map[string]*sdkschema.Resource{
				"podlegacy_pod": podSpecLegacyResource(),
			}}
			return tf5to6server.UpgradeServer(context.Background(), provider.GRPCProvider)
		},
	}
}

func podSpecLegacyResource() *sdkschema.Resource {
	legacyPod := kubernetes.Provider().ResourcesMap["kubernetes_pod"]
	return &sdkschema.Resource{
		SchemaVersion: legacyPod.SchemaVersion,
		Schema:        legacyPod.Schema,
		Timeouts:      legacyPod.Timeouts,
		CreateContext: func(ctx context.Context, data *sdkschema.ResourceData, _ interface{}) sdkdiag.Diagnostics {
			data.SetId("test")
			_ = ctx
			return nil
		},
		ReadContext: func(context.Context, *sdkschema.ResourceData, interface{}) sdkdiag.Diagnostics {
			return nil
		},
		UpdateContext: func(context.Context, *sdkschema.ResourceData, interface{}) sdkdiag.Diagnostics {
			return nil
		},
		DeleteContext: func(_ context.Context, data *sdkschema.ResourceData, _ interface{}) sdkdiag.Diagnostics {
			data.SetId("")
			return nil
		},
	}
}

func podSpecLegacyConfig(spec string) string {
	return fmt.Sprintf(`resource "podlegacy_pod" "test" {
  metadata {
    name = "test"
  }
  spec {
%s
  }
}`, spec)
}

func podSpecNativeConfig(spec string) string {
	return fmt.Sprintf(`resource "podquantity_native" "test" {
  spec {
%s
  }
}`, spec)
}

type podSpecCaptureResourceAction struct {
	resourceAddress string
	action          *plancheck.ResourceActionType
	t               *testing.T
}

func (c podSpecCaptureResourceAction) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, rc := range req.Plan.ResourceChanges {
		if rc.Address != c.resourceAddress {
			continue
		}
		action, err := podSpecActionType(rc.Change.Actions)
		if err != nil {
			resp.Error = fmt.Errorf("%s - %w", c.resourceAddress, err)
			return
		}
		*c.action = action
		if c.t != nil {
			c.t.Logf("%s replacement paths: %v", c.resourceAddress, rc.Change.ReplacePaths)
		}
		return
	}
	resp.Error = fmt.Errorf("%s - Resource not found in plan ResourceChanges", c.resourceAddress)
}

func podSpecActionType(actions tfjson.Actions) (plancheck.ResourceActionType, error) {
	switch {
	case actions.NoOp():
		return plancheck.ResourceActionNoop, nil
	case actions.Create():
		return plancheck.ResourceActionCreate, nil
	case actions.Read():
		return plancheck.ResourceActionRead, nil
	case actions.Update():
		return plancheck.ResourceActionUpdate, nil
	case actions.DestroyBeforeCreate():
		return plancheck.ResourceActionDestroyBeforeCreate, nil
	case actions.CreateBeforeDestroy():
		return plancheck.ResourceActionCreateBeforeDestroy, nil
	case actions.Delete():
		return plancheck.ResourceActionDestroy, nil
	case actions.Replace():
		return plancheck.ResourceActionReplace, nil
	default:
		return "", fmt.Errorf("unexpected action(s): %v", actions)
	}
}

func podSpecActionMatches(want, got plancheck.ResourceActionType) bool {
	if want == got {
		return true
	}
	if want != plancheck.ResourceActionReplace {
		return false
	}
	return got == plancheck.ResourceActionDestroyBeforeCreate || got == plancheck.ResourceActionCreateBeforeDestroy || got == plancheck.ResourceActionReplace
}
