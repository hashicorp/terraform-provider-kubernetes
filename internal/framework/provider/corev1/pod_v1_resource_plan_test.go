// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

func TestPodV1SpecRequiresReplacement(t *testing.T) {
	base := func() corev1.PodSpec {
		return corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app", Image: "image:1",
				VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
				Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("500m"),
				}},
			}},
			Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
		}
	}
	deadline := func(seconds int64) func(*corev1.PodSpec) {
		return func(s *corev1.PodSpec) { s.ActiveDeadlineSeconds = ptr.To(seconds) }
	}
	for name, test := range map[string]struct {
		prior, edit func(*corev1.PodSpec)
		want        bool
	}{
		"unchanged":             {edit: func(*corev1.PodSpec) {}},
		"host users default":    {edit: func(s *corev1.PodSpec) { s.HostUsers = ptr.To(true) }},
		"enable user namespace": {edit: func(s *corev1.PodSpec) { s.HostUsers = ptr.To(false) }, want: true},
		"disable user namespace": {prior: func(s *corev1.PodSpec) { s.HostUsers = ptr.To(false) }, edit: func(s *corev1.PodSpec) {
			s.HostUsers = nil
		}, want: true},
		"proc mount default": {edit: func(s *corev1.PodSpec) {
			s.Containers[0].SecurityContext = &corev1.SecurityContext{ProcMount: ptr.To(corev1.DefaultProcMount)}
		}},
		"unmasked proc mount": {edit: func(s *corev1.PodSpec) {
			s.Containers[0].SecurityContext = &corev1.SecurityContext{ProcMount: ptr.To(corev1.UnmaskedProcMount)}
		}, want: true},
		"image volume": {edit: func(s *corev1.PodSpec) {
			s.Volumes[0].VolumeSource = corev1.VolumeSource{Image: &corev1.ImageVolumeSource{Reference: "registry.example/data:v1"}}
		}, want: true},
		"active deadline is set":     {edit: deadline(60)},
		"active deadline is lowered": {prior: deadline(120), edit: deadline(60)},
		"active deadline is raised":  {prior: deadline(60), edit: deadline(120), want: true},
		"active deadline is removed": {prior: deadline(60), edit: func(s *corev1.PodSpec) { s.ActiveDeadlineSeconds = nil }, want: true},
		"container image": {edit: func(s *corev1.PodSpec) {
			s.Containers[0].Image = "image:2"
		}, want: true},
		"empty and unset values are equivalent": {edit: func(s *corev1.PodSpec) {
			s.SecurityContext = &corev1.PodSecurityContext{SupplementalGroups: []int64{}}
			s.Containers[0].Args = []string{}
			s.NodeSelector = map[string]string{}
			s.HostNetwork = false
		}},
		"quantity spelling": {edit: func(s *corev1.PodSpec) {
			s.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("0.5")
		}},
		"volume mount path": {edit: func(s *corev1.PodSpec) {
			s.Containers[0].VolumeMounts[0].MountPath = "/other"
		}, want: true},
		"added toleration": {edit: func(s *corev1.PodSpec) {
			s.Tolerations = []corev1.Toleration{{Key: "k", Operator: corev1.TolerationOpExists}}
		}, want: true},
		"init container image": {edit: func(s *corev1.PodSpec) {
			s.InitContainers = []corev1.Container{{Name: "init", Image: "image:1"}}
		}, want: true},
	} {
		t.Run(name, func(t *testing.T) {
			prior, planned := base(), base()
			if test.prior != nil {
				test.prior(&prior)
				test.prior(&planned)
			}
			test.edit(&planned)
			if got := podV1SpecRequiresReplacement(prior, planned); got != test.want {
				t.Fatalf("requires replacement = %t, want %t", got, test.want)
			}
		})
	}
}

type planClientsets struct {
	kubernetes.KubeClientsets
	client *k8sclient.Clientset
}

func (c planClientsets) MainClientset() (*k8sclient.Clientset, error) { return c.client, nil }
func (planClientsets) GetIgnoreAnnotations() []string                 { return nil }
func (planClientsets) GetIgnoreLabels() []string                      { return nil }

// A Pod created by SDKv2 holds runAsNonRoot: false, which the prior state
// cannot tell apart from an unset field.
func TestPodV1ModifyPlanReadsLivePod(t *testing.T) {
	ctx := context.Background()
	for name, test := range map[string]struct {
		status      int
		wantReplace bool
		wantError   bool
		resources   string
	}{
		"live Pod holds the removed block":            {status: http.StatusOK, wantReplace: true},
		"Pod no longer exists":                        {status: http.StatusNotFound},
		"forbidden":                                   {status: http.StatusForbidden, wantError: true},
		"unavailable":                                 {status: http.StatusServiceUnavailable, wantError: true},
		"empty legacy resources with API unavailable": {status: http.StatusInternalServerError, resources: `[]`},
		"empty resource element with API unavailable": {status: http.StatusInternalServerError, resources: `[{}]`},
		"zero resource maps with API unavailable":     {status: http.StatusInternalServerError, resources: `[{"limits":{},"requests":{}}]`},
	} {
		t.Run(name, func(t *testing.T) {
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Path != "/api/v1/namespaces/ns/pods/p" || test.status != http.StatusOK {
					http.Error(w, "unavailable", test.status)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(&corev1.Pod{Spec: corev1.PodSpec{
					Containers:      []corev1.Container{{Name: "c", Image: "i"}},
					SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: ptr.To(false)},
				}})
			}))
			defer server.Close()
			client, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			pod := &PodV1{SDKv2Meta: func() any { return planClientsets{client: client} }}
			var schemaResp fwresource.SchemaResponse
			pod.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
			value := func(spec string) tfsdk.State {
				raw := tfprotov6.RawState{JSON: []byte(`{"id":"ns/p","metadata":[{"name":"p","namespace":"ns"}],
					"spec":[{"container":[{"name":"c","image":"i"}]` + spec + `}]}`)}
				v, err := raw.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				return tfsdk.State{Schema: schemaResp.Schema, Raw: v}
			}
			state, planned := value(`,"security_context":[{"run_as_non_root":false}]`), value("")
			if test.resources != "" {
				legacy := `{"id":"ns/p","metadata":[{"name":"p","namespace":"ns"}],"target_state":[],"spec":[{"container":[{"name":"c","image":"i","resources":RESOURCES}]}]}`
				upgraded := fwresource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
				pod.UpgradeState(ctx)[1].StateUpgrader(ctx, fwresource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: []byte(strings.Replace(legacy, "RESOURCES", test.resources, 1))}}, &upgraded)
				if upgraded.Diagnostics.HasError() {
					t.Fatal(upgraded.Diagnostics)
				}
				state = upgraded.State
				raw := &tfprotov6.RawState{JSON: []byte(strings.Replace(legacy, "RESOURCES", `{"limits":{},"requests":{}}`, 1))}
				planned.Raw, err = raw.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				if !state.Raw.Equal(planned.Raw) {
					t.Fatal("upgraded state does not equal object plan")
				}
			}
			req := fwresource.ModifyPlanRequest{
				State:  state,
				Plan:   tfsdk.Plan(planned),
				Config: tfsdk.Config(planned),
			}
			resp := fwresource.ModifyPlanResponse{Plan: req.Plan}
			pod.ModifyPlan(ctx, req, &resp)
			if test.resources != "" && requests.Load() != 0 {
				t.Fatalf("unchanged plan made %d GETs", requests.Load())
			}
			if resp.Diagnostics.HasError() != test.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if replace := len(resp.RequiresReplace) == 1 && resp.RequiresReplace[0].Equal(path.Root("spec")); replace != test.wantReplace {
				t.Fatalf("requires replace = %v, want %t", resp.RequiresReplace, test.wantReplace)
			}
		})
	}
}
