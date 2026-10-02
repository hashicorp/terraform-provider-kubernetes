// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

// The update paths of the apps/v1 workloads must leave every list exactly as
// planned. Strategic merge patch identifies container ports only by
// containerPort and topology spread constraints only by topologyKey, while the
// API allows several entries to share those values. These tests drive the real
// Update methods against an API server double that applies each patch with the
// same apimachinery and JSON-patch libraries the kube-apiserver uses.
//
// They test the update path only. The prior state and the plan are flattened
// from API objects rather than planned by Terraform from a configuration, so
// they cannot catch planning problems such as computed attributes of a newly
// added container; the acceptance tests cover those.

type workloadListMergeCase struct {
	name   string
	before func(*corev1.PodSpec)
	after  func(*corev1.PodSpec)
	// live mutates the server object after the prior state was recorded, to
	// model fields written by other actors that Terraform does not manage.
	live func(*corev1.PodSpec)
	// check verifies the server object beyond the planned Pod specification.
	check func(*testing.T, corev1.PodSpec)
	// threeWayOnly marks drift cases that only the StatefulSet update, which
	// merges the plan into the live Pod specification, brings back to the
	// plan. Deployment and DaemonSet send a two-way patch from the prior state
	// to the plan, so a list that drifted after the last refresh and is
	// unchanged in configuration is left as the server has it.
	threeWayOnly bool
}

func port(number int32, protocol corev1.Protocol, name string) corev1.ContainerPort {
	return corev1.ContainerPort{ContainerPort: number, Protocol: protocol, Name: name}
}

func spreadConstraint(key string, when corev1.UnsatisfiableConstraintAction, skew int32) corev1.TopologySpreadConstraint {
	return corev1.TopologySpreadConstraint{
		TopologyKey:       key,
		WhenUnsatisfiable: when,
		MaxSkew:           skew,
		LabelSelector:     &metav1.LabelSelector{MatchLabels: map[string]string{"app": "merge"}},
	}
}

func setPorts(ports ...corev1.ContainerPort) func(*corev1.PodSpec) {
	return func(spec *corev1.PodSpec) { spec.Containers[0].Ports = ports }
}

func setInitPorts(ports ...corev1.ContainerPort) func(*corev1.PodSpec) {
	return func(spec *corev1.PodSpec) {
		spec.InitContainers = []corev1.Container{{
			Name:                     "init",
			Image:                    "busybox:1.36",
			ImagePullPolicy:          corev1.PullIfNotPresent,
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Ports:                    ports,
		}}
	}
}

func setSpread(constraints ...corev1.TopologySpreadConstraint) func(*corev1.PodSpec) {
	return func(spec *corev1.PodSpec) { spec.TopologySpreadConstraints = constraints }
}

func both(fns ...func(*corev1.PodSpec)) func(*corev1.PodSpec) {
	return func(spec *corev1.PodSpec) {
		for _, fn := range fns {
			fn(spec)
		}
	}
}

func workloadListMergeCases() []workloadListMergeCase {
	tcp53 := port(53, corev1.ProtocolTCP, "dns-tcp")
	udp53 := port(53, corev1.ProtocolUDP, "dns-udp")
	httpPort := port(8080, corev1.ProtocolTCP, "http")
	zoneHard := spreadConstraint("topology.kubernetes.io/zone", corev1.DoNotSchedule, 1)
	zoneSoft := spreadConstraint("topology.kubernetes.io/zone", corev1.ScheduleAnyway, 2)
	return []workloadListMergeCase{
		{name: "ports/add-udp-beside-tcp", before: setPorts(tcp53), after: setPorts(tcp53, udp53)},
		{name: "ports/remove-udp-keeps-tcp", before: setPorts(tcp53, udp53), after: setPorts(tcp53)},
		{name: "ports/remove-tcp-keeps-udp", before: setPorts(tcp53, udp53, httpPort), after: setPorts(udp53, httpPort)},
		{name: "ports/rename-udp", before: setPorts(tcp53, udp53), after: setPorts(tcp53, port(53, corev1.ProtocolUDP, "dns-udp2"))},
		{name: "ports/swap-order", before: setPorts(tcp53, udp53), after: setPorts(udp53, tcp53)},
		{name: "ports/udp-to-sctp-beside-tcp", before: setPorts(tcp53, udp53), after: setPorts(tcp53, port(53, corev1.ProtocolSCTP, "dns-udp"))},
		{name: "ports/remove-all", before: setPorts(tcp53, udp53), after: setPorts()},
		{
			name:   "ports/new-container-with-tcp-and-udp",
			before: setPorts(tcp53),
			after: both(setPorts(tcp53), func(spec *corev1.PodSpec) {
				sidecar := spec.Containers[0]
				sidecar.Name = "sidecar"
				sidecar.Ports = []corev1.ContainerPort{tcp53, udp53}
				spec.Containers = append(spec.Containers, sidecar)
			}),
		},
		{name: "init-ports/add-udp-beside-tcp", before: setInitPorts(tcp53), after: setInitPorts(tcp53, udp53)},
		{name: "init-ports/remove-udp-keeps-tcp", before: setInitPorts(tcp53, udp53), after: setInitPorts(tcp53)},
		{name: "spread/add-second-for-same-key", before: setSpread(zoneHard), after: setSpread(zoneHard, zoneSoft)},
		{name: "spread/change-skew-of-second", before: setSpread(zoneHard, zoneSoft), after: setSpread(zoneHard, spreadConstraint("topology.kubernetes.io/zone", corev1.ScheduleAnyway, 3))},
		{name: "spread/remove-first-for-same-key", before: setSpread(zoneHard, zoneSoft), after: setSpread(zoneSoft)},
		{name: "spread/remove-all", before: setSpread(zoneHard, zoneSoft), after: setSpread()},
		{
			name: "host-aliases/duplicate-ip-added",
			before: func(spec *corev1.PodSpec) {
				spec.HostAliases = []corev1.HostAlias{{IP: "10.0.0.1", Hostnames: []string{"a.example"}}}
			},
			after: func(spec *corev1.PodSpec) {
				spec.HostAliases = []corev1.HostAlias{
					{IP: "10.0.0.1", Hostnames: []string{"a.example"}},
					{IP: "10.0.0.1", Hostnames: []string{"b.example"}},
				}
			},
		},
		{
			name:   "env/duplicate-name-added",
			before: func(spec *corev1.PodSpec) { spec.Containers[0].Env = []corev1.EnvVar{{Name: "MODE", Value: "a"}} },
			after: func(spec *corev1.PodSpec) {
				spec.Containers[0].Env = []corev1.EnvVar{{Name: "MODE", Value: "a"}, {Name: "MODE", Value: "b"}}
			},
		},
		{
			// Control: an ordinary update must still keep fields that the
			// provider does not model and that another actor wrote, which is why
			// the update uses a strategic merge rather than a whole-spec replace.
			name:   "control/image-change-keeps-unmanaged-fields",
			before: setPorts(tcp53, udp53),
			after: both(setPorts(tcp53, udp53), func(spec *corev1.PodSpec) {
				spec.Containers[0].Image = "busybox:1.37"
			}),
			live: func(spec *corev1.PodSpec) {
				spec.HostUsers = ptr.To(false)
				spec.Containers[0].ResizePolicy = []corev1.ContainerResizePolicy{{ResourceName: corev1.ResourceCPU, RestartPolicy: corev1.NotRequired}}
			},
			check: func(t *testing.T, spec corev1.PodSpec) {
				if spec.HostUsers == nil || *spec.HostUsers {
					t.Errorf("unmanaged hostUsers was not preserved: %v", spec.HostUsers)
				}
				if len(spec.Containers[0].ResizePolicy) != 1 {
					t.Errorf("unmanaged container resizePolicy was not preserved: %#v", spec.Containers[0].ResizePolicy)
				}
			},
		},
		{
			name:   "control/ports-change-keeps-unmanaged-fields",
			before: setPorts(tcp53),
			after:  setPorts(tcp53, udp53),
			live: func(spec *corev1.PodSpec) {
				spec.HostUsers = ptr.To(false)
				spec.Containers[0].ResizePolicy = []corev1.ContainerResizePolicy{{ResourceName: corev1.ResourceCPU, RestartPolicy: corev1.NotRequired}}
			},
			check: func(t *testing.T, spec corev1.PodSpec) {
				if spec.HostUsers == nil || *spec.HostUsers {
					t.Errorf("unmanaged hostUsers was not preserved: %v", spec.HostUsers)
				}
				if len(spec.Containers[0].ResizePolicy) != 1 {
					t.Errorf("unmanaged container resizePolicy was not preserved: %#v", spec.Containers[0].ResizePolicy)
				}
			},
		},
		// The cases below model a live object that drifted after the prior
		// state was recorded, as with -refresh=false or a saved plan applied
		// after an out-of-band change.
		{
			// A container added out of band and then added in configuration: the
			// server merges the patch into its entry, so the ports must be
			// replaced rather than merged by containerPort.
			name:   "drift/container-added-out-of-band-then-in-config",
			before: setPorts(tcp53),
			after:  both(setPorts(tcp53), addSidecar(tcp53, udp53)),
			live:   addSidecar(udp53),
		},
		{
			name:         "drift/sidecar-removed-out-of-band-while-main-changes",
			before:       both(setPorts(tcp53), addSidecar(httpPort)),
			after:        both(setPorts(tcp53), addSidecar(httpPort), setImage("busybox:1.37")),
			live:         func(spec *corev1.PodSpec) { spec.Containers = spec.Containers[:1] },
			threeWayOnly: true,
		},
		{
			name:         "drift/live-lost-a-planned-port",
			before:       setPorts(tcp53, udp53),
			after:        both(setPorts(tcp53, udp53), setImage("busybox:1.37")),
			live:         setPorts(tcp53),
			threeWayOnly: true,
		},
		{
			name:         "drift/live-gained-a-port",
			before:       setPorts(tcp53),
			after:        both(setPorts(tcp53), setImage("busybox:1.37")),
			live:         setPorts(tcp53, udp53),
			threeWayOnly: true,
		},
		{
			name:         "drift/live-gained-a-port-while-only-a-pod-field-changes",
			before:       setPorts(tcp53),
			after:        both(setPorts(tcp53), func(spec *corev1.PodSpec) { spec.DNSPolicy = corev1.DNSDefault }),
			live:         setPorts(tcp53, udp53),
			threeWayOnly: true,
		},
		{
			name:   "drift/live-gained-a-duplicate-env",
			before: func(spec *corev1.PodSpec) { spec.Containers[0].Env = []corev1.EnvVar{{Name: "MODE", Value: "a"}} },
			after: func(spec *corev1.PodSpec) {
				spec.Containers[0].Env = []corev1.EnvVar{{Name: "MODE", Value: "a"}}
				spec.Containers[0].Image = "busybox:1.37"
			},
			live: func(spec *corev1.PodSpec) {
				spec.Containers[0].Env = append(spec.Containers[0].Env, corev1.EnvVar{Name: "MODE", Value: "b"})
			},
			threeWayOnly: true,
		},
	}
}

func setImage(image string) func(*corev1.PodSpec) {
	return func(spec *corev1.PodSpec) { spec.Containers[0].Image = image }
}

func addSidecar(ports ...corev1.ContainerPort) func(*corev1.PodSpec) {
	return func(spec *corev1.PodSpec) {
		sidecar := *spec.Containers[0].DeepCopy()
		sidecar.Name = "sidecar"
		sidecar.Image = "busybox:1.36"
		sidecar.Env = nil
		sidecar.Ports = ports
		spec.Containers = append(spec.Containers, sidecar)
	}
}

func listMergeBasePodSpec() corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:                     "main",
			Image:                    "busybox:1.36",
			ImagePullPolicy:          corev1.PullIfNotPresent,
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		}},
		RestartPolicy:                 corev1.RestartPolicyAlways,
		DNSPolicy:                     corev1.DNSClusterFirst,
		SchedulerName:                 corev1.DefaultSchedulerName,
		SecurityContext:               &corev1.PodSecurityContext{},
		TerminationGracePeriodSeconds: ptr.To(int64(30)),
	}
}

func listMergePodSpec(mutate func(*corev1.PodSpec)) corev1.PodSpec {
	spec := listMergeBasePodSpec()
	if mutate != nil {
		mutate(&spec)
	}
	return spec
}

// listMergeAPI is a single-object API server double. It applies JSON and
// strategic merge patches with the libraries used by kube-apiserver and
// rejects payloads that leave patch directives or unknown fields behind.
type listMergeAPI struct {
	t        *testing.T
	mu       sync.Mutex
	path     string
	newObj   func() any
	object   []byte
	patches  []string
	revision int
}

func (api *listMergeAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()
	if req.URL.Path != api.path {
		api.t.Errorf("unexpected API request: %s %s", req.Method, req.URL.Path)
		writeListMergeStatus(w, http.StatusNotFound, "NotFound", "unexpected path")
		return
	}
	switch req.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(api.object)
	case http.MethodPatch:
		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeListMergeStatus(w, http.StatusBadRequest, "BadRequest", err.Error())
			return
		}
		contentType := req.Header.Get("Content-Type")
		api.patches = append(api.patches, contentType+" "+string(body))
		var patched []byte
		switch {
		case strings.HasPrefix(contentType, "application/strategic-merge-patch+json"):
			patched, err = strategicpatch.StrategicMergePatch(api.object, body, api.newObj())
		case strings.HasPrefix(contentType, "application/json-patch+json"):
			var operations jsonpatch.Patch
			operations, err = jsonpatch.DecodePatch(body)
			if err == nil {
				patched, err = operations.Apply(api.object)
			}
		default:
			err = fmt.Errorf("unsupported patch type %q", contentType)
		}
		if err != nil {
			writeListMergeStatus(w, http.StatusUnprocessableEntity, "Invalid", err.Error())
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(patched))
		decoder.DisallowUnknownFields()
		typed := api.newObj()
		if err := decoder.Decode(typed); err != nil {
			writeListMergeStatus(w, http.StatusUnprocessableEntity, "Invalid", "patched object: "+err.Error())
			return
		}
		api.revision++
		accessor := typed.(metav1.Object)
		accessor.SetResourceVersion(strconv.Itoa(api.revision))
		accessor.SetGeneration(accessor.GetGeneration() + 1)
		api.object, _ = json.Marshal(typed)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(api.object)
	default:
		api.t.Errorf("unexpected API request: %s %s", req.Method, req.URL.Path)
		writeListMergeStatus(w, http.StatusMethodNotAllowed, "MethodNotAllowed", req.Method)
	}
}

func writeListMergeStatus(w http.ResponseWriter, code int, reason, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"kind": "Status", "apiVersion": "v1", "status": "Failure", "reason": reason, "message": message, "code": code,
	})
}

func (api *listMergeAPI) set(t *testing.T, object any) {
	t.Helper()
	api.mu.Lock()
	defer api.mu.Unlock()
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	api.object = encoded
}

func (api *listMergeAPI) get(t *testing.T, into any) {
	t.Helper()
	api.mu.Lock()
	defer api.mu.Unlock()
	if err := json.Unmarshal(api.object, into); err != nil {
		t.Fatal(err)
	}
}

func startListMergeAPI(t *testing.T, path string, newObj func() any) (*listMergeAPI, deploymentReadClientsets) {
	t.Helper()
	api := &listMergeAPI{t: t, path: path, newObj: newObj}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)
	client, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	return api, deploymentReadClientsets{client: client}
}

func listMergeObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: "merge", Namespace: "default", UID: "uid-merge", Generation: 1, ResourceVersion: "1",
		Labels: map[string]string{"app": "merge"},
	}
}

func listMergeTemplate(spec corev1.PodSpec) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "merge"}},
		Spec:       spec,
	}
}

type listMergeSchemaResource interface {
	resource.Resource
	resource.ResourceWithIdentity
}

// listMergeSchemas holds a resource's schemas, built once per test because
// building them dominates the cost of each case.
type listMergeSchemas struct {
	resource fwschema.Schema
	identity identityschema.Schema
}

func newListMergeSchemas(t *testing.T, r listMergeSchemaResource) listMergeSchemas {
	t.Helper()
	ctx := context.Background()
	schemaResponse := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	identityResponse := resource.IdentitySchemaResponse{}
	r.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResponse)
	if schemaResponse.Diagnostics.HasError() || identityResponse.Diagnostics.HasError() {
		t.Fatal(schemaResponse.Diagnostics, identityResponse.Diagnostics)
	}
	return listMergeSchemas{resource: schemaResponse.Schema, identity: identityResponse.IdentitySchema}
}

func listMergeRequest(t *testing.T, schemas listMergeSchemas, state, plan map[string]attr.Value) (resource.UpdateRequest, *resource.UpdateResponse) {
	t.Helper()
	ctx := context.Background()
	build := func(values map[string]attr.Value) tftypes.Value {
		target := tfsdk.State{
			Schema: schemas.resource,
			Raw:    tftypes.NewValue(schemas.resource.Type().TerraformType(ctx), nil),
		}
		for name, value := range values {
			if d := target.SetAttribute(ctx, path.Root(name), value); d.HasError() {
				t.Fatalf("setting %s: %v", name, d)
			}
		}
		return target.Raw
	}
	stateRaw, planRaw := build(state), build(plan)
	identity := &tfsdk.ResourceIdentity{
		Schema: schemas.identity,
		Raw:    tftypes.NewValue(schemas.identity.Type().TerraformType(ctx), nil),
	}
	request := resource.UpdateRequest{
		State:  tfsdk.State{Schema: schemas.resource, Raw: stateRaw},
		Plan:   tfsdk.Plan{Schema: schemas.resource, Raw: planRaw},
		Config: tfsdk.Config{Schema: schemas.resource, Raw: planRaw},
	}
	response := &resource.UpdateResponse{
		State:    tfsdk.State{Schema: schemas.resource, Raw: planRaw},
		Identity: identity,
	}
	return request, response
}

// assertListMergeResult checks the server object against the planned Pod
// specification and checks that the new state agrees with the plan, which is
// what Terraform core verifies after apply.
func assertListMergeResult(t *testing.T, api *listMergeAPI, tc workloadListMergeCase, got, want corev1.PodSpec, request resource.UpdateRequest, response *resource.UpdateResponse) {
	t.Helper()
	if response.Diagnostics.HasError() {
		t.Fatalf("update failed: %v\npatches: %s", response.Diagnostics, strings.Join(api.patches, "\n"))
	}
	if len(got.Containers) != len(want.Containers) {
		t.Fatalf("containers = %#v, want %#v", got.Containers, want.Containers)
	}
	for i := range want.Containers {
		if !reflect.DeepEqual(got.Containers[i].Ports, want.Containers[i].Ports) {
			t.Errorf("container %q ports = %#v\nwant %#v\npatches: %s", want.Containers[i].Name, got.Containers[i].Ports, want.Containers[i].Ports, strings.Join(api.patches, "\n"))
		}
		if !reflect.DeepEqual(got.Containers[i].Env, want.Containers[i].Env) {
			t.Errorf("container %q env = %#v\nwant %#v\npatches: %s", want.Containers[i].Name, got.Containers[i].Env, want.Containers[i].Env, strings.Join(api.patches, "\n"))
		}
		if got.Containers[i].Image != want.Containers[i].Image {
			t.Errorf("container %q image = %q, want %q", want.Containers[i].Name, got.Containers[i].Image, want.Containers[i].Image)
		}
	}
	if len(got.InitContainers) != len(want.InitContainers) {
		t.Fatalf("init containers = %#v, want %#v", got.InitContainers, want.InitContainers)
	}
	for i := range want.InitContainers {
		if !reflect.DeepEqual(got.InitContainers[i].Ports, want.InitContainers[i].Ports) {
			t.Errorf("init container %q ports = %#v\nwant %#v\npatches: %s", want.InitContainers[i].Name, got.InitContainers[i].Ports, want.InitContainers[i].Ports, strings.Join(api.patches, "\n"))
		}
	}
	if !reflect.DeepEqual(got.HostAliases, want.HostAliases) {
		t.Errorf("host aliases = %#v\nwant %#v\npatches: %s", got.HostAliases, want.HostAliases, strings.Join(api.patches, "\n"))
	}
	if !reflect.DeepEqual(got.TopologySpreadConstraints, want.TopologySpreadConstraints) {
		t.Errorf("topology spread constraints = %#v\nwant %#v\npatches: %s", got.TopologySpreadConstraints, want.TopologySpreadConstraints, strings.Join(api.patches, "\n"))
	}
	if tc.check != nil {
		tc.check(t, got)
	}

	ctx := context.Background()
	var planned, applied types.List
	if d := request.Plan.GetAttribute(ctx, path.Root("spec"), &planned); d.HasError() {
		t.Fatal(d)
	}
	if d := response.State.GetAttribute(ctx, path.Root("spec"), &applied); d.HasError() {
		t.Fatal(d)
	}
	if !planned.Equal(applied) {
		t.Errorf("new state spec differs from the plan (Terraform reports an inconsistent result):\nplan:  %s\nstate: %s", planned, applied)
	}
}

func TestDeploymentV1UpdateListMergeKeys(t *testing.T) {
	schemas := newListMergeSchemas(t, &DeploymentV1{})
	for _, tc := range workloadListMergeCases() {
		if tc.threeWayOnly {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			api, clients := startListMergeAPI(t, "/apis/apps/v1/namespaces/default/deployments/merge", func() any { return &appsv1.Deployment{} })
			object := func(spec corev1.PodSpec) *appsv1.Deployment {
				return &appsv1.Deployment{
					TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
					ObjectMeta: listMergeObjectMeta(),
					Spec: appsv1.DeploymentSpec{
						Replicas:                ptr.To(int32(1)),
						Selector:                &metav1.LabelSelector{MatchLabels: map[string]string{"app": "merge"}},
						Template:                listMergeTemplate(spec),
						Strategy:                appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
						RevisionHistoryLimit:    ptr.To(int32(10)),
						ProgressDeadlineSeconds: ptr.To(int32(600)),
					},
				}
			}
			before, after := object(listMergePodSpec(tc.before)), object(listMergePodSpec(tc.after))
			state, d := deploymentModelFromObject(ctx, before, DeploymentV1Model{WaitForRollout: types.BoolValue(false)}, clients)
			if d.HasError() {
				t.Fatal(d)
			}
			plan, d := deploymentModelFromObject(ctx, after, state, clients)
			if d.HasError() {
				t.Fatal(d)
			}
			live := before.DeepCopy()
			if tc.live != nil {
				tc.live(&live.Spec.Template.Spec)
			}
			api.set(t, live)

			r := &DeploymentV1{SDKv2Meta: func() any { return clients }}
			request, response := listMergeRequest(t, schemas, listMergeValues(t, schemas, state.ID, state.Metadata, state.Spec), listMergeValues(t, schemas, plan.ID, plan.Metadata, plan.Spec))
			r.Update(ctx, request, response)

			var got appsv1.Deployment
			api.get(t, &got)
			assertListMergeResult(t, api, tc, got.Spec.Template.Spec, after.Spec.Template.Spec, request, response)
		})
	}
}

func TestDaemonSetV1UpdateListMergeKeys(t *testing.T) {
	schemas := newListMergeSchemas(t, &DaemonSetV1{})
	for _, tc := range workloadListMergeCases() {
		if tc.threeWayOnly {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			api, clients := startListMergeAPI(t, "/apis/apps/v1/namespaces/default/daemonsets/merge", func() any { return &appsv1.DaemonSet{} })
			object := func(spec corev1.PodSpec) *appsv1.DaemonSet {
				return &appsv1.DaemonSet{
					TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
					ObjectMeta: listMergeObjectMeta(),
					Spec: appsv1.DaemonSetSpec{
						Selector:             &metav1.LabelSelector{MatchLabels: map[string]string{"app": "merge"}},
						Template:             listMergeTemplate(spec),
						UpdateStrategy:       appsv1.DaemonSetUpdateStrategy{Type: appsv1.OnDeleteDaemonSetStrategyType},
						RevisionHistoryLimit: ptr.To(int32(10)),
					},
				}
			}
			before, after := object(listMergePodSpec(tc.before)), object(listMergePodSpec(tc.after))
			r := &DaemonSetV1{SDKv2Meta: func() any { return clients }}
			var d diag.Diagnostics
			state := r.daemonSetStateFromObject(ctx, DaemonSetV1Model{WaitForRollout: types.BoolValue(false)}, before, clients, &d)
			plan := r.daemonSetStateFromObject(ctx, state, after, clients, &d)
			if d.HasError() {
				t.Fatal(d)
			}
			live := before.DeepCopy()
			if tc.live != nil {
				tc.live(&live.Spec.Template.Spec)
			}
			api.set(t, live)

			request, response := listMergeRequest(t, schemas, listMergeValues(t, schemas, state.ID, state.Metadata, state.Spec), listMergeValues(t, schemas, plan.ID, plan.Metadata, plan.Spec))
			r.Update(ctx, request, response)

			var got appsv1.DaemonSet
			api.get(t, &got)
			assertListMergeResult(t, api, tc, got.Spec.Template.Spec, after.Spec.Template.Spec, request, response)
		})
	}
}

func TestStatefulSetV1UpdateListMergeKeys(t *testing.T) {
	schemas := newListMergeSchemas(t, &StatefulSetV1{})
	for _, tc := range workloadListMergeCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			api, clients := startListMergeAPI(t, "/apis/apps/v1/namespaces/default/statefulsets/merge", func() any { return &appsv1.StatefulSet{} })
			object := func(spec corev1.PodSpec) *appsv1.StatefulSet {
				return &appsv1.StatefulSet{
					TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"},
					ObjectMeta: listMergeObjectMeta(),
					Spec: appsv1.StatefulSetSpec{
						Replicas:             ptr.To(int32(1)),
						Selector:             &metav1.LabelSelector{MatchLabels: map[string]string{"app": "merge"}},
						Template:             listMergeTemplate(spec),
						ServiceName:          "merge",
						PodManagementPolicy:  appsv1.OrderedReadyPodManagement,
						UpdateStrategy:       appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType},
						RevisionHistoryLimit: ptr.To(int32(10)),
					},
				}
			}
			before, after := object(listMergePodSpec(tc.before)), object(listMergePodSpec(tc.after))
			r := &StatefulSetV1{SDKv2Meta: func() any { return clients }}
			state, _, d := r.flattenStateFromObject(ctx, clients, StatefulSetV1Model{WaitForRollout: types.BoolValue(false)}, before, false)
			if d.HasError() {
				t.Fatal(d)
			}
			plan, _, d := r.flattenStateFromObject(ctx, clients, state, after, false)
			if d.HasError() {
				t.Fatal(d)
			}
			live := before.DeepCopy()
			if tc.live != nil {
				tc.live(&live.Spec.Template.Spec)
			}
			api.set(t, live)

			request, response := listMergeRequest(t, schemas, listMergeValues(t, schemas, state.ID, state.Metadata, state.Spec), listMergeValues(t, schemas, plan.ID, plan.Metadata, plan.Spec))
			r.Update(ctx, request, response)

			var got appsv1.StatefulSet
			api.get(t, &got)
			assertListMergeResult(t, api, tc, got.Spec.Template.Spec, after.Spec.Template.Spec, request, response)
		})
	}
}

// listMergeValues converts a resource model into root attribute values using
// the resource schema, so the fixtures follow schema changes automatically.
func listMergeValues(t *testing.T, schemas listMergeSchemas, id types.String, metadata []common.NamespacedMetadataModel, spec any) map[string]attr.Value {
	t.Helper()
	ctx := context.Background()
	attributeType := func(name string) attr.Type {
		if block, ok := schemas.resource.GetBlocks()[name]; ok {
			return block.Type()
		}
		return schemas.resource.GetAttributes()[name].GetType()
	}
	metadataValue, d := types.ListValueFrom(ctx, attributeType("metadata").(types.ListType).ElemType, metadata)
	if d.HasError() {
		t.Fatal(d)
	}
	specValue, ok := spec.(attr.Value)
	if !ok {
		specValue, d = types.ListValueFrom(ctx, attributeType("spec").(types.ListType).ElemType, spec)
		if d.HasError() {
			t.Fatal(d)
		}
	}
	return map[string]attr.Value{
		"id": id, "metadata": metadataValue, "spec": specValue, "wait_for_rollout": types.BoolValue(false),
	}
}
