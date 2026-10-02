// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestThreeWayStrategicMerge(t *testing.T) {
	const (
		tcp53   = `{"containerPort":53,"protocol":"TCP"}`
		udp53   = `{"containerPort":53,"protocol":"UDP"}`
		tcp8080 = `{"containerPort":8080,"protocol":"TCP"}`
	)
	pod := func(image string, ports ...string) string {
		return `{"containers":[{"name":"app","image":"` + image + `","ports":[` + strings.Join(ports, ",") + `]}]}`
	}
	spread := func(whens ...string) string {
		var constraints []string
		for _, when := range whens {
			constraints = append(constraints, `{"maxSkew":1,"topologyKey":"kubernetes.io/hostname","whenUnsatisfiable":"`+when+`"}`)
		}
		return `{"containers":[{"name":"app","image":"a"}],"topologySpreadConstraints":[` + strings.Join(constraints, ",") + `]}`
	}
	cases := []struct {
		name                        string
		original, modified, current string
		want                        string
	}{
		{name: "add UDP next to TCP", original: pod("a", tcp53), modified: pod("a", tcp53, udp53), want: pod("a", tcp53, udp53)},
		{name: "remove UDP", original: pod("a", tcp53, udp53), modified: pod("a", tcp53), want: pod("a", tcp53)},
		{name: "interleaved", original: pod("a", tcp53), modified: pod("a", tcp53, tcp8080, udp53), want: pod("a", tcp53, tcp8080, udp53)},
		{name: "reorder", original: pod("a", tcp53, tcp8080, udp53), modified: pod("a", udp53, tcp8080, tcp53), want: pod("a", udp53, tcp8080, tcp53)},
		{name: "remove all ports", original: pod("a", tcp53, udp53), modified: `{"containers":[{"name":"app","image":"a"}]}`, want: `{"containers":[{"name":"app","image":"a"}]}`},
		{
			name:     "new container",
			original: `{"containers":[{"name":"main","image":"a"}]}`,
			modified: `{"containers":[{"name":"main","image":"a"},{"name":"app","image":"a","ports":[` + tcp53 + `,` + tcp8080 + `,` + udp53 + `]}]}`,
			want:     `{"containers":[{"name":"main","image":"a"},{"name":"app","image":"a","ports":[` + tcp53 + `,` + tcp8080 + `,` + udp53 + `]}]}`,
		},
		{
			name:     "topology spread constraints sharing topologyKey",
			original: spread("DoNotSchedule"),
			modified: spread("DoNotSchedule", "ScheduleAnyway"),
			want:     spread("DoNotSchedule", "ScheduleAnyway"),
		},
		{
			name:     "current already in planned order",
			original: pod("a", tcp53),
			modified: pod("a", tcp53, tcp8080, udp53),
			current:  pod("a", tcp53, tcp8080, udp53),
			want:     pod("a", tcp53, tcp8080, udp53),
		},
		{
			name:     "image update keeps live-only fields and restores port order",
			original: pod("a", tcp53, udp53),
			modified: pod("b", tcp53, udp53),
			current:  `{"containers":[{"name":"app","image":"a","ports":[` + udp53 + `,` + tcp53 + `],"resizePolicy":[{"resourceName":"cpu","restartPolicy":"NotRequired"}],"futureField":true}]}`,
			want:     `{"containers":[{"name":"app","image":"b","ports":[` + tcp53 + `,` + udp53 + `],"resizePolicy":[{"resourceName":"cpu","restartPolicy":"NotRequired"}],"futureField":true}]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			current := tc.current
			if current == "" {
				current = tc.original
			}
			got, err := ThreeWayStrategicMerge([]byte(tc.original), []byte(tc.modified), []byte(current), corev1.PodSpec{})
			if err != nil {
				t.Fatal(err)
			}
			var gotDoc, wantDoc any
			if err := json.Unmarshal(got, &gotDoc); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.want), &wantDoc); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotDoc, wantDoc) {
				t.Errorf("got  %s\nwant %s", got, tc.want)
			}
		})
	}
}

func TestStrategicMergeSpecOps(t *testing.T) {
	spec := func(image string) appsv1.DeploymentSpec {
		return appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: image}},
		}}}
	}
	live := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{"replicas": int64(3), "strategy": map[string]any{}, "template": map[string]any{
			"metadata": map[string]any{},
			"spec": map[string]any{"containers": []any{
				map[string]any{"name": "app", "image": "a", "resources": map[string]any{}, "futureField": true},
			}},
		}},
	}}
	ops, err := StrategicMergeSpecOps(live, spec("a"), spec("b"), appsv1.Deployment{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ops.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"path":"/spec","value":{"replicas":3,"strategy":{},"template":{"metadata":{},"spec":{"containers":[{"futureField":true,"image":"b","name":"app","resources":{}}]}}},"op":"replace"}]`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
