// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1_test

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
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclparse"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const networkingV1SDKRelease = "3.2.1"

var networkingV1Factories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

var networkingV1Types = []string{
	"kubernetes_ingress_v1",
	"kubernetes_ingress_class_v1",
	"kubernetes_network_policy_v1",
}

// This opt-in test downloads the pinned release, but all Kubernetes traffic is
// confined to an in-process HTTP server. It never uses acceptance credentials.
// These cases cover the actual mux and a persisted SDK refresh. The resource
// unit tests cover individual omitted/empty collection upgrade permutations.
func TestNetworkingV1CoreIngressClassUpgrade(t *testing.T) {
	networkingCoreUpgrade(t, "kubernetes_ingress_class_v1", []string{"minimal", "full"})
}

func TestNetworkingV1CoreNetworkPolicyUpgrade(t *testing.T) {
	networkingCoreUpgrade(t, "kubernetes_network_policy_v1", []string{"full"})
}

type networkingCoreObject interface {
	runtime.Object
	metav1.Object
}

func networkingCoreUpgrade(t *testing.T, resourceType string, variants []string) {
	t.Helper()
	if os.Getenv("KUBE_NETWORKING_CORE_TEST") != "1" {
		t.Skip("set KUBE_NETWORKING_CORE_TEST=1 for the local fake-API Terraform Core upgrade test")
	}
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("Terraform CLI is required for the local fake-API upgrade test")
	}
	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			// Keep plugin Unix socket paths below macOS's limit while retaining
			// every temporary file inside this repository.
			privateDir, err := filepath.Abs(filepath.Join("../../../..", ".nc-"+acctest.RandString(6)))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(privateDir, 0700); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.RemoveAll(privateDir); err != nil {
					t.Error(err)
				}
			})
			for _, entry := range os.Environ() {
				key, _, _ := strings.Cut(entry, "=")
				if strings.HasPrefix(key, "TF_") || strings.HasPrefix(key, "TFE_") ||
					strings.HasPrefix(key, "KUBE") || strings.HasPrefix(key, "KUBERNETES_") {
					t.Setenv(key, "")
				}
			}
			t.Setenv("HOME", privateDir)
			t.Setenv("TMPDIR", privateDir)
			t.Setenv("TF_ACC_TEMP_DIR", privateDir)
			t.Setenv("TF_ACC_TERRAFORM_PATH", terraformPath)
			t.Setenv("TF_IN_AUTOMATION", "1")
			t.Setenv("CHECKPOINT_DISABLE", "1")
			cliConfig := filepath.Join(privateDir, "terraform.rc")
			if err := os.WriteFile(cliConfig, []byte("disable_checkpoint = true\nprovider_installation {\n  direct {}\n}\n"), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("TF_CLI_CONFIG_FILE", cliConfig)
			kubeConfig := filepath.Join(privateDir, "kubeconfig")
			if err := os.WriteFile(kubeConfig, []byte("apiVersion: v1\nkind: Config\nclusters: []\ncontexts: []\nusers: []\ncurrent-context: \"\"\n"), 0600); err != nil {
				t.Fatal(err)
			}

			name := "tf-networking-core-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			var mu sync.Mutex
			var object networkingCoreObject
			basePath := "/apis/networking.k8s.io/v1/ingressclasses"
			if resourceType == "kubernetes_network_policy_v1" {
				basePath = "/apis/networking.k8s.io/v1/namespaces/default/networkpolicies"
			}
			creates, deletes := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if req.URL.Path != basePath && req.URL.Path != basePath+"/"+name {
					t.Errorf("unexpected fake API request: %s %s", req.Method, req.URL.Path)
					http.Error(w, "unexpected fake API endpoint", http.StatusNotFound)
					return
				}
				notFound := func() {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","reason":"NotFound","code":404}`)
				}
				switch req.Method {
				case http.MethodPost:
					if req.URL.Path != basePath || object != nil {
						t.Errorf("unexpected create or replacement: %s", req.URL.Path)
						http.Error(w, "unexpected create", http.StatusConflict)
						return
					}
					raw, err := io.ReadAll(req.Body)
					if err != nil {
						t.Error(err)
						http.Error(w, "read request failed", http.StatusBadRequest)
						return
					}
					var created networkingCoreObject = &networking.IngressClass{}
					if resourceType == "kubernetes_network_policy_v1" {
						created = &networking.NetworkPolicy{}
					}
					if _, _, err := scheme.Codecs.UniversalDeserializer().Decode(raw, nil, created); err != nil {
						t.Error(err)
						http.Error(w, "decode request failed", http.StatusBadRequest)
						return
					}
					creates++
					created.GetObjectKind().SetGroupVersionKind(networking.SchemeGroupVersion.WithKind(networkingKind(resourceType)))
					created.SetUID(types.UID(fmt.Sprintf("fake-uid-%d", creates)))
					created.SetResourceVersion("1")
					created.SetGeneration(1)
					if class, ok := created.(*networking.IngressClass); ok && class.Spec.Parameters != nil && class.Spec.Parameters.Scope == nil {
						scope := "Cluster"
						class.Spec.Parameters.Scope = &scope
					}
					if resourceType == "kubernetes_network_policy_v1" {
						created.SetNamespace("default")
					}
					object = created
				case http.MethodGet:
					if object == nil {
						notFound()
						return
					}
				case http.MethodDelete:
					if object == nil {
						notFound()
						return
					}
					deletes++
					object = nil
					fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Success"}`)
					return
				default:
					t.Errorf("unchanged upgrade attempted a write: %s %s", req.Method, req.URL.Path)
					http.Error(w, "unexpected mutation", http.StatusMethodNotAllowed)
					return
				}
				if req.Method == http.MethodPost {
					raw, err := json.Marshal(object)
					if err != nil {
						t.Error(err)
						http.Error(w, "encode object failed", http.StatusInternalServerError)
						return
					}
					t.Logf("Fake API POST content_type=%q decoded=%T; POST and unchanged GET JSON: %s", req.Header.Get("Content-Type"), object, raw)
				}
				if err := json.NewEncoder(w).Encode(object); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			defer func() {
				mu.Lock()
				defer mu.Unlock()
				if creates != 1 || deletes != 1 || object != nil {
					t.Errorf("unexpected fake lifecycle: creates=%d deletes=%d object=%#v", creates, deletes, object)
				}
			}()
			// Import verification constructs a separate provider configuration.
			// Pin its defaults to the same fake API rather than localhost:80.
			t.Setenv("KUBE_HOST", server.URL)
			t.Setenv("KUBE_CONFIG_PATH", kubeConfig)
			t.Setenv("KUBECONFIG", kubeConfig)
			checkProvider := kubernetes.Provider()
			if diagnostics := checkProvider.Configure(context.Background(), sdkterraform.NewResourceConfigRaw(map[string]interface{}{
				"host": server.URL, "config_path": kubeConfig,
			})); diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			config := fmt.Sprintf(`
provider "kubernetes" {
  host        = %q
  config_path = %q
}
`, server.URL, kubeConfig) + networkingV1Config(resourceType, name, variant)
			var remote networkingRemoteSnapshot
			var state networkingStateSnapshot
			var writes networkingWriteTracker
			factories := networkingObservedFactories(&writes)
			resource.Test(t, resource.TestCase{
				IsUnitTest: true,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_12_0),
				},
				CheckDestroy: networkingCheckDestroy(checkProvider, resourceType, name),
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: networkingV1SDKRelease},
						},
						Config: config,
						ConfigStateChecks: []statecheck.StateCheck{
							networkingCaptureState{address: resourceType + ".test", snapshot: &state, report: t, reportStage: "created"},
						},
					},
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: networkingV1SDKRelease},
						},
						Config: config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: networkingCheckRemote(checkProvider, resourceType, name, &remote, true),
						ConfigStateChecks: []statecheck.StateCheck{
							networkingCaptureState{address: resourceType + ".test", snapshot: &state, report: t, reportStage: "refreshed"},
						},
					},
					{
						ProtoV6ProviderFactories: factories,
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan(), networkingCorePlanReport{t: t}},
						},
						Check: networkingCheckRemote(checkProvider, resourceType, name, &remote, false),
						ConfigStateChecks: []statecheck.StateCheck{
							networkingCaptureState{address: resourceType + ".test", snapshot: &state, compare: true},
							&writes,
						},
					},
					{
						ProtoV6ProviderFactories: factories,
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: networkingCheckRemote(checkProvider, resourceType, name, &remote, false),
						ConfigStateChecks: []statecheck.StateCheck{
							networkingCaptureState{address: resourceType + ".test", snapshot: &state, compare: true},
							&writes,
						},
					},
					{
						ResourceName:             resourceType + ".test",
						ProtoV6ProviderFactories: factories,
						ImportState:              true,
						ImportStateVerify:        true,
					},
				},
			})
		})
	}
}

type networkingCorePlanReport struct {
	t *testing.T
}

type networkingEmptyMetadataPlan struct {
	address       string
	policyVariant string
}

func (c networkingEmptyMetadataPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if req.Plan == nil {
		resp.Error = fmt.Errorf("plan is missing")
		return
	}
	if len(req.Plan.ResourceDrift) != 0 {
		resp.Error = fmt.Errorf("metadata normalization must not conceal refresh drift")
		return
	}
	for name, output := range req.Plan.OutputChanges {
		if !output.Actions.NoOp() {
			resp.Error = fmt.Errorf("unexpected output change: %s", name)
			return
		}
	}
	var found *tfjson.ResourceChange
	for _, change := range req.Plan.ResourceChanges {
		if change.Address == c.address {
			found = change
		} else if change.Change != nil && !change.Change.Actions.NoOp() {
			resp.Error = fmt.Errorf("unexpected resource change: %s", change.Address)
			return
		}
	}
	if found == nil || found.Change == nil {
		resp.Error = fmt.Errorf("resource change %s is missing", c.address)
		return
	}
	before, ok := found.Change.Before.(map[string]interface{})
	if !ok {
		resp.Error = fmt.Errorf("resource %s has no previous object", c.address)
		return
	}
	expected, err := networkingEmptyMetadataValues(before)
	if err == nil && c.policyVariant != "" {
		expected, err = networkingEmptyPolicySelectorValues(expected, c.policyVariant)
	}
	if err != nil {
		resp.Error = err
		return
	}
	normalizing := !reflect.DeepEqual(expected, before)
	if normalizing {
		if !reflect.DeepEqual(found.Change.Actions, tfjson.Actions{tfjson.ActionUpdate}) {
			resp.Error = fmt.Errorf("explicit-empty normalization must be an in-place update, got %v", found.Change.Actions)
			return
		}
	} else if !found.Change.Actions.NoOp() {
		resp.Error = fmt.Errorf("already-empty values must have a no-op plan, got %v", found.Change.Actions)
		return
	}
	if err := networkingMetadataUnknowns(found.Change.AfterUnknown, "", normalizing, c.address, expected, found.Change.After); err != nil {
		resp.Error = err
		return
	}
	if diff := cmp.Diff(expected, found.Change.After); diff != "" {
		resp.Error = fmt.Errorf("plan changed more than the approved explicit-empty values (-allowed +actual):\n%s", diff)
	}
}

func networkingMetadataUnknowns(value interface{}, path string, normalizing bool, address string, expected, after interface{}) error {
	switch value := value.(type) {
	case nil:
	case bool:
		if !value {
			return nil
		}
		allowed := path == "metadata[0].generation" || path == "metadata[0].resource_version" ||
			(strings.HasPrefix(address, "kubernetes_ingress_v1.") && (path == "spec[0].ingress_class_name" || path == "status")) ||
			(strings.HasPrefix(address, "kubernetes_ingress_class_v1.") && path == "spec[0].parameters[0].scope")
		if !normalizing || !allowed {
			return fmt.Errorf("unexpected planned unknown at %s", path)
		}
		if after != nil {
			return fmt.Errorf("planned unknown at %s also has a known value", path)
		}
	case map[string]interface{}:
		if len(value) == 0 {
			return nil
		}
		expectedFields, expectedOK := expected.(map[string]interface{})
		afterFields, afterOK := after.(map[string]interface{})
		if !expectedOK || !afterOK {
			return fmt.Errorf("planned unknown object at %s has no matching values", path)
		}
		for key, child := range value {
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			if err := networkingMetadataUnknowns(child, childPath, normalizing, address, expectedFields[key], afterFields[key]); err != nil {
				return err
			}
			if unknown, ok := child.(bool); ok && unknown {
				// Core can either omit an unknown attribute or emit a null placeholder.
				if _, present := afterFields[key]; present {
					expectedFields[key] = nil
				} else {
					delete(expectedFields, key)
				}
			}
		}
	case []interface{}:
		if len(value) == 0 {
			return nil
		}
		expectedElements, expectedOK := expected.([]interface{})
		afterElements, afterOK := after.([]interface{})
		if !expectedOK || !afterOK || len(value) != len(expectedElements) || len(value) != len(afterElements) {
			return fmt.Errorf("planned unknown list at %s has no matching values", path)
		}
		for i, child := range value {
			if err := networkingMetadataUnknowns(child, fmt.Sprintf("%s[%d]", path, i), normalizing, address, expectedElements[i], afterElements[i]); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("invalid planned unknown at %s", path)
	}
	return nil
}

func networkingCloneValue(value interface{}) interface{} {
	switch value := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(value))
		for key, child := range value {
			result[key] = networkingCloneValue(child)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(value))
		for i, child := range value {
			result[i] = networkingCloneValue(child)
		}
		return result
	default:
		return value
	}
}

func networkingEmptyMetadataValues(values map[string]interface{}) (map[string]interface{}, error) {
	expected := networkingCloneValue(values).(map[string]interface{})
	metadata, ok := expected["metadata"].([]interface{})
	if !ok || len(metadata) != 1 {
		return nil, fmt.Errorf("expected one metadata block")
	}
	fields, ok := metadata[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("metadata block is missing")
	}
	for _, key := range []string{"labels", "annotations"} {
		if value := fields[key]; value != nil {
			entries, ok := value.(map[string]interface{})
			if !ok || len(entries) != 0 {
				return nil, fmt.Errorf("metadata.%s is not null or empty: %#v", key, value)
			}
		}
		fields[key] = map[string]interface{}{}
	}
	return expected, nil
}

func networkingEmptyPolicySelectorValues(values map[string]interface{}, variant string) (map[string]interface{}, error) {
	if variant != "empty" && variant != "selector_maps_empty" && variant != "selector_values_empty" {
		return nil, fmt.Errorf("no selector normalization is approved for fixture %q", variant)
	}
	expected := networkingCloneValue(values).(map[string]interface{})
	paths := [][]interface{}{
		{"spec", 0, "pod_selector", 0},
		{"spec", 0, "ingress", 0, "from", 0, "namespace_selector", 0},
		{"spec", 0, "ingress", 0, "from", 0, "pod_selector", 0},
		{"spec", 0, "egress", 0, "to", 0, "namespace_selector", 0},
		{"spec", 0, "egress", 0, "to", 0, "pod_selector", 0},
	}
	if variant == "empty" {
		paths = paths[:1]
	}
	for _, path := range paths {
		var value interface{} = expected
		for _, step := range path {
			switch step := step.(type) {
			case string:
				fields, ok := value.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("selector fixture path %v is missing", path)
				}
				value = fields[step]
			case int:
				blocks, ok := value.([]interface{})
				if !ok || len(blocks) <= step {
					return nil, fmt.Errorf("selector fixture path %v is missing", path)
				}
				value = blocks[step]
			}
		}
		selector, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("selector fixture path %v is not an object", path)
		}
		if variant != "selector_values_empty" {
			if labels := selector["match_labels"]; labels != nil {
				entries, ok := labels.(map[string]interface{})
				if !ok || len(entries) != 0 {
					return nil, fmt.Errorf("match_labels at %v is not null or empty", path)
				}
			}
			selector["match_labels"] = map[string]interface{}{}
		}
		if variant != "selector_maps_empty" {
			expressions, ok := selector["match_expressions"].([]interface{})
			count := 2
			if variant == "empty" {
				count = 1
			}
			if !ok || len(expressions) != count {
				return nil, fmt.Errorf("expected %d explicit-empty expressions at %v", count, path)
			}
			for _, value := range expressions {
				expression, ok := value.(map[string]interface{})
				if !ok || (expression["operator"] != "Exists" && expression["operator"] != "DoesNotExist") {
					return nil, fmt.Errorf("expression at %v does not use Exists or DoesNotExist", path)
				}
				if value := expression["values"]; value != nil {
					entries, ok := value.([]interface{})
					if !ok || len(entries) != 0 {
						return nil, fmt.Errorf("expression values at %v are not null or empty", path)
					}
				}
				expression["values"] = []interface{}{}
			}
		}
	}
	return expected, nil
}

func TestNetworkingV1EmptyPolicySelectorsPlanBoundary(t *testing.T) {
	const address = "kubernetes_network_policy_v1.test"
	for _, variant := range []string{"empty", "selector_maps_empty", "selector_values_empty"} {
		t.Run(variant, func(t *testing.T) {
			selector := map[string]interface{}{"match_labels": nil, "match_expressions": nil}
			if variant != "selector_maps_empty" {
				expressions := []interface{}{
					map[string]interface{}{"key": "absent", "operator": "DoesNotExist", "values": nil},
				}
				if variant == "selector_values_empty" {
					selector["match_labels"] = map[string]interface{}{"app": "migration"}
					expressions = append(expressions, map[string]interface{}{"key": "present", "operator": "Exists", "values": nil})
				}
				selector["match_expressions"] = expressions
			}
			peer := map[string]interface{}{
				"namespace_selector": []interface{}{networkingCloneValue(selector)},
				"pod_selector":       []interface{}{networkingCloneValue(selector)},
			}
			before := map[string]interface{}{
				"id": "default/migration",
				"metadata": []interface{}{map[string]interface{}{
					"labels": nil, "annotations": nil, "uid": "uid-1", "generation": float64(1), "resource_version": "1",
				}},
				"spec": []interface{}{map[string]interface{}{
					"pod_selector": []interface{}{selector},
					"policy_types": []interface{}{"Ingress", "Egress"},
					"ingress":      []interface{}{map[string]interface{}{"from": []interface{}{networkingCloneValue(peer)}}},
					"egress":       []interface{}{map[string]interface{}{"to": []interface{}{networkingCloneValue(peer)}}},
				}},
			}
			expected, err := networkingEmptyMetadataValues(before)
			if err == nil {
				expected, err = networkingEmptyPolicySelectorValues(expected, variant)
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, test := range []string{"allowed", "replacement", "unrelated_spec", "unapproved_collection", "omitted_fixture"} {
				t.Run(test, func(t *testing.T) {
					after := networkingCloneValue(expected).(map[string]interface{})
					change := &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionUpdate}, Before: before, After: after}
					check := networkingEmptyMetadataPlan{address: address, policyVariant: variant}
					spec := after["spec"].([]interface{})[0].(map[string]interface{})
					switch test {
					case "replacement":
						change.Actions = tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}
					case "unrelated_spec":
						spec["policy_types"] = []interface{}{"Ingress"}
					case "unapproved_collection":
						if variant == "selector_maps_empty" {
							spec["pod_selector"].([]interface{})[0].(map[string]interface{})["match_expressions"] = []interface{}{}
						} else {
							spec["egress"].([]interface{})[0].(map[string]interface{})["to"].([]interface{})[0].(map[string]interface{})["pod_selector"].([]interface{})[0].(map[string]interface{})["match_labels"] = map[string]interface{}{}
						}
					case "omitted_fixture":
						check.policyVariant = "selector_maps_omitted"
					}
					plan := &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{{Address: address, Change: change}}}
					var response plancheck.CheckPlanResponse
					check.CheckPlan(context.Background(), plancheck.CheckPlanRequest{Plan: plan}, &response)
					if (response.Error != nil) != (test != "allowed") {
						t.Fatalf("unexpected normalization boundary result: %v", response.Error)
					}
				})
			}
			if _, err := networkingEmptyPolicySelectorValues(expected, variant); err != nil {
				t.Fatalf("already-normalized values must be accepted: %s", err)
			}
		})
	}
}

func TestNetworkingV1EmptyMetadataPlanBoundary(t *testing.T) {
	const address = "kubernetes_ingress_class_v1.test"
	tests := []struct {
		name    string
		address string
		mutate  func(*tfjson.Plan)
		fail    bool
	}{
		{name: "both_maps"},
		{name: "one_map", mutate: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.Before.(map[string]interface{})["metadata"].([]interface{})[0].(map[string]interface{})["annotations"] = map[string]interface{}{}
		}},
		{name: "already_empty", mutate: func(plan *tfjson.Plan) {
			change := plan.ResourceChanges[0].Change
			change.Before = networkingCloneValue(change.After)
			change.Actions = tfjson.Actions{tfjson.ActionNoop}
		}},
		{name: "computed_metadata_unknowns", mutate: func(plan *tfjson.Plan) {
			change := plan.ResourceChanges[0].Change
			metadata := change.After.(map[string]interface{})["metadata"].([]interface{})[0].(map[string]interface{})
			metadata["generation"], metadata["resource_version"] = nil, nil
			change.AfterUnknown = map[string]interface{}{"metadata": []interface{}{map[string]interface{}{"generation": true, "resource_version": true}}}
		}},
		{name: "computed_metadata_unknowns_omitted", mutate: func(plan *tfjson.Plan) {
			change := plan.ResourceChanges[0].Change
			metadata := change.After.(map[string]interface{})["metadata"].([]interface{})[0].(map[string]interface{})
			delete(metadata, "generation")
			delete(metadata, "resource_version")
			change.AfterUnknown = map[string]interface{}{"metadata": []interface{}{map[string]interface{}{"generation": true, "resource_version": true}}}
		}},
		{name: "computed_ingress_class_unknown", address: "kubernetes_ingress_v1.test", mutate: func(plan *tfjson.Plan) {
			change := plan.ResourceChanges[0].Change
			change.Before.(map[string]interface{})["spec"].([]interface{})[0].(map[string]interface{})["ingress_class_name"] = ""
			change.AfterUnknown = map[string]interface{}{"spec": []interface{}{map[string]interface{}{"ingress_class_name": true}}}
		}},
		{name: "computed_ingress_status_unknown", address: "kubernetes_ingress_v1.test", mutate: func(plan *tfjson.Plan) {
			change := plan.ResourceChanges[0].Change
			change.Before.(map[string]interface{})["status"] = []interface{}{map[string]interface{}{
				"load_balancer": []interface{}{map[string]interface{}{"ingress": []interface{}{}}},
			}}
			change.AfterUnknown = map[string]interface{}{"status": true}
		}},
		{name: "computed_scope_unknown", mutate: func(plan *tfjson.Plan) {
			change := plan.ResourceChanges[0].Change
			change.Before.(map[string]interface{})["spec"].([]interface{})[0].(map[string]interface{})["parameters"] = []interface{}{map[string]interface{}{"name": "example", "scope": "Cluster"}}
			change.After.(map[string]interface{})["spec"].([]interface{})[0].(map[string]interface{})["parameters"] = []interface{}{map[string]interface{}{"name": "example"}}
			change.AfterUnknown = map[string]interface{}{"spec": []interface{}{map[string]interface{}{"parameters": []interface{}{map[string]interface{}{"scope": true}}}}}
		}},
		{name: "known_value_marked_unknown", fail: true, mutate: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.AfterUnknown = map[string]interface{}{"metadata": []interface{}{map[string]interface{}{"generation": true}}}
		}},
		{name: "replacement", fail: true, mutate: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.Actions = tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}
		}},
		{name: "missing_update", fail: true, mutate: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.Actions = tfjson.Actions{tfjson.ActionNoop}
		}},
		{name: "already_empty_update", fail: true, mutate: func(plan *tfjson.Plan) {
			change := plan.ResourceChanges[0].Change
			change.Before = networkingCloneValue(change.After)
		}},
		{name: "omitted_map", fail: true, mutate: func(plan *tfjson.Plan) {
			change := plan.ResourceChanges[0].Change
			change.Before, change.After = change.After, change.Before
		}},
		{name: "nonempty_metadata", fail: true, mutate: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.Before.(map[string]interface{})["metadata"].([]interface{})[0].(map[string]interface{})["labels"] = map[string]interface{}{"owner": "other"}
		}},
		{name: "selector_change", fail: true, mutate: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.After.(map[string]interface{})["spec"].([]interface{})[0].(map[string]interface{})["pod_selector"] = []interface{}{map[string]interface{}{"match_labels": map[string]interface{}{}}}
		}},
		{name: "tls_hosts_change", fail: true, mutate: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.After.(map[string]interface{})["spec"].([]interface{})[0].(map[string]interface{})["tls"] = []interface{}{map[string]interface{}{"hosts": []interface{}{}}}
		}},
		{name: "spec_unknown", fail: true, mutate: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.AfterUnknown = map[string]interface{}{"spec": true}
		}},
		{name: "unrelated_resource", fail: true, mutate: func(plan *tfjson.Plan) {
			plan.ResourceChanges = append(plan.ResourceChanges, &tfjson.ResourceChange{
				Address: "kubernetes_namespace_v1.other", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionUpdate}},
			})
		}},
		{name: "refresh_drift", fail: true, mutate: func(plan *tfjson.Plan) {
			plan.ResourceDrift = []*tfjson.ResourceChange{plan.ResourceChanges[0]}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := map[string]interface{}{
				"id": "migration", "spec": []interface{}{map[string]interface{}{"controller": "example.com/controller"}},
				"metadata": []interface{}{map[string]interface{}{
					"name": "migration", "uid": "uid-1", "generation": float64(1), "resource_version": "1",
					"labels": nil, "annotations": nil,
				}},
			}
			after, err := networkingEmptyMetadataValues(before)
			if err != nil {
				t.Fatal(err)
			}
			plan := &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{{
				Address: address, Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionUpdate}, Before: before, After: after},
			}}}
			if tt.mutate != nil {
				tt.mutate(plan)
			}
			testAddress := address
			if tt.address != "" {
				testAddress = tt.address
				plan.ResourceChanges[0].Address = testAddress
			}
			originalBefore := networkingCloneValue(plan.ResourceChanges[0].Change.Before)
			originalAfter := networkingCloneValue(plan.ResourceChanges[0].Change.After)
			var response plancheck.CheckPlanResponse
			networkingEmptyMetadataPlan{address: testAddress}.CheckPlan(context.Background(), plancheck.CheckPlanRequest{Plan: plan}, &response)
			if (response.Error != nil) != tt.fail {
				t.Fatalf("expected failure %t, got %v", tt.fail, response.Error)
			}
			if !reflect.DeepEqual(originalBefore, plan.ResourceChanges[0].Change.Before) {
				t.Fatal("plan check modified the original baseline")
			}
			if !reflect.DeepEqual(originalAfter, plan.ResourceChanges[0].Change.After) {
				t.Fatal("plan check modified the original planned values")
			}
		})
	}
}

func TestNetworkingV1EmptyMetadataStateBoundary(t *testing.T) {
	before := map[string]interface{}{
		"id": "migration", "spec": []interface{}{map[string]interface{}{"controller": "example.com/controller"}},
		"metadata": []interface{}{map[string]interface{}{"labels": nil, "annotations": nil, "uid": "uid-1"}},
	}
	after, err := networkingEmptyMetadataValues(before)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := networkingStateSnapshot{values: before, providerName: "registry.terraform.io/hashicorp/kubernetes"}
	current := &tfjson.StateResource{
		Address: "kubernetes_ingress_class_v1.test", AttributeValues: after, ProviderName: snapshot.providerName,
	}
	request := statecheck.CheckStateRequest{State: &tfjson.State{Values: &tfjson.StateValues{
		RootModule: &tfjson.StateModule{Resources: []*tfjson.StateResource{current}},
	}}}
	check := networkingCaptureState{address: current.Address, snapshot: &snapshot, compare: true, emptyMetadata: true}
	var response statecheck.CheckStateResponse
	check.CheckState(context.Background(), request, &response)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	after["spec"].([]interface{})[0].(map[string]interface{})["controller"] = "example.com/changed"
	check.CheckState(context.Background(), request, &response)
	if response.Error == nil {
		t.Fatal("metadata exception concealed a specification change")
	}
	if before["metadata"].([]interface{})[0].(map[string]interface{})["labels"] != nil {
		t.Fatal("state comparison modified the released baseline")
	}
}

func TestNetworkingV1WriteTracker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	var tracker networkingWriteTracker
	client := &http.Client{Transport: networkingObservedTransport{next: http.DefaultTransport, writes: &tracker}}
	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		req, err := http.NewRequest(method, server.URL+"/apis/networking.k8s.io/v1/ingressclasses/test", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		var check statecheck.CheckStateResponse
		tracker.CheckState(context.Background(), statecheck.CheckStateRequest{}, &check)
		if (check.Error != nil) != (method == http.MethodPatch) {
			t.Fatalf("unexpected write observation for %s: %v", method, check.Error)
		}
	}
}

func (c networkingCorePlanReport) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, _ *plancheck.CheckPlanResponse) {
	for _, drift := range req.Plan.ResourceDrift {
		raw, err := json.Marshal(drift.Change)
		if err != nil {
			c.t.Error(err)
			continue
		}
		c.t.Logf("Core refreshed drift for %s: %s", drift.Address, raw)
	}
	for _, change := range req.Plan.ResourceChanges {
		if change.Change != nil && !change.Change.Actions.NoOp() {
			raw, err := json.Marshal(change.Change)
			if err != nil {
				c.t.Error(err)
				continue
			}
			c.t.Logf("Core planned change for %s: %s", change.Address, raw)
		}
	}
}

func TestNetworkingV1MuxRegistration(t *testing.T) {
	ctx := context.Background()
	sdk := kubernetes.Provider()
	server, err := networkingV1Factories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	networkingNoProtocolErrors(t, schemas.Diagnostics)
	identities, err := server.GetResourceIdentitySchemas(ctx, &tfprotov6.GetResourceIdentitySchemasRequest{})
	if err != nil {
		t.Fatal(err)
	}
	networkingNoProtocolErrors(t, identities.Diagnostics)
	for _, name := range networkingV1Types {
		t.Run(name, func(t *testing.T) {
			if sdk.ResourcesMap[name] != nil {
				t.Fatal("managed v1 type is still registered on SDKv2")
			}
			got := schemas.ResourceSchemas[name]
			if got == nil || got.Version != 0 {
				t.Fatalf("missing resource or changed schema version: %#v", got)
			}
			identity := identities.IdentitySchemas[name]
			if identity == nil || identity.Version != 1 {
				t.Fatalf("missing identity or changed identity version: %#v", identity)
			}
			fields := map[string]tftypes.Type{
				"api_version": tftypes.String, "kind": tftypes.String, "name": tftypes.String,
			}
			if name != "kubernetes_ingress_class_v1" {
				fields["namespace"] = tftypes.String
			}
			if !identity.ValueType().Equal(tftypes.Object{AttributeTypes: fields}) {
				t.Fatalf("unexpected identity shape: %s", identity.ValueType())
			}
		})
	}
	for _, alias := range []string{
		"kubernetes_ingress", "kubernetes_ingress_class", "kubernetes_network_policy",
	} {
		if sdk.ResourcesMap[alias] == nil || schemas.ResourceSchemas[alias] == nil {
			t.Errorf("SDKv2 alias %s was removed", alias)
		}
	}
	if sdk.DataSourcesMap["kubernetes_ingress_v1"] == nil || schemas.DataSourceSchemas["kubernetes_ingress_v1"] == nil {
		t.Fatal("SDKv2 ingress v1 data source was removed")
	}
}

func networkingNoProtocolErrors(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}

func TestNetworkingV1MigrationConfigurations(t *testing.T) {
	ctx := context.Background()
	server, err := networkingV1Factories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	networkingNoProtocolErrors(t, schemas.Diagnostics)
	for _, resourceType := range networkingV1Types {
		variants := []string{"minimal", "full", "empty"}
		if resourceType == "kubernetes_ingress_class_v1" {
			variants = append(variants, "remove_scalars", "remove_parameters")
		} else if resourceType == "kubernetes_network_policy_v1" {
			variants = append(variants, "selector_empty", "selector_maps_empty", "selector_values_empty", "selector_maps_omitted", "selector_values_omitted", "selector_except_omitted")
		}
		for _, variant := range variants {
			t.Run(resourceType+"/"+variant, func(t *testing.T) {
				config := networkingDecodeConfiguration(t, schemas.ResourceSchemas[resourceType],
					networkingV1Config(resourceType, "tf-networking-offline", variant))
				response, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
					TypeName: resourceType, Config: config,
				})
				if err != nil {
					t.Fatal(err)
				}
				networkingNoProtocolErrors(t, response.Diagnostics)
			})
		}
	}
}

func TestNetworkingV1NamespaceIsolation(t *testing.T) {
	for _, namespace := range []string{"", "tf-networking-isolated"} {
		t.Run("namespace="+namespace, func(t *testing.T) {
			t.Setenv("KUBE_NETWORKING_TEST_NAMESPACE", namespace)
			expectedNamespace, prefix := namespace, namespace
			if namespace == "" {
				expectedNamespace, prefix = "default", "tf-networking"
			}
			if got := networkingTestNamespace(); got != expectedNamespace {
				t.Fatalf("namespace = %q, want %q", got, expectedNamespace)
			}
			if name := networkingTestName("isolation"); !strings.HasPrefix(name, prefix+"-isolation-") {
				t.Fatalf("resource name %q is outside the test prefix", name)
			}
			for _, resourceType := range networkingV1Types {
				config := networkingV1Config(resourceType, "test", "minimal")
				hasNamespace := strings.Contains(config, fmt.Sprintf("namespace = %q", namespace))
				if want := namespace != "" && resourceType != "kubernetes_ingress_class_v1"; hasNamespace != want {
					t.Fatalf("%s namespace configuration = %t, want %t", resourceType, hasNamespace, want)
				}
			}
		})
	}
}

func TestNetworkingV1IngressWaitFlagPlan(t *testing.T) {
	ctx := context.Background()
	const resourceType = "kubernetes_ingress_v1"
	server, err := networkingV1Factories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	networkingNoProtocolErrors(t, schemas.Diagnostics)
	resourceSchema := schemas.ResourceSchemas[resourceType]
	var waitAttribute *tfprotov6.SchemaAttribute
	for _, attribute := range resourceSchema.Block.Attributes {
		if attribute.Name == "wait_for_load_balancer" {
			waitAttribute = attribute
		}
	}
	if waitAttribute == nil || !waitAttribute.Optional || waitAttribute.Required || waitAttribute.Computed {
		t.Fatalf("wait_for_load_balancer must remain Optional, not Computed: %#v", waitAttribute)
	}
	prior, err := tfprotov6.NewDynamicValue(resourceSchema.ValueType(), tftypes.NewValue(resourceSchema.ValueType(), nil))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		value interface{}
	}{
		{"omitted", nil},
		{"false", false},
		{"true", true},
		{"unknown", tftypes.UnknownValue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := networkingDecodeConfiguration(t, resourceSchema,
				networkingV1Config(resourceType, "tf-networking-offline", "minimal"))
			value, err := config.Unmarshal(resourceSchema.ValueType())
			if err != nil {
				t.Fatal(err)
			}
			var attributes map[string]tftypes.Value
			if err := value.As(&attributes); err != nil {
				t.Fatal(err)
			}
			want := tftypes.NewValue(tftypes.Bool, tc.value)
			attributes["wait_for_load_balancer"] = want
			dynamic, err := tfprotov6.NewDynamicValue(resourceSchema.ValueType(), tftypes.NewValue(resourceSchema.ValueType(), attributes))
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: resourceType, Config: &dynamic, PriorState: &prior, ProposedNewState: &dynamic,
			})
			if err != nil {
				t.Fatal(err)
			}
			networkingNoProtocolErrors(t, response.Diagnostics)
			planned, err := response.PlannedState.Unmarshal(resourceSchema.ValueType())
			if err != nil {
				t.Fatal(err)
			}
			if err := planned.As(&attributes); err != nil {
				t.Fatal(err)
			}
			if got := attributes["wait_for_load_balancer"]; !got.Equal(want) {
				t.Fatalf("wait_for_load_balancer plan changed: got %s, want %s", got, want)
			}
		})
	}
}

func TestNetworkingV1SDKValidationParity(t *testing.T) {
	ctx := context.Background()
	sdk, err := tf5to6server.UpgradeServer(ctx, kubernetes.Provider().GRPCProvider)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := networkingV1Factories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := actual.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	networkingNoProtocolErrors(t, schemas.Diagnostics)
	for _, tc := range []struct {
		name, resourceType, spec string
		wantError                bool
	}{
		{"controller omitted", "kubernetes_ingress_class_v1", "", false},
		{"controller empty", "kubernetes_ingress_class_v1", `controller = ""`, false},
		{"parameter scalars omitted", "kubernetes_ingress_class_v1", `
parameters {
  kind = "ConfigMap"
  name = "migration"
}`, false},
		{"parameter scalars empty", "kubernetes_ingress_class_v1", `
parameters {
  kind      = "ConfigMap"
  name      = "migration"
  api_group = ""
  namespace = ""
}`, false},
		{"parameter kind missing", "kubernetes_ingress_class_v1", `
parameters {
  name = "migration"
}`, true},
		{"parameter name missing", "kubernetes_ingress_class_v1", `
parameters {
  kind = "ConfigMap"
}`, true},
		{"selector scalars omitted", "kubernetes_network_policy_v1", `
pod_selector {
  match_expressions {}
}
policy_types = ["Ingress"]`, false},
		{"selector scalars empty", "kubernetes_network_policy_v1", `
pod_selector {
  match_expressions {
    key      = ""
    operator = ""
    values   = []
  }
}
policy_types = ["Ingress"]`, false},
		{"policy types missing", "kubernetes_network_policy_v1", `pod_selector {}`, true},
		{"pod selector missing", "kubernetes_network_policy_v1", `policy_types = ["Ingress"]`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := networkingDecodeConfiguration(t, schemas.ResourceSchemas[tc.resourceType], fmt.Sprintf(`
resource %q "test" {
  metadata {
    name = "tf-networking-offline"
  }
  spec {
    %s
  }
}
`, tc.resourceType, tc.spec))
			for _, provider := range []struct {
				name, resourceType string
				server             tfprotov6.ProviderServer
			}{
				{"SDKv2", strings.TrimSuffix(tc.resourceType, "_v1"), sdk},
				{"mux", tc.resourceType, actual},
			} {
				response, err := provider.server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
					TypeName: provider.resourceType, Config: config,
				})
				if err != nil {
					t.Fatal(err)
				}
				hasError := false
				for _, diagnostic := range response.Diagnostics {
					hasError = hasError || diagnostic.Severity == tfprotov6.DiagnosticSeverityError
				}
				if hasError != tc.wantError {
					t.Errorf("%s validation error=%t, want %t", provider.name, hasError, tc.wantError)
					for _, diagnostic := range response.Diagnostics {
						t.Logf("%s: %s", diagnostic.Summary, diagnostic.Detail)
					}
				}
			}
		})
	}
}

func networkingDecodeConfiguration(t *testing.T, resourceSchema *tfprotov6.Schema, config string) *tfprotov6.DynamicValue {
	t.Helper()
	file, diagnostics := hclparse.NewParser().ParseHCL([]byte(config), "main.tf")
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	content, diagnostics := file.Body.Content(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: "resource", LabelNames: []string{"type", "name"}}},
	})
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	if len(content.Blocks) != 1 {
		t.Fatal("expected one managed resource")
	}
	value, diagnostics := hcldec.Decode(content.Blocks[0].Body, networkingDecodeSpec(t, resourceSchema.Block), nil)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	raw, err := ctyjson.Marshal(value, value.Type())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := (&tfprotov6.DynamicValue{JSON: raw}).Unmarshal(resourceSchema.ValueType())
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := tfprotov6.NewDynamicValue(resourceSchema.ValueType(), decoded)
	if err != nil {
		t.Fatal(err)
	}
	return &dynamic
}

func networkingDecodeSpec(t *testing.T, block *tfprotov6.SchemaBlock) hcldec.ObjectSpec {
	t.Helper()
	spec := hcldec.ObjectSpec{}
	for _, attribute := range block.Attributes {
		rawType, err := json.Marshal(attribute.ValueType())
		if err != nil {
			t.Fatal(err)
		}
		valueType, err := ctyjson.UnmarshalType(rawType)
		if err != nil {
			t.Fatal(err)
		}
		// Leave required-value checks to the real provider validation RPC.
		spec[attribute.Name] = &hcldec.AttrSpec{Name: attribute.Name, Type: valueType}
	}
	for _, nested := range block.BlockTypes {
		child := networkingDecodeSpec(t, nested.Block)
		switch nested.Nesting {
		case tfprotov6.SchemaNestedBlockNestingModeList:
			spec[nested.TypeName] = &hcldec.BlockListSpec{TypeName: nested.TypeName, Nested: child}
		case tfprotov6.SchemaNestedBlockNestingModeSet:
			spec[nested.TypeName] = &hcldec.BlockSetSpec{TypeName: nested.TypeName, Nested: child}
		case tfprotov6.SchemaNestedBlockNestingModeSingle:
			spec[nested.TypeName] = &hcldec.BlockSpec{TypeName: nested.TypeName, Nested: child}
		default:
			t.Fatalf("unsupported test schema nesting: %s", nested.TypeName)
		}
	}
	return spec
}

func TestAccNetworkingV1_UpgradeFromSDKv2(t *testing.T) {
	for _, resourceType := range networkingV1Types {
		variants := []string{"minimal", "full", "empty"}
		if resourceType == "kubernetes_network_policy_v1" {
			variants = append(variants, "selector_maps_empty", "selector_values_empty")
		}
		for _, variant := range variants {
			t.Run(resourceType+"/"+variant, func(t *testing.T) {
				testAccNetworkingV1Migration(t, resourceType, variant, false)
			})
		}
	}
}

func TestAccNetworkingV1_MoveFromIngressClassAlias(t *testing.T) {
	for _, variant := range []string{"minimal", "full", "empty"} {
		t.Run(variant, func(t *testing.T) {
			testAccNetworkingV1Migration(t, "kubernetes_ingress_class_v1", variant, true)
		})
	}
}

func TestAccNetworkingV1_MoveFromNetworkPolicyAlias(t *testing.T) {
	for _, variant := range []string{"minimal", "full", "empty", "selector_maps_empty", "selector_values_empty"} {
		t.Run(variant, func(t *testing.T) {
			testAccNetworkingV1Migration(t, "kubernetes_network_policy_v1", variant, true)
		})
	}
}

func TestAccNetworkingV1_IngressClassUpgradeThenRemoveParameters(t *testing.T) {
	const resourceType = "kubernetes_ingress_class_v1"
	name := networkingTestName("remove")
	p := kubernetes.Provider()
	full := networkingV1Config(resourceType, name, "full")
	scalarsRemoved := networkingV1Config(resourceType, name, "remove_scalars")
	parametersRemoved := networkingV1Config(resourceType, name, "remove_parameters")
	var before networkingRemoteSnapshot
	checkRemoval := func(parameters bool) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			expected := before
			spec := networking.IngressClassSpec{Controller: "example.com/networking-migration"}
			if parameters {
				scope := "Cluster"
				spec.Parameters = &networking.IngressClassParametersReference{
					Kind: "ConfigMap", Name: "migration", Scope: &scope,
				}
			}
			expected.Spec = spec
			return networkingCheckRemote(p, resourceType, name, &expected, false)(state)
		}
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { networkingPreCheck(t, p) },
		CheckDestroy: networkingCheckDestroy(p, resourceType, name),
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: networkingV1SDKRelease},
				},
				Config: full,
			},
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: networkingV1SDKRelease},
				},
				Config: full,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: networkingCheckRemote(p, resourceType, name, &before, true),
			},
			{
				ProtoV6ProviderFactories: networkingV1Factories,
				Config:                   full,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: networkingCheckRemote(p, resourceType, name, &before, false),
			},
			{
				ProtoV6ProviderFactories: networkingV1Factories,
				Config:                   scalarsRemoved,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceType+".test", plancheck.ResourceActionUpdate)},
				},
				Check: checkRemoval(true),
			},
			{
				ProtoV6ProviderFactories: networkingV1Factories,
				Config:                   parametersRemoved,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceType+".test", plancheck.ResourceActionUpdate)},
				},
				Check: checkRemoval(false),
			},
			{
				ProtoV6ProviderFactories: networkingV1Factories,
				Config:                   parametersRemoved,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccNetworkingV1_UpdateAndRemoveOptional(t *testing.T) {
	for _, resourceType := range []string{"kubernetes_ingress_v1", "kubernetes_network_policy_v1"} {
		t.Run(resourceType, func(t *testing.T) {
			name := networkingTestName("update")
			p := kubernetes.Provider()
			full := networkingV1Config(resourceType, name, "full")
			minimal := networkingV1Config(resourceType, name, "minimal")
			var before networkingRemoteSnapshot
			checkMinimal := func(state *terraform.State) error {
				expected := before
				expected.Labels, expected.Annotations = maps.Clone(before.Labels), maps.Clone(before.Annotations)
				delete(expected.Labels, "app")
				delete(expected.Annotations, "example.com/owner")
				if len(expected.Labels) == 0 {
					expected.Labels = nil
				}
				if len(expected.Annotations) == 0 {
					expected.Annotations = nil
				}
				if resourceType == "kubernetes_ingress_v1" {
					className := "networking-migration"
					expected.Spec = networking.IngressSpec{
						IngressClassName: &className,
						DefaultBackend: &networking.IngressBackend{
							Service: &networking.IngressServiceBackend{
								Name: "backend", Port: networking.ServiceBackendPort{Number: 80},
							},
						},
					}
				} else {
					expected.Spec = networking.NetworkPolicySpec{PolicyTypes: []networking.PolicyType{networking.PolicyTypeIngress}}
				}
				return networkingCheckRemote(p, resourceType, name, &expected, false)(state)
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { networkingPreCheck(t, p) },
				ProtoV6ProviderFactories: networkingV1Factories,
				CheckDestroy:             networkingCheckDestroy(p, resourceType, name),
				Steps: []resource.TestStep{
					{Config: full, Check: networkingCheckRemote(p, resourceType, name, &before, true)},
					{
						Config: minimal,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceType+".test", plancheck.ResourceActionUpdate)},
						},
						Check: checkMinimal,
					},
					{
						Config: minimal,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: checkMinimal,
					},
					{
						Config: full,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceType+".test", plancheck.ResourceActionUpdate)},
						},
						Check: networkingCheckRemote(p, resourceType, name, &before, false),
					},
					{
						Config: full,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: networkingCheckRemote(p, resourceType, name, &before, false),
					},
				},
			})
		})
	}
}

func TestAccNetworkingV1_IngressDataSource(t *testing.T) {
	const resourceType = "kubernetes_ingress_v1"
	name := networkingTestName("data-source")
	p := kubernetes.Provider()
	config := networkingV1Config(resourceType, name, "minimal") + `
data "kubernetes_ingress_v1" "test" {
  metadata {
    name      = kubernetes_ingress_v1.test.metadata[0].name
    namespace = kubernetes_ingress_v1.test.metadata[0].namespace
  }
}
`
	var before networkingRemoteSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { networkingPreCheck(t, p) },
		ProtoV6ProviderFactories: networkingV1Factories,
		CheckDestroy:             networkingCheckDestroy(p, resourceType, name),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					networkingCheckRemote(p, resourceType, name, &before, true),
					resource.TestCheckResourceAttrPair("data.kubernetes_ingress_v1.test", "metadata.0.uid", resourceType+".test", "metadata.0.uid"),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: networkingCheckRemote(p, resourceType, name, &before, false),
			},
		},
	})
}

func testAccNetworkingV1Migration(t *testing.T, resourceType, variant string, move bool) {
	t.Helper()
	name := networkingTestName("upgrade")
	checkProvider := kubernetes.Provider()
	config := networkingV1Config(resourceType, name, variant)
	sourceType, sourceConfig, targetConfig := resourceType, config, config
	var versionChecks []tfversion.TerraformVersionCheck
	if move {
		sourceType = strings.TrimSuffix(resourceType, "_v1")
		sourceConfig = strings.ReplaceAll(config, resourceType, sourceType)
		targetConfig += fmt.Sprintf(`
moved {
  from = %s.test
  to   = %s.test
}
`, sourceType, resourceType)
		versionChecks = []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)}
	}
	var remote networkingRemoteSnapshot
	var state networkingStateSnapshot
	var writes networkingWriteTracker
	factories := networkingObservedFactories(&writes)
	upgradeCheck := plancheck.ExpectEmptyPlan()
	emptyMetadata := variant == "empty" || variant == "selector_maps_empty" || variant == "selector_values_empty"
	policyVariant := ""
	if resourceType == "kubernetes_network_policy_v1" && emptyMetadata {
		policyVariant = variant
	}
	if emptyMetadata {
		upgradeCheck = networkingEmptyMetadataPlan{address: resourceType + ".test", policyVariant: policyVariant}
	}
	testCase := resource.TestCase{
		PreCheck:               func() { networkingPreCheck(t, checkProvider) },
		TerraformVersionChecks: versionChecks,
		CheckDestroy:           networkingCheckDestroy(checkProvider, resourceType, name),
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: networkingV1SDKRelease},
				},
				Config: sourceConfig,
				Check:  networkingCheckRemote(checkProvider, sourceType, name, &remote, true),
				ConfigStateChecks: []statecheck.StateCheck{
					networkingCaptureState{address: sourceType + ".test", snapshot: &state},
				},
			},
			{
				ProtoV6ProviderFactories: factories,
				Config:                   targetConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{networkingCorePlanReport{t: t}, upgradeCheck},
				},
				Check: networkingCheckRemote(checkProvider, resourceType, name, &remote, false),
				ConfigStateChecks: []statecheck.StateCheck{
					networkingCaptureState{address: resourceType + ".test", snapshot: &state, compare: true, emptyMetadata: emptyMetadata, policyVariant: policyVariant},
					&writes,
				},
			},
			{
				ProtoV6ProviderFactories: factories,
				Config:                   targetConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: networkingCheckRemote(checkProvider, resourceType, name, &remote, false),
				ConfigStateChecks: []statecheck.StateCheck{
					networkingCaptureState{address: resourceType + ".test", snapshot: &state, compare: true, emptyMetadata: emptyMetadata, policyVariant: policyVariant},
					&writes,
				},
			},
		},
	}
	if resourceType == "kubernetes_ingress_class_v1" {
		// Released Create leaves computed metadata unset until Read. Persist a
		// refreshed, unchanged released state before comparing the migration.
		testCase.Steps = append([]resource.TestStep{{
			ExternalProviders: map[string]resource.ExternalProvider{
				"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: networkingV1SDKRelease},
			},
			Config: sourceConfig,
		}}, testCase.Steps...)
		testCase.Steps[1].ConfigPlanChecks = resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		}
	}
	resource.ParallelTest(t, testCase)
}

func TestAccNetworkingV1_ImportAndIdentity(t *testing.T) {
	for _, resourceType := range networkingV1Types {
		t.Run(resourceType, func(t *testing.T) {
			name := networkingTestName("import")
			p := kubernetes.Provider()
			config := networkingV1Config(resourceType, name, "minimal")
			identity := map[string]knownvalue.Check{
				"api_version": knownvalue.StringExact("networking.k8s.io/v1"),
				"kind":        knownvalue.StringExact(networkingKind(resourceType)),
				"name":        knownvalue.StringExact(name),
			}
			if resourceType != "kubernetes_ingress_class_v1" {
				identity["namespace"] = knownvalue.StringExact(networkingTestNamespace())
			}
			var remote networkingRemoteSnapshot
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { networkingPreCheck(t, p) },
				ProtoV6ProviderFactories: networkingV1Factories,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_12_0),
				},
				CheckDestroy: networkingCheckDestroy(p, resourceType, name),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  networkingCheckRemote(p, resourceType, name, &remote, true),
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectIdentity(resourceType+".test", identity),
						},
					},
					{
						ResourceName:      resourceType + ".test",
						ImportState:       true,
						ImportStateVerify: true,
					},
					{
						Config: config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: networkingCheckRemote(p, resourceType, name, &remote, false),
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectIdentity(resourceType+".test", identity),
						},
					},
				},
			})
		})
	}
}

func TestAccNetworkingV1_Disappears(t *testing.T) {
	for _, resourceType := range networkingV1Types {
		t.Run(resourceType, func(t *testing.T) {
			name := networkingTestName("gone")
			p := kubernetes.Provider()
			config := networkingV1Config(resourceType, name, "minimal")
			var before, after networkingRemoteSnapshot
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { networkingPreCheck(t, p) },
				ProtoV6ProviderFactories: networkingV1Factories,
				CheckDestroy:             networkingCheckDestroy(p, resourceType, name),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							networkingCheckRemote(p, resourceType, name, &before, true),
							func(*terraform.State) error { return networkingDelete(p, resourceType, name) },
						),
						ExpectNonEmptyPlan: true,
					},
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							networkingCheckRemote(p, resourceType, name, &after, true),
							func(*terraform.State) error {
								if before.UID == after.UID {
									return fmt.Errorf("externally deleted object was not recreated")
								}
								return nil
							},
						),
					},
				},
			})
		})
	}
}

type networkingStateSnapshot struct {
	values          map[string]interface{}
	identity        map[string]interface{}
	identityVersion *uint64
	providerName    string
}

type networkingCaptureState struct {
	address       string
	snapshot      *networkingStateSnapshot
	compare       bool
	emptyMetadata bool
	policyVariant string
	report        *testing.T
	reportStage   string
}

func (c networkingCaptureState) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	if req.State == nil || req.State.Values == nil || req.State.Values.RootModule == nil {
		resp.Error = fmt.Errorf("state is missing")
		return
	}
	var found *tfjson.StateResource
	for _, r := range req.State.Values.RootModule.Resources {
		if r.Address == c.address {
			found = r
			break
		}
	}
	if found == nil {
		resp.Error = fmt.Errorf("resource %s is missing", c.address)
		return
	}
	if found.SchemaVersion != 0 {
		resp.Error = fmt.Errorf("resource schema version changed to %d", found.SchemaVersion)
		return
	}
	if found.IdentitySchemaVersion != nil && *found.IdentitySchemaVersion != 1 {
		resp.Error = fmt.Errorf("identity schema version changed to %d", *found.IdentitySchemaVersion)
		return
	}
	got := networkingStateSnapshot{
		values: found.AttributeValues, identity: found.IdentityValues,
		identityVersion: found.IdentitySchemaVersion, providerName: found.ProviderName,
	}
	if !c.compare {
		*c.snapshot = got
		if c.report != nil {
			raw, err := json.Marshal(got.values)
			if err != nil {
				resp.Error = err
				return
			}
			c.report.Logf("Released state (%s) for %s: %s", c.reportStage, c.address, raw)
		}
		return
	}
	expected := c.snapshot.values
	if c.emptyMetadata {
		var err error
		expected, err = networkingEmptyMetadataValues(expected)
		if err != nil {
			resp.Error = err
			return
		}
	}
	if c.policyVariant != "" {
		var err error
		expected, err = networkingEmptyPolicySelectorValues(expected, c.policyVariant)
		if err != nil {
			resp.Error = err
			return
		}
	}
	if !reflect.DeepEqual(expected, got.values) ||
		!reflect.DeepEqual(c.snapshot.identity, got.identity) ||
		!reflect.DeepEqual(c.snapshot.identityVersion, got.identityVersion) ||
		c.snapshot.providerName != got.providerName {
		resp.Error = fmt.Errorf("upgrade changed state, identity, or provider for %s:\nvalues: %s\nidentity: %s",
			c.address, cmp.Diff(expected, got.values), cmp.Diff(c.snapshot.identity, got.identity))
	}
}

type networkingWriteTracker struct {
	mu     sync.Mutex
	writes []string
}

func (c *networkingWriteTracker) CheckState(_ context.Context, _ statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.writes) != 0 {
		resp.Error = fmt.Errorf("state-only migration sent Kubernetes writes: %v", c.writes)
	}
}

type networkingObservedTransport struct {
	next   http.RoundTripper
	writes *networkingWriteTracker
}

func (t networkingObservedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodOptions {
		t.writes.mu.Lock()
		t.writes.writes = append(t.writes.writes, req.Method+" "+req.URL.Path)
		t.writes.mu.Unlock()
	}
	return t.next.RoundTrip(req)
}

type networkingObservedMetadata struct {
	kubernetes.KubeClientsets
	kubernetes.MetadataFilters
	client *kubernetesclient.Clientset
}

func (m networkingObservedMetadata) MainClientset() (*kubernetesclient.Clientset, error) {
	return m.client, nil
}

func networkingObservedFactories(writes *networkingWriteTracker) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"kubernetes": func() (tfprotov6.ProviderServer, error) {
			p := kubernetes.Provider()
			configure := p.ConfigureProvider
			p.ConfigureProvider = func(ctx context.Context, req schema.ConfigureProviderRequest, resp *schema.ConfigureProviderResponse) {
				configure(ctx, req, resp)
				if resp.Diagnostics.HasError() || resp.Meta == nil {
					return
				}
				clients := resp.Meta.(kubernetes.KubeClientsets)
				client, err := clients.MainClientset()
				if err != nil {
					resp.Diagnostics = append(resp.Diagnostics, diag.FromErr(err)...)
					return
				}
				restClient := client.NetworkingV1().RESTClient().(*rest.RESTClient)
				httpClient := *restClient.Client
				transport := httpClient.Transport
				if transport == nil {
					transport = http.DefaultTransport
				}
				httpClient.Transport = networkingObservedTransport{next: transport, writes: writes}
				restClient.Client = &httpClient
				resp.Meta = networkingObservedMetadata{
					KubeClientsets: clients, MetadataFilters: resp.Meta.(kubernetes.MetadataFilters), client: client,
				}
			}
			return mux.MuxServerWithProvider(context.Background(), "test", p)
		},
	}
}

type networkingRemoteSnapshot struct {
	UID         types.UID
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
	Spec        interface{}
}

func networkingPreCheck(t *testing.T, p *schema.Provider) {
	t.Helper()
	hasFileCfg := (os.Getenv("KUBE_CTX_AUTH_INFO") != "" && os.Getenv("KUBE_CTX_CLUSTER") != "") ||
		os.Getenv("KUBE_CTX") != "" || os.Getenv("KUBE_CONFIG_PATH") != ""
	hasUserCredentials := os.Getenv("KUBE_USER") != "" && os.Getenv("KUBE_PASSWORD") != ""
	hasClientCert := os.Getenv("KUBE_CLIENT_CERT_DATA") != "" && os.Getenv("KUBE_CLIENT_KEY_DATA") != ""
	hasStaticCfg := os.Getenv("KUBE_HOST") != "" && os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") != "" &&
		(hasUserCredentials || hasClientCert || os.Getenv("KUBE_TOKEN") != "")
	if !hasFileCfg && !hasStaticCfg && !hasUserCredentials {
		t.Fatal("explicit Kubernetes acceptance-test configuration is required")
	}
	if diagnostics := p.Configure(context.Background(), sdkterraform.NewResourceConfigRaw(nil)); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
}

func networkingTestNamespace() string {
	if namespace := os.Getenv("KUBE_NETWORKING_TEST_NAMESPACE"); namespace != "" {
		return namespace
	}
	return "default"
}

func networkingTestName(purpose string) string {
	prefix := "tf-networking"
	if namespace := os.Getenv("KUBE_NETWORKING_TEST_NAMESPACE"); namespace != "" {
		prefix = namespace
	}
	return prefix + "-" + purpose + "-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
}

func networkingKind(resourceType string) string {
	switch resourceType {
	case "kubernetes_ingress_v1":
		return "Ingress"
	case "kubernetes_ingress_class", "kubernetes_ingress_class_v1":
		return "IngressClass"
	case "kubernetes_network_policy", "kubernetes_network_policy_v1":
		return "NetworkPolicy"
	default:
		panic("unsupported networking migration type: " + resourceType)
	}
}

func networkingRead(p *schema.Provider, resourceType, name string) (networkingRemoteSnapshot, error) {
	var result networkingRemoteSnapshot
	client, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		return result, err
	}
	ctx := context.Background()
	var metadata metav1.ObjectMeta
	switch networkingKind(resourceType) {
	case "Ingress":
		object, err := client.NetworkingV1().Ingresses(networkingTestNamespace()).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return result, err
		}
		metadata, result.Spec = object.ObjectMeta, object.Spec
	case "IngressClass":
		object, err := client.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return result, err
		}
		metadata, result.Spec = object.ObjectMeta, object.Spec
	case "NetworkPolicy":
		object, err := client.NetworkingV1().NetworkPolicies(networkingTestNamespace()).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return result, err
		}
		metadata, result.Spec = object.ObjectMeta, object.Spec
	}
	result.UID, result.Name, result.Namespace = metadata.UID, metadata.Name, metadata.Namespace
	result.Labels, result.Annotations = maps.Clone(metadata.Labels), maps.Clone(metadata.Annotations)
	return result, nil
}

func networkingCheckRemote(p *schema.Provider, resourceType, name string, snapshot *networkingRemoteSnapshot, capture bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		r, ok := state.RootModule().Resources[resourceType+".test"]
		if !ok || r.Primary == nil || r.Primary.ID == "" {
			return fmt.Errorf("%s.test is missing from state", resourceType)
		}
		got, err := networkingRead(p, resourceType, name)
		if err != nil {
			return err
		}
		if got.UID == "" || r.Primary.Attributes["metadata.0.uid"] != string(got.UID) {
			return fmt.Errorf("remote UID %q does not match state UID %q for %s", got.UID, r.Primary.Attributes["metadata.0.uid"], resourceType)
		}
		if capture {
			*snapshot = got
		} else if diff := cmp.Diff(*snapshot, got); diff != "" {
			return fmt.Errorf("upgrade changed remote identity, metadata, or desired spec (-before +after):\n%s", diff)
		}
		return nil
	}
}

func networkingCheckDestroy(p *schema.Provider, resourceType, name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		_, err := networkingRead(p, resourceType, name)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("%s %s still exists", resourceType, name)
	}
}

func networkingDelete(p *schema.Provider, resourceType, name string) error {
	client, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	switch networkingKind(resourceType) {
	case "Ingress":
		return client.NetworkingV1().Ingresses(networkingTestNamespace()).Delete(ctx, name, metav1.DeleteOptions{})
	case "IngressClass":
		return client.NetworkingV1().IngressClasses().Delete(ctx, name, metav1.DeleteOptions{})
	default:
		return client.NetworkingV1().NetworkPolicies(networkingTestNamespace()).Delete(ctx, name, metav1.DeleteOptions{})
	}
}

func networkingV1Config(resourceType, name, variant string) string {
	metadata := ""
	if variant == "full" || variant == "remove_scalars" || variant == "remove_parameters" {
		metadata = `
    annotations = { "example.com/owner" = "networking-migration" }
    labels      = { "app" = "networking-migration" }
`
	} else if variant == "empty" || strings.HasPrefix(variant, "selector_") {
		metadata = `
    annotations = {}
    labels      = {}
`
	}
	if namespace := os.Getenv("KUBE_NETWORKING_TEST_NAMESPACE"); namespace != "" && resourceType != "kubernetes_ingress_class_v1" {
		metadata += fmt.Sprintf("    namespace = %q\n", namespace)
	}
	spec, extra := "", ""
	switch resourceType {
	case "kubernetes_ingress_v1":
		spec = `
    default_backend {
      service {
        name = "backend"
        port {
          number = 80
        }
      }
    }
`
		if variant == "full" {
			spec += `
    ingress_class_name = "networking-migration"
    rule {
      host = "migration.example.com"
      http {
        path {
          path      = "/api"
          path_type = "Prefix"
          backend {
            service {
              name = "api"
              port {
                name = "http"
              }
            }
          }
        }
        path {
          path      = "/assets"
          path_type = "Exact"
          backend {
            resource {
              api_group = "example.com"
              kind      = "StorageBucket"
              name      = "assets"
            }
          }
        }
      }
    }
    tls {
      hosts       = ["migration.example.com"]
      secret_name = "migration-tls"
    }
`
			extra = `
  wait_for_load_balancer = false
  timeouts {
    create = "5m"
    delete = "5m"
  }
`
		} else if variant == "empty" {
			spec += `
    tls {
      hosts       = []
      secret_name = ""
    }
`
			extra = "\n  wait_for_load_balancer = false\n"
		}
	case "kubernetes_ingress_class_v1":
		spec = "\n    controller = \"example.com/networking-migration\"\n"
		if variant == "full" {
			spec += fmt.Sprintf(`
    parameters {
      api_group = "example.com"
      kind      = "IngressConfiguration"
      name      = "migration"
      scope     = "Namespace"
      namespace = %q
    }
`, networkingTestNamespace())
		} else if variant == "empty" {
			spec += `
    parameters {
      api_group = ""
      kind      = "ConfigMap"
      name      = "migration"
      namespace = ""
    }
`
		} else if variant == "remove_scalars" {
			spec += `
    parameters {
      kind  = "ConfigMap"
      name  = "migration"
      scope = "Cluster"
    }
`
		}
	case "kubernetes_network_policy_v1":
		spec = `
    pod_selector {}
    policy_types = ["Ingress"]
`
		if variant == "full" {
			spec = `
    pod_selector {
      match_labels = { app = "networking-migration" }
      match_expressions {
        key      = "tier"
        operator = "In"
        values   = ["api", "web"]
      }
    }
    policy_types = ["Ingress", "Egress"]
    ingress {
      ports {
        port     = "8080"
        end_port = 8090
        protocol = "TCP"
      }
      ports {
        port     = "dns"
        protocol = "UDP"
      }
      from {
        namespace_selector {
          match_labels = { team = "platform" }
        }
        pod_selector {
          match_expressions {
            key      = "enabled"
            operator = "Exists"
          }
        }
      }
      from {
        ip_block {
          cidr   = "10.0.0.0/8"
          except = ["10.1.0.0/16"]
        }
      }
    }
    egress {
      ports {
        port     = "53"
        protocol = "UDP"
      }
      to {
        ip_block {
          cidr   = "192.168.0.0/16"
          except = ["192.168.1.0/24"]
        }
      }
      to {
        namespace_selector {}
        pod_selector {
          match_labels = { app = "dns" }
        }
      }
    }
`
		} else if variant == "empty" {
			spec = `
    pod_selector {
      match_labels = {}
      match_expressions {
        key      = "blocked"
        operator = "DoesNotExist"
        values   = []
      }
    }
    policy_types = ["Ingress", "Egress"]
    ingress {
      ports {
        port     = ""
        end_port = 0
      }
      from {
        namespace_selector {}
        pod_selector {}
      }
    }
    egress {
      to {
        ip_block {
          cidr   = "10.0.0.0/8"
          except = []
        }
      }
    }
`
		} else if variant == "selector_maps_empty" || variant == "selector_values_empty" {
			selector := "match_labels = {}"
			if variant == "selector_values_empty" {
				selector = `
        match_labels = { app = "migration" }
        match_expressions {
          key      = "present"
          operator = "Exists"
          values   = []
        }
        match_expressions {
          key      = "absent"
          operator = "DoesNotExist"
          values   = []
        }
`
			}
			spec = fmt.Sprintf(`
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
`, selector, selector, selector, selector, selector)
		} else if strings.HasPrefix(variant, "selector_") {
			matchLabels, values, except := "match_labels = {}", "values = []", "except = []"
			peerMatchLabels := matchLabels
			switch variant {
			case "selector_maps_omitted":
				matchLabels = ""
				peerMatchLabels = ""
			case "selector_values_omitted":
				values = ""
				peerMatchLabels = `match_labels = { team = "migration" }`
			case "selector_except_omitted":
				except = ""
				peerMatchLabels = `match_labels = { team = "migration" }`
			}
			spec = fmt.Sprintf(`
    pod_selector {
      %s
      match_expressions {
        key      = "blocked"
        operator = "DoesNotExist"
        %s
      }
    }
    policy_types = ["Ingress", "Egress"]
    ingress {
      from {
        namespace_selector {
          %s
        }
        pod_selector {
          %s
        }
      }
    }
    egress {
      to {
        ip_block {
          cidr = "10.0.0.0/8"
          %s
        }
      }
    }
`, matchLabels, values, peerMatchLabels, peerMatchLabels, except)
		}
	default:
		panic("unsupported networking migration type: " + resourceType)
	}
	return fmt.Sprintf(`
resource %q "test" {
  metadata {
    name = %q
%s
  }
  spec {
%s
  }
%s
}
`, resourceType, name, metadata, spec, extra)
}
