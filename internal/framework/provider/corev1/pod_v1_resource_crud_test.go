// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

func TestPodFeatureRetention(t *testing.T) {
	expected := corev1.PodSpec{
		HostUsers: ptr.To(false),
		Containers: []corev1.Container{{Name: "app", SecurityContext: &corev1.SecurityContext{ProcMount: ptr.To(corev1.UnmaskedProcMount)},
			VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}}}},
		InitContainers: []corev1.Container{{Name: "init", SecurityContext: &corev1.SecurityContext{ProcMount: ptr.To(corev1.UnmaskedProcMount)}}},
		Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{Image: &corev1.ImageVolumeSource{
			Reference: "example/data:1", PullPolicy: corev1.PullIfNotPresent,
		}}}},
	}
	for name, test := range map[string]struct {
		edit func(*corev1.PodSpec)
		want string
	}{
		"retained": {edit: func(*corev1.PodSpec) {}},
		"admission reorders": {edit: func(actual *corev1.PodSpec) {
			actual.Containers = append([]corev1.Container{{Name: "injected"}}, actual.Containers...)
			actual.Volumes = append([]corev1.Volume{{Name: "injected"}}, actual.Volumes...)
		}},
		"host users dropped": {edit: func(actual *corev1.PodSpec) { actual.HostUsers = nil }, want: "UserNamespacesSupport"},
		"proc mount rewritten": {edit: func(actual *corev1.PodSpec) {
			actual.Containers[0].SecurityContext.ProcMount = ptr.To(corev1.DefaultProcMount)
		}, want: "ProcMountType"},
		"init proc mount dropped": {edit: func(actual *corev1.PodSpec) { actual.InitContainers[0].SecurityContext = nil }, want: "ProcMountType"},
		"image dropped":           {edit: func(actual *corev1.PodSpec) { actual.Volumes = nil }, want: "ImageVolume"},
		"image became empty dir": {edit: func(actual *corev1.PodSpec) {
			actual.Volumes[0].VolumeSource = corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}
		}, want: "ImageVolume"},
		"reference rewritten": {edit: func(actual *corev1.PodSpec) { actual.Volumes[0].Image.Reference = "other:1" }, want: "ImageVolume"},
		"policy rewritten":    {edit: func(actual *corev1.PodSpec) { actual.Volumes[0].Image.PullPolicy = corev1.PullAlways }, want: "ImageVolume"},
		"mount dropped":       {edit: func(actual *corev1.PodSpec) { actual.Containers[0].VolumeMounts = nil }, want: "ImageVolume"},
	} {
		t.Run(name, func(t *testing.T) {
			actual := expected.DeepCopy()
			test.edit(actual)
			err := podspec.CheckPodFeaturePreservation(&expected, actual)
			if test.want == "" && err != nil || test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
	t.Run("omitted and explicit defaults", func(t *testing.T) {
		defaults := corev1.PodSpec{HostUsers: ptr.To(true), Containers: []corev1.Container{{Name: "app", SecurityContext: &corev1.SecurityContext{ProcMount: ptr.To(corev1.DefaultProcMount)}}}}
		actual := corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}}
		if podspec.HasGatedFeatures(&defaults) || podspec.HasGatedFeatures(&actual) || !podspec.HasGatedFeatures(&expected) {
			t.Fatal("incorrect preflight selection")
		}
		if err := podspec.CheckPodFeaturePreservation(&defaults, &actual); err != nil {
			t.Fatal(err)
		}
		actual.HostUsers = ptr.To(false)
		if err := podspec.CheckPodFeaturePreservation(&defaults, &actual); err == nil {
			t.Fatal("admission must not silently change configured host_users=true")
		}
	})
	for name, spec := range map[string]corev1.PodSpec{
		"user namespace":  {HostUsers: expected.HostUsers},
		"proc mount":      {Containers: expected.Containers},
		"init proc mount": {InitContainers: expected.InitContainers},
		"image volume":    {Volumes: expected.Volumes},
	} {
		t.Run("preflight/"+name, func(t *testing.T) {
			if !podspec.HasGatedFeatures(&spec) {
				t.Fatal("feature request must be checked before writing")
			}
		})
	}
}

func TestPodV1CreateFeaturePreflight(t *testing.T) {
	ctx := context.Background()
	for name, test := range map[string]struct {
		feature, dropPreview, dropWrite bool
		wantWrites                      int
	}{
		"ordinary Pod avoids preflight": {wantWrites: 1},
		"feature retained":              {feature: true, wantWrites: 1},
		"dry run drops feature":         {feature: true, dropPreview: true},
		"real write drops feature":      {feature: true, dropWrite: true, wantWrites: 1},
	} {
		t.Run(name, func(t *testing.T) {
			var writes, previews int
			var current corev1.Pod
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPost {
					var obj corev1.Pod
					if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					preview := r.URL.Query().Get("dryRun") == metav1.DryRunAll
					if preview {
						previews++
					} else {
						writes++
					}
					if preview && test.dropPreview || !preview && test.dropWrite {
						obj.Spec.HostUsers = nil
					}
					obj.Status.Phase = corev1.PodRunning
					current = obj
					_ = json.NewEncoder(w).Encode(&obj)
					return
				}
				_ = json.NewEncoder(w).Encode(&current)
			}))
			defer server.Close()
			client, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL, ContentConfig: rest.ContentConfig{ContentType: "application/json"}})
			if err != nil {
				t.Fatal(err)
			}
			pod := &PodV1{SDKv2Meta: func() any { return planClientsets{client: client} }}
			var schemaResponse fwresource.SchemaResponse
			pod.Schema(ctx, fwresource.SchemaRequest{}, &schemaResponse)
			feature := ""
			if test.feature {
				feature = `,"host_users":false`
			}
			raw := tfprotov6.RawState{JSON: []byte(`{"metadata":[{"name":"p","namespace":"ns"}],"spec":[{"container":[{"name":"app","image":"example/app:1"}]` + feature + `}]}`)}
			value, err := raw.Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			planned := tfsdk.State{Schema: schemaResponse.Schema, Raw: value}
			if diags := planned.SetAttribute(ctx, path.Root("id"), types.StringUnknown()); diags.HasError() {
				t.Fatal(diags)
			}
			response := fwresource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			pod.Create(ctx, fwresource.CreateRequest{Plan: tfsdk.Plan(planned), Config: tfsdk.Config{Schema: schemaResponse.Schema, Raw: value}}, &response)
			if response.Diagnostics.HasError() != (test.dropPreview || test.dropWrite) {
				t.Fatalf("diagnostics = %v", response.Diagnostics)
			}
			if writes != test.wantWrites || previews != 0 && !test.feature || previews != 1 && test.feature {
				t.Fatalf("writes = %d, previews = %d", writes, previews)
			}
			if writes > 0 {
				var id types.String
				if diags := response.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() || id.ValueString() != "ns/p" {
					t.Fatalf("created identity was lost: %s; %v", id, diags)
				}
			}
		})
	}
}

func TestPodV1UpdatePatch(t *testing.T) {
	spec := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "image:1"}}}
	for _, test := range []struct {
		name   string
		prior  metav1.ObjectMeta
		plan   metav1.ObjectMeta
		live   metav1.ObjectMeta
		expect map[string]interface{}
	}{
		{
			name: "state-only empty maps",
			plan: metav1.ObjectMeta{Labels: map[string]string{}, Annotations: map[string]string{}},
			live: metav1.ObjectMeta{Labels: map[string]string{"controller": "keep"}},
		},
		{
			name: "first managed key retains external metadata",
			plan: metav1.ObjectMeta{Labels: map[string]string{"managed": "new"}},
			live: metav1.ObjectMeta{Labels: map[string]string{"controller": "keep"}},
			expect: map[string]interface{}{
				"metadata": map[string]interface{}{"labels": map[string]interface{}{"managed": "new"}},
			},
		},
		{
			name:  "remove only managed keys",
			prior: metav1.ObjectMeta{Annotations: map[string]string{"managed/x~y": "old"}},
			live:  metav1.ObjectMeta{Annotations: map[string]string{"managed/x~y": "old", "controller": "keep"}},
			expect: map[string]interface{}{
				"metadata": map[string]interface{}{"annotations": map[string]interface{}{"managed/x~y": nil}},
			},
		},
		{
			name:  "already converged metadata",
			prior: metav1.ObjectMeta{Labels: map[string]string{"managed": "old"}},
			plan:  metav1.ObjectMeta{Labels: map[string]string{"managed": "new"}},
			live:  metav1.ObjectMeta{Labels: map[string]string{"managed": "new", "controller": "keep"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual := podV1UpdatePatch(test.prior, test.plan, &spec, &spec, &corev1.Pod{
				ObjectMeta: test.live,
				Spec:       spec,
			})
			if test.expect == nil {
				test.expect = map[string]interface{}{}
			}
			if !reflect.DeepEqual(actual, test.expect) {
				t.Fatalf("patch = %#v, want %#v", actual, test.expect)
			}
		})
	}
	t.Run("active deadline edit", func(t *testing.T) {
		seconds := int64(120)
		planned := spec.DeepCopy()
		planned.ActiveDeadlineSeconds = &seconds
		patch := podV1UpdatePatch(metav1.ObjectMeta{}, metav1.ObjectMeta{}, &spec, planned, &corev1.Pod{Spec: spec})
		data, err := json.Marshal(patch)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"spec":{"activeDeadlineSeconds":120}}` {
			t.Fatalf("unexpected patch: %s", data)
		}
	})
}
