// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestFrameworkPodSpecFeatures(t *testing.T) {
	for _, hostUsers := range []*bool{nil, ptr.To(true), ptr.To(false)} {
		input := corev1.PodSpec{
			HostUsers: hostUsers,
			Containers: []corev1.Container{{Name: "app", Image: "busybox", SecurityContext: &corev1.SecurityContext{
				ProcMount: ptr.To(corev1.UnmaskedProcMount),
			}}},
			InitContainers: []corev1.Container{{Name: "init", Image: "busybox", SecurityContext: &corev1.SecurityContext{
				ProcMount: ptr.To(corev1.DefaultProcMount),
			}}},
			Volumes: []corev1.Volume{
				{Name: "kube-api-access-test", VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{}}},
				{Name: "artifact", VolumeSource: corev1.VolumeSource{Image: &corev1.ImageVolumeSource{
					Reference: "registry.example.com/artifact:v1", PullPolicy: corev1.PullIfNotPresent,
				}}},
			},
		}
		flat, err := FlattenPodSpec(input)
		if err != nil {
			t.Fatal(err)
		}
		object := flat[0].(map[string]interface{})
		if hostUsers == nil {
			if _, exists := object["host_users"]; exists {
				t.Fatal("omitted hostUsers must stay omitted for the Framework state builder")
			}
		} else if object["host_users"] != *hostUsers {
			t.Fatalf("host_users = %v, want %t", object["host_users"], *hostUsers)
		}
		encoded, err := json.Marshal(flat)
		if err != nil {
			t.Fatal(err)
		}
		var decoded []interface{}
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		got, err := ExpandPodSpec(decoded)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.HostUsers, hostUsers) {
			t.Fatalf("hostUsers = %v, want %v", got.HostUsers, hostUsers)
		}
		if got.Containers[0].SecurityContext.ProcMount == nil || *got.Containers[0].SecurityContext.ProcMount != corev1.UnmaskedProcMount {
			t.Fatal("container proc_mount was lost")
		}
		if got.InitContainers[0].SecurityContext.ProcMount == nil || *got.InitContainers[0].SecurityContext.ProcMount != corev1.DefaultProcMount {
			t.Fatal("init container proc_mount was lost")
		}
		if len(got.Volumes) != 1 || got.Volumes[0].Name != "artifact" || !reflect.DeepEqual(got.Volumes[0].Image, input.Volumes[1].Image) {
			t.Fatalf("image volume was lost after filtering the injected volume: %#v", got.Volumes)
		}
		legacy, err := flattenPodSpec(input, true)
		if err != nil {
			t.Fatal(err)
		}
		legacyObject := legacy[0].(map[string]interface{})
		if _, exists := legacyObject["host_users"]; exists {
			t.Fatal("SDKv2 state must not contain Framework-only host_users")
		}
		if _, exists := legacyObject["container"].([]interface{})[0].(map[string]interface{})["security_context"].([]interface{})[0].(map[string]interface{})["proc_mount"]; exists {
			t.Fatal("SDKv2 state must not contain Framework-only proc_mount")
		}
		if _, exists := legacyObject["volume"].([]interface{})[0].(map[string]interface{})["image"]; exists {
			t.Fatal("SDKv2 state must not contain Framework-only image source")
		}
	}
}

func TestFrameworkPodSpecAdditionalFields(t *testing.T) {
	t.Run("unset fields preserve SDK output", func(t *testing.T) {
		input := corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{}, Affinity: &corev1.Affinity{},
			Containers: []corev1.Container{{
				SecurityContext: &corev1.SecurityContext{}, LivenessProbe: &corev1.Probe{},
				Lifecycle: &corev1.Lifecycle{PreStop: &corev1.LifecycleHandler{}},
			}},
		}
		legacy, err := flattenPodSpec(input, true)
		if err != nil {
			t.Fatal(err)
		}
		framework, err := FlattenPodSpec(input)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(legacy, framework); diff != "" {
			t.Fatalf("omitted fields changed flattening (-SDKv2 +Framework):\n%s", diff)
		}
	})

	block := func(value map[string]interface{}) []interface{} { return []interface{}{value} }
	term := map[string]interface{}{
		"topology_key":        "kubernetes.io/hostname",
		"match_label_keys":    schema.NewSet(schema.HashString, []interface{}{"pod-template-hash"}),
		"mismatch_label_keys": schema.NewSet(schema.HashString, []interface{}{"tenant"}),
	}
	affinity := block(map[string]interface{}{
		"required_during_scheduling_ignored_during_execution": block(term),
		"preferred_during_scheduling_ignored_during_execution": block(map[string]interface{}{
			"weight": 10, "pod_affinity_term": block(term),
		}),
	})
	container := map[string]interface{}{
		"name": "app",
		"security_context": block(map[string]interface{}{
			"app_armor_profile": map[string]interface{}{"type": "Localhost", "localhost_profile": "profiles/app"},
		}),
		"liveness_probe": block(map[string]interface{}{
			"termination_grace_period_seconds": 5,
			"tcp_socket":                       block(map[string]interface{}{"port": "8080", "host": "127.0.0.1"}),
		}),
		"startup_probe": block(map[string]interface{}{
			"termination_grace_period_seconds": 6,
			"tcp_socket":                       block(map[string]interface{}{"port": "8080", "host": "127.0.0.2"}),
		}),
		"lifecycle": block(map[string]interface{}{
			"post_start": block(map[string]interface{}{"sleep": map[string]interface{}{"seconds": 2}}),
			"pre_stop":   block(map[string]interface{}{"sleep": map[string]interface{}{"seconds": 0}}),
		}),
		"volume_mount": block(map[string]interface{}{
			"name": "data", "mount_path": "/data", "read_only": true, "recursive_read_only": "Enabled",
		}),
	}
	raw := block(map[string]interface{}{
		"set_hostname_as_fqdn": false,
		"security_context": block(map[string]interface{}{
			"app_armor_profile":          map[string]interface{}{"type": "RuntimeDefault"},
			"supplemental_groups_policy": "Strict", "se_linux_change_policy": "Recursive",
		}),
		"affinity":  block(map[string]interface{}{"pod_affinity": affinity, "pod_anti_affinity": affinity}),
		"container": block(container), "init_container": block(container),
	})
	spec, err := ExpandPodSpec(raw)
	if err != nil {
		t.Fatal(err)
	}
	if spec.SetHostnameAsFQDN == nil || *spec.SetHostnameAsFQDN {
		t.Fatal("explicit set_hostname_as_fqdn = false was lost")
	}
	wantContext := &corev1.PodSecurityContext{
		AppArmorProfile:          &corev1.AppArmorProfile{Type: corev1.AppArmorProfileTypeRuntimeDefault},
		SupplementalGroupsPolicy: ptr.To(corev1.SupplementalGroupsPolicyStrict),
		SELinuxChangePolicy:      ptr.To(corev1.SELinuxChangePolicyRecursive),
	}
	if diff := cmp.Diff(wantContext, spec.SecurityContext); diff != "" {
		t.Fatalf("Pod security context (-want +got):\n%s", diff)
	}
	for _, c := range append(spec.Containers, spec.InitContainers...) {
		if c.SecurityContext.AppArmorProfile.LocalhostProfile == nil || *c.SecurityContext.AppArmorProfile.LocalhostProfile != "profiles/app" ||
			c.LivenessProbe.TerminationGracePeriodSeconds == nil || *c.LivenessProbe.TerminationGracePeriodSeconds != 5 ||
			c.StartupProbe.TerminationGracePeriodSeconds == nil || *c.StartupProbe.TerminationGracePeriodSeconds != 6 ||
			c.LivenessProbe.TCPSocket.Host != "127.0.0.1" || c.StartupProbe.TCPSocket.Host != "127.0.0.2" ||
			c.Lifecycle.PostStart.Sleep == nil || c.Lifecycle.PostStart.Sleep.Seconds != 2 ||
			c.Lifecycle.PreStop.Sleep == nil || c.Lifecycle.PreStop.Sleep.Seconds != 0 ||
			c.VolumeMounts[0].RecursiveReadOnly == nil || *c.VolumeMounts[0].RecursiveReadOnly != corev1.RecursiveReadOnlyEnabled {
			t.Fatalf("container fields were not expanded: %#v", c)
		}
	}
	for _, terms := range [][]corev1.PodAffinityTerm{
		spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
		{spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm},
		spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
		{spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm},
	} {
		if diff := cmp.Diff([]string{"pod-template-hash"}, terms[0].MatchLabelKeys); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff([]string{"tenant"}, terms[0].MismatchLabelKeys); diff != "" {
			t.Fatal(diff)
		}
	}
	// A filtered token mount must not shift the recursive-read-only value onto another mount.
	spec.Containers[0].VolumeMounts = append([]corev1.VolumeMount{{Name: "kube-api-access-test", MountPath: "/token"}}, spec.Containers[0].VolumeMounts...)
	flat, err := FlattenPodSpec(*spec)
	if err != nil {
		t.Fatal(err)
	}
	encoded := fmt.Sprintf("%#v", flat)
	legacy, err := flattenPodSpec(*spec.DeepCopy(), true)
	if err != nil {
		t.Fatal(err)
	}
	legacyFields := fmt.Sprintf("%#v", legacy)
	for _, field := range []string{"set_hostname_as_fqdn", "app_armor_profile", "supplemental_groups_policy", "se_linux_change_policy", "termination_grace_period_seconds", "sleep", "host", "recursive_read_only", "match_label_keys", "mismatch_label_keys"} {
		key := `"` + field + `":`
		if !strings.Contains(encoded, key) || strings.Contains(legacyFields, key) {
			t.Fatalf("%s must be present only in Framework flattening", field)
		}
	}
	flatContainer := flat[0].(map[string]interface{})["container"].([]interface{})[0].(map[string]interface{})
	mounts := flatContainer["volume_mount"].([]interface{})
	if len(mounts) != 1 || mounts[0].(map[string]interface{})["recursive_read_only"] != "Enabled" {
		t.Fatalf("recursive read-only mount lost after filtering: %#v", mounts)
	}

	t.Run("Windows explicit false and omitted fields", func(t *testing.T) {
		windows, err := ExpandPodSpec(block(map[string]interface{}{"container": block(map[string]interface{}{
			"name": "windows", "security_context": block(map[string]interface{}{
				"windows_options": map[string]interface{}{"host_process": false, "run_as_username": "ContainerUser"},
			}),
		})}))
		if err != nil {
			t.Fatal(err)
		}
		options := windows.Containers[0].SecurityContext.WindowsOptions
		if options.HostProcess == nil || *options.HostProcess || options.GMSACredentialSpec != nil || options.RunAsUserName == nil || *options.RunAsUserName != "ContainerUser" {
			t.Fatalf("Windows optional values lost: %#v", options)
		}
		flat, err := FlattenPodSpec(*windows)
		if err != nil {
			t.Fatal(err)
		}
		value := flat[0].(map[string]interface{})["container"].([]interface{})[0].(map[string]interface{})["security_context"].([]interface{})[0].(map[string]interface{})
		if diff := cmp.Diff(map[string]interface{}{"host_process": false, "run_as_username": "ContainerUser"}, value["windows_options"]); diff != "" {
			t.Fatal(diff)
		}
		if _, exists := value["app_armor_profile"]; exists {
			t.Fatal("omitted AppArmor profile must remain absent")
		}
	})
}

func TestFlattenTolerations(t *testing.T) {
	cases := []struct {
		Input          []corev1.Toleration
		isTemplate     bool
		ExpectedOutput []interface{}
	}{
		{
			[]corev1.Toleration{
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
			},
			false,
			[]interface{}{
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
			},
		},
		{
			[]corev1.Toleration{
				{
					Key:      "node-role.kubernetes.io/other-worker",
					Operator: "Exists",
				},
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
			},
			false,
			[]interface{}{
				map[string]interface{}{
					"key":      "node-role.kubernetes.io/other-worker",
					"operator": "Exists",
				},
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
			},
		},
		{
			[]corev1.Toleration{
				{
					Effect:            "NoExecute",
					TolerationSeconds: ptr.To(int64(120)),
				},
			},
			false,
			[]interface{}{
				map[string]interface{}{
					"effect":             "NoExecute",
					"toleration_seconds": "120",
				},
			},
		},
		{
			[]corev1.Toleration{
				{
					Effect:            "NoExecute",
					Key:               "node.kubernetes.io/unreachable",
					Operator:          "Exists",
					TolerationSeconds: ptr.To(int64(120)),
				},
			},
			false,
			[]interface{}{},
		},
		{
			[]corev1.Toleration{},
			false,
			[]interface{}{},
		},
		{
			[]corev1.Toleration{},
			true,
			[]interface{}{},
		},
		{
			[]corev1.Toleration{
				{
					Effect:            "NoExecute",
					Key:               "node.kubernetes.io/unreachable",
					Operator:          "Exists",
					TolerationSeconds: ptr.To(int64(120)),
				},
			},
			true,
			[]interface{}{
				map[string]interface{}{
					"effect":             "NoExecute",
					"key":                "node.kubernetes.io/unreachable",
					"operator":           "Exists",
					"toleration_seconds": "120",
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenTolerations(tc.Input, tc.isTemplate)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandTolerations(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput []*corev1.Toleration
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
			},
			[]*corev1.Toleration{
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"key":   "node-role.kubernetes.io/spot-worker",
					"value": "true",
				},
				map[string]interface{}{
					"key":      "node-role.kubernetes.io/other-worker",
					"operator": "Exists",
				},
			},
			[]*corev1.Toleration{
				{
					Key:   "node-role.kubernetes.io/spot-worker",
					Value: "true",
				},
				{
					Key:      "node-role.kubernetes.io/other-worker",
					Operator: "Exists",
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"effect":             "NoExecute",
					"toleration_seconds": "120",
				},
			},
			[]*corev1.Toleration{
				{
					Effect:            "NoExecute",
					TolerationSeconds: ptr.To(int64(120)),
				},
			},
		},
		{
			[]interface{}{},
			[]*corev1.Toleration{},
		},
	}

	for _, tc := range cases {
		output, err := expandTolerations(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenSecretVolumeSource(t *testing.T) {
	cases := []struct {
		Input          *corev1.SecretVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&corev1.SecretVolumeSource{
				DefaultMode: ptr.To(int32(0644)),
				SecretName:  "secret1",
				Optional:    ptr.To(true),
				Items: []corev1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: ptr.To(int32(0600)),
						Path: "etc/foo.txt",
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0644",
					"secret_name":  "secret1",
					"optional":     true,
					"items": []interface{}{
						map[string]interface{}{
							"key":  "foo.txt",
							"mode": "0600",
							"path": "etc/foo.txt",
						},
					},
				},
			},
		},
		{
			&corev1.SecretVolumeSource{
				DefaultMode: ptr.To(int32(0755)),
				SecretName:  "secret2",
				Items: []corev1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0755",
					"secret_name":  "secret2",
					"items": []interface{}{
						map[string]interface{}{
							"key":  "bar.txt",
							"path": "etc/bar.txt",
						},
					},
				},
			},
		},
		{
			&corev1.SecretVolumeSource{},
			[]interface{}{map[string]interface{}{}},
		},
	}

	for _, tc := range cases {
		output := flattenSecretVolumeSource(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandSecretVolumeSource(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *corev1.SecretVolumeSource
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0644",
					"secret_name":  "secret1",
					"optional":     true,
					"items": []interface{}{
						map[string]interface{}{
							"key":  "foo.txt",
							"mode": "0600",
							"path": "etc/foo.txt",
						},
					},
				},
			},
			&corev1.SecretVolumeSource{
				DefaultMode: ptr.To(int32(0644)),
				SecretName:  "secret1",
				Optional:    ptr.To(true),
				Items: []corev1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: ptr.To(int32(0600)),
						Path: "etc/foo.txt",
					},
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0755",
					"secret_name":  "secret2",
					"items": []interface{}{
						map[string]interface{}{
							"key":  "bar.txt",
							"path": "etc/bar.txt",
						},
					},
				},
			},
			&corev1.SecretVolumeSource{
				DefaultMode: ptr.To(int32(0755)),
				SecretName:  "secret2",
				Items: []corev1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
		},
		{
			[]interface{}{},
			&corev1.SecretVolumeSource{},
		},
	}

	for _, tc := range cases {
		output, err := expandSecretVolumeSource(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenEmptyDirVolumeSource(t *testing.T) {
	size, _ := resource.ParseQuantity("64Mi")

	cases := []struct {
		Input          *corev1.EmptyDirVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
			[]interface{}{
				map[string]interface{}{
					"medium": "Memory",
				},
			},
		},
		{
			&corev1.EmptyDirVolumeSource{
				Medium:    corev1.StorageMediumMemory,
				SizeLimit: &size,
			},
			[]interface{}{
				map[string]interface{}{
					"medium":     "Memory",
					"size_limit": "64Mi",
				},
			},
		},
		{
			&corev1.EmptyDirVolumeSource{},
			[]interface{}{
				map[string]interface{}{
					"medium": "",
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenEmptyDirVolumeSource(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenConfigMapVolumeSource(t *testing.T) {
	cases := []struct {
		Input          *corev1.ConfigMapVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			&corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "configmap1",
				},
				DefaultMode: ptr.To(int32(0644)),
				Optional:    ptr.To(true),
				Items: []corev1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: ptr.To(int32(0600)),
						Path: "etc/foo.txt",
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0644",
					"name":         "configmap1",
					"optional":     true,
					"items": []interface{}{
						map[string]interface{}{
							"key":  "foo.txt",
							"mode": "0600",
							"path": "etc/foo.txt",
						},
					},
				},
			},
		},
		{
			&corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "configmap2",
				},
				DefaultMode: ptr.To(int32(0755)),
				Items: []corev1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0755",
					"name":         "configmap2",
					"items": []interface{}{
						map[string]interface{}{
							"key":  "bar.txt",
							"path": "etc/bar.txt",
						},
					},
				},
			},
		},
		{
			&corev1.ConfigMapVolumeSource{},
			[]interface{}{map[string]interface{}{"name": ""}},
		},
	}

	for _, tc := range cases {
		output := flattenConfigMapVolumeSource(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandConfigMapVolumeSource(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *corev1.ConfigMapVolumeSource
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0644",
					"name":         "configmap1",
					"optional":     true,
					"items": []interface{}{
						map[string]interface{}{
							"key":  "foo.txt",
							"mode": "0600",
							"path": "etc/foo.txt",
						},
					},
				},
			},
			&corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "configmap1",
				},
				DefaultMode: ptr.To(int32(0644)),
				Optional:    ptr.To(true),
				Items: []corev1.KeyToPath{
					{
						Key:  "foo.txt",
						Mode: ptr.To(int32(0600)),
						Path: "etc/foo.txt",
					},
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"default_mode": "0755",
					"name":         "configmap2",
					"items": []interface{}{
						map[string]interface{}{
							"key":  "bar.txt",
							"path": "etc/bar.txt",
						},
					},
				},
			},
			&corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "configmap2",
				},
				DefaultMode: ptr.To(int32(0755)),
				Items: []corev1.KeyToPath{
					{
						Key:  "bar.txt",
						Path: "etc/bar.txt",
					},
				},
			},
		},
		{
			[]interface{}{},
			&corev1.ConfigMapVolumeSource{},
		},
	}

	for _, tc := range cases {
		output, err := expandConfigMapVolumeSource(tc.Input)
		if err != nil {
			t.Fatalf("Unexpected failure in expander.\nInput: %#v, error: %#v", tc.Input, err)
		}
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandThenFlatten_projected_volume(t *testing.T) {
	cases := []struct {
		Input *corev1.ProjectedVolumeSource
	}{
		{
			Input: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: "secret-1"},
						},
					},
					{
						ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: "config-1"},
						},
					},
					{
						ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: "config-2"},
						},
					},
					{
						DownwardAPI: &corev1.DownwardAPIProjection{
							Items: []corev1.DownwardAPIVolumeFile{
								{Path: "path-1"},
							},
						},
					},
					{
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Audience: "audience-1",
						},
					},
				},
			},
		},
	}
	for _, tc := range cases {
		in := tc.Input
		flattenedFirst := flattenProjectedVolumeSource(in)
		out, err := expandProjectedVolumeSource(flattenedFirst)
		if err != nil {
			t.Fatal(err)
		}
		if !cmp.Equal(in, out) {
			t.Fatal(cmp.Diff(in, out))
		}

		flattenedAgain := flattenProjectedVolumeSource(out)
		if !cmp.Equal(flattenedFirst, flattenedAgain) {
			t.Fatal(cmp.Diff(flattenedFirst, flattenedAgain))
		}
	}

}

func TestExpandCSIVolumeSource(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *corev1.CSIVolumeSource
	}{
		{
			Input: []interface{}{
				map[string]interface{}{
					"driver":    "secrets-store.csi.k8s.io",
					"read_only": true,
					"volume_attributes": map[string]interface{}{
						"secretProviderClass": "azure-keyvault",
					},
					"fs_type": "nfs",
					"node_publish_secret_ref": []interface{}{
						map[string]interface{}{
							"name": "secrets-store",
						},
					},
				},
			},
			ExpectedOutput: &corev1.CSIVolumeSource{
				Driver:   "secrets-store.csi.k8s.io",
				ReadOnly: ptr.To(true),
				FSType:   ptr.To("nfs"),
				VolumeAttributes: map[string]string{
					"secretProviderClass": "azure-keyvault",
				},
				NodePublishSecretRef: &corev1.LocalObjectReference{
					Name: "secrets-store",
				},
			},
		},
		{
			Input: []interface{}{
				map[string]interface{}{
					"driver": "other-csi-driver.k8s.io",
					"volume_attributes": map[string]interface{}{
						"objects": `array: 
						- |
							objectName: secret-1
							objectType: secret `,
					},
				},
			},
			ExpectedOutput: &corev1.CSIVolumeSource{
				Driver:   "other-csi-driver.k8s.io",
				ReadOnly: nil,
				FSType:   nil,
				VolumeAttributes: map[string]string{
					"objects": `array: 
						- |
							objectName: secret-1
							objectType: secret `,
				},
				NodePublishSecretRef: nil,
			},
		},
	}
	for _, tc := range cases {
		output := expandCSIVolumeSource(tc.Input)
		if !reflect.DeepEqual(tc.ExpectedOutput, output) {
			t.Fatalf("Unexpected output from CSI Volume Source Expander. \nExpected: %#v, Given: %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenCSIVolumeSource(t *testing.T) {
	cases := []struct {
		Input          *corev1.CSIVolumeSource
		ExpectedOutput []interface{}
	}{
		{
			Input: &corev1.CSIVolumeSource{
				Driver:   "secrets-store.csi.k8s.io",
				ReadOnly: ptr.To(true),
				FSType:   ptr.To("nfs"),
				VolumeAttributes: map[string]string{
					"secretProviderClass": "azure-keyvault",
				},
				NodePublishSecretRef: &corev1.LocalObjectReference{
					Name: "secrets-store",
				},
			},
			ExpectedOutput: []interface{}{
				map[string]interface{}{
					"driver":    "secrets-store.csi.k8s.io",
					"read_only": true,
					"volume_attributes": map[string]string{
						"secretProviderClass": "azure-keyvault",
					},
					"fs_type": "nfs",
					"node_publish_secret_ref": []interface{}{
						map[string]interface{}{
							"name": "secrets-store",
						},
					},
				},
			},
		},
		{
			Input: &corev1.CSIVolumeSource{
				Driver:   "other-csi-driver.k8s.io",
				ReadOnly: nil,
				FSType:   nil,
				VolumeAttributes: map[string]string{
					"objects": `array: 
					- |
						objectName: secret-1
						objectType: secret `,
				},
				NodePublishSecretRef: nil,
			},
			ExpectedOutput: []interface{}{
				map[string]interface{}{
					"driver": "other-csi-driver.k8s.io",
					"volume_attributes": map[string]string{
						"objects": `array: 
					- |
						objectName: secret-1
						objectType: secret `,
					},
				},
			},
		},
	}
	for _, tc := range cases {
		output := flattenCSIVolumeSource(tc.Input)
		if !reflect.DeepEqual(tc.ExpectedOutput, output) {
			t.Fatalf("Unexpected result from flattener. \nExpected %#v, \nGiven %#v",
				tc.ExpectedOutput, output)
		}
	}
}
