// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	api "k8s.io/api/core/v1"
	kquantity "k8s.io/apimachinery/pkg/api/resource"
)

const podSpecDefaultedPlanConfig = `
resource "kubernetes_pod_v1" "test" {
  metadata {
    name      = "example"
    namespace = "test"
    labels    = { app = "original" }
  }
  spec {
    container {
      name  = "app"
      image = "busybox:1.36"
    }
    init_container {
      name  = "setup"
      image = "busybox:1.36"
    }
  }
}`

func TestPodV1SpecDefaultedMetadataPlanCore(t *testing.T) {
	for _, guarded := range []bool{false, true} {
		t.Run(fmt.Sprintf("guarded_%t", guarded), func(t *testing.T) {
			p := podSpecDefaultedHTTPResource(t)
			var native resource.Resource = p
			if !guarded {
				native = &podSpecUnguardedDefaultTestResource{PodV1: p}
			}
			action := plancheck.ResourceActionUpdate
			var replacementCheck plancheck.PlanCheck = podSpecNoReplacePaths{}
			if !guarded {
				action = plancheck.ResourceActionDestroyBeforeCreate
				replacementCheck = podSpecOldDefaultedReplacePaths{}
			}
			updated := strings.Replace(podSpecDefaultedPlanConfig, `app = "original"`, `app = "updated"`, 1)
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"kubernetes": providerserver.NewProtocol6WithError(podSpecQuantityNativeProvider{resource: native}),
				},
				Steps: []testresource.TestStep{
					{Config: podSpecDefaultedPlanConfig},
					{
						Config: updated, PlanOnly: true, ExpectNonEmptyPlan: true,
						ConfigPlanChecks: testresource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("kubernetes_pod_v1.test", action),
							replacementCheck,
						}},
					},
				},
			})
		})
	}
}

func TestPodV1SpecDefaultedUpdateCore(t *testing.T) {
	for name, updated := range map[string]string{
		"metadata labels": strings.Replace(podSpecDefaultedPlanConfig, `app = "original"`, `app = "updated"`, 1),
		"target state":    strings.Replace(podSpecDefaultedPlanConfig, `  spec {`, "  target_state = [\"Running\"]\n  spec {", 1),
	} {
		t.Run(name, func(t *testing.T) {
			p := podSpecDefaultedHTTPResource(t)
			check := testresource.ComposeTestCheckFunc(
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "metadata.0.uid", "quantity-uid"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.node_name", "worker"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.service_account_name", "default"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.image_pull_secrets.0.name", "injected"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.readiness_gate.0.condition_type", "example.com/Ready"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.container.0.resources.0.requests.cpu", "1"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.init_container.0.resources.0.requests.cpu", "1"),
			)
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"kubernetes": providerserver.NewProtocol6WithError(podSpecQuantityNativeProvider{resource: p}),
				},
				Steps: []testresource.TestStep{
					{Config: podSpecDefaultedPlanConfig, Check: check},
					{
						Config: updated,
						ConfigPlanChecks: testresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("kubernetes_pod_v1.test", plancheck.ResourceActionUpdate),
							podSpecNoReplacePaths{},
						}},
						Check: check,
					},
					{Config: updated, PlanOnly: true},
				},
			})
		})
	}
}

func TestPodV1SpecAdmissionDefaultsMetadataUpdateCore(t *testing.T) {
	for _, test := range []struct {
		name, resources, cpu, memory string
	}{
		{"omitted resources", "", "1", "128Mi"},
		{"null resources", "resources = null", "1", "128Mi"},
		{"empty resources object", "resources = [{}]", "1", "128Mi"},
		{"omitted requests", `resources = [{ limits = { cpu = "2", memory = "512Mi" } }]`, "2", "512Mi"},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := podSpecDefaultedHTTPResource(t)
			config := strings.ReplaceAll(podSpecDefaultedPlanConfig, `image = "busybox:1.36"`,
				`image = "busybox:1.36"`+"\n      "+test.resources)
			updated := strings.Replace(config, `app = "original"`, `app = "updated"`, 1)
			checks := []testresource.TestCheckFunc{
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "metadata.0.uid", "quantity-uid"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.service_account_name", "default"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.image_pull_secrets.0.name", "injected"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.readiness_gate.0.condition_type", "example.com/Ready"),
				testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.volume.#", "0"),
			}
			for _, container := range []string{"container", "init_container"} {
				prefix := "spec.0." + container + ".0."
				checks = append(checks,
					testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", prefix+"resources.0.limits.cpu", test.cpu),
					testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", prefix+"resources.0.limits.memory", test.memory),
					testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", prefix+"resources.0.requests.cpu", "1"),
					testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", prefix+"resources.0.requests.memory", "64Mi"),
					testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", prefix+"volume_mount.#", "0"),
				)
			}
			check := testresource.ComposeTestCheckFunc(checks...)
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"kubernetes": providerserver.NewProtocol6WithError(podSpecQuantityNativeProvider{resource: p}),
				},
				Steps: []testresource.TestStep{
					{Config: config, Check: check},
					{
						Config: updated,
						ConfigPlanChecks: testresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("kubernetes_pod_v1.test", plancheck.ResourceActionUpdate),
							podSpecNoReplacePaths{},
						}},
						Check: testresource.ComposeTestCheckFunc(check,
							testresource.TestCheckResourceAttr("kubernetes_pod_v1.test", "metadata.0.labels.app", "updated")),
					},
					{Config: updated, PlanOnly: true},
				},
			})
		})
	}
}

func TestPodV1SpecDefaultedIdentityImportCore(t *testing.T) {
	p := podSpecDefaultedHTTPResource(t)
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"kubernetes": providerserver.NewProtocol6WithError(podSpecQuantityNativeProvider{resource: p}),
		},
		Steps: []testresource.TestStep{
			{Config: podSpecDefaultedPlanConfig},
			{
				ResourceName: "kubernetes_pod_v1.test", ImportState: true,
				ImportStateKind: testresource.ImportBlockWithResourceIdentity,
				ImportPlanChecks: testresource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("kubernetes_pod_v1.test", plancheck.ResourceActionNoop),
					podSpecNoReplacePaths{},
				}},
			},
		},
	})
}

type podSpecNoReplacePaths struct{}

func (podSpecNoReplacePaths) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, change := range req.Plan.ResourceChanges {
		if change.Address == "kubernetes_pod_v1.test" {
			if len(change.Change.ReplacePaths) != 0 {
				resp.Error = fmt.Errorf("unexpected Pod replacement paths: %v", change.Change.ReplacePaths)
			}
			return
		}
	}
	resp.Error = fmt.Errorf("Pod not found in plan")
}

type podSpecOldDefaultedReplacePaths struct{}

func (podSpecOldDefaultedReplacePaths) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	expected := map[string]bool{
		"[spec 0 hostname]":                                    true,
		"[spec 0 node_name]":                                   true,
		"[spec 0 scheduler_name]":                              true,
		"[spec 0 service_account_name]":                        true,
		"[spec 0 image_pull_secrets]":                          true,
		"[spec 0 readiness_gate]":                              true,
		"[spec 0 container 0 image_pull_policy]":               true,
		"[spec 0 container 0 termination_message_policy]":      true,
		"[spec 0 container 0 restart_policy]":                  true,
		"[spec 0 init_container 0 image_pull_policy]":          true,
		"[spec 0 init_container 0 termination_message_policy]": true,
		"[spec 0 init_container 0 restart_policy]":             true,
	}
	for _, change := range req.Plan.ResourceChanges {
		if change.Address != "kubernetes_pod_v1.test" {
			continue
		}
		for _, path := range change.Change.ReplacePaths {
			key := fmt.Sprint(path)
			if !expected[key] {
				resp.Error = fmt.Errorf("unexpected old-schema replacement path: %s", key)
				return
			}
			delete(expected, key)
		}
		if len(expected) != 0 {
			resp.Error = fmt.Errorf("missing old-schema replacement paths: %v", expected)
		}
		return
	}
	resp.Error = fmt.Errorf("Pod not found in plan")
}

func podSpecDefaultedHTTPResource(t *testing.T) *PodV1 {
	t.Helper()
	return podSpecQuantityHTTPResourceWithDefaults(t, func(pod *api.Pod) {
		pod.Spec.NodeName = "worker"
		pod.Spec.Hostname = "api-hostname"
		pod.Spec.SchedulerName = "default-scheduler"
		pod.Spec.ServiceAccountName = "default"
		pod.Spec.ImagePullSecrets = []api.LocalObjectReference{{Name: "injected"}}
		pod.Spec.ReadinessGates = []api.PodReadinessGate{{ConditionType: "example.com/Ready"}}
		pod.Spec.Volumes = append(pod.Spec.Volumes, api.Volume{
			Name: "kube-api-access-abc12",
			VolumeSource: api.VolumeSource{Projected: &api.ProjectedVolumeSource{
				Sources: []api.VolumeProjection{{
					ServiceAccountToken: &api.ServiceAccountTokenProjection{Path: "token"},
				}},
			}},
		})
		for _, containers := range [][]api.Container{pod.Spec.Containers, pod.Spec.InitContainers} {
			for i := range containers {
				if containers[i].Resources.Limits == nil {
					containers[i].Resources.Limits = api.ResourceList{
						api.ResourceCPU:    kquantity.MustParse("1"),
						api.ResourceMemory: kquantity.MustParse("128Mi"),
					}
				}
				if containers[i].Resources.Requests == nil {
					containers[i].Resources.Requests = api.ResourceList{
						api.ResourceCPU:    kquantity.MustParse("1"),
						api.ResourceMemory: kquantity.MustParse("64Mi"),
					}
				}
				containers[i].VolumeMounts = append(containers[i].VolumeMounts, api.VolumeMount{
					Name:      "kube-api-access-abc12",
					MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
					ReadOnly:  true,
				})
			}
		}
	})
}

// Reproduce the old schema's computed-to-unknown replacement before the guards.
type podSpecUnguardedDefaultTestResource struct{ *PodV1 }

func (p *podSpecUnguardedDefaultTestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	p.PodV1.Schema(ctx, req, resp)
	spec := resp.Schema.Blocks["spec"].(schema.ListNestedBlock)
	for name, attribute := range spec.NestedObject.Attributes {
		switch attribute := attribute.(type) {
		case schema.StringAttribute:
			if attribute.Computed && attribute.Default == nil {
				attribute.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
				spec.NestedObject.Attributes[name] = attribute
			}
		case schema.ListAttribute:
			if attribute.Computed {
				attribute.PlanModifiers = []planmodifier.List{listplanmodifier.RequiresReplace()}
				spec.NestedObject.Attributes[name] = attribute
			}
		}
	}
	for _, name := range []string{"container", "init_container"} {
		container := spec.NestedObject.Blocks[name].(schema.ListNestedBlock)
		for name, attribute := range container.NestedObject.Attributes {
			if attribute, ok := attribute.(schema.StringAttribute); ok && attribute.Computed && attribute.Default == nil {
				attribute.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
				container.NestedObject.Attributes[name] = attribute
			}
		}
	}
}
