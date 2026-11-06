// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	corev1 "k8s.io/api/core/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/util"
	restclient "k8s.io/client-go/rest"
)

func TestBootstrapConnectionConfig(t *testing.T) {
	block := cty.ObjectVal(map[string]cty.Value{
		"host":                   cty.StringVal("https://cluster.example.com"),
		"token":                  cty.StringVal("a-service-account-token"),
		"cluster_ca_certificate": cty.NullVal(cty.String),
		"insecure":               cty.True,
	})

	cfg, diags := initializeConfiguration(&ctyConfigGetter{v: block}, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if cfg == nil {
		t.Fatal("expected a client config")
	}
	if cfg.Host != "https://cluster.example.com" {
		t.Errorf("host: got %q", cfg.Host)
	}
	if cfg.BearerToken != "a-service-account-token" {
		t.Errorf("token: got %q", cfg.BearerToken)
	}
	if !cfg.TLSClientConfig.Insecure {
		t.Error("insecure: expected true")
	}
}

func TestBootstrapConnectionIgnoresEnvironment(t *testing.T) {
	resetEnv := unsetEnv(t)
	defer resetEnv()

	os.Setenv("KUBE_CONFIG_PATHS", strings.Join([]string{
		"test-fixtures/kube-config.yaml",
		"test-fixtures/kube-config-secondary.yaml",
	}, string(os.PathListSeparator)))

	// The fixtures have no current-context, so both halves name one. The only
	// difference between them is whether the environment may be consulted.
	settings := map[string]interface{}{"config_context": "oidc"}

	cfg, diags := initializeConfiguration(mapConfigGetter(settings), true)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if cfg == nil {
		t.Fatal("expected the provider path to load the kubeconfig named by KUBE_CONFIG_PATHS")
	}

	cfg, _ = initializeConfiguration(mapConfigGetter(settings), false)
	if cfg != nil {
		t.Fatalf("expected a bootstrap_connection block to ignore KUBE_CONFIG_PATHS, got host %q", cfg.Host)
	}
}

func TestCtyToNativeMaterialisesUnsetCollections(t *testing.T) {
	exec := cty.ObjectVal(map[string]cty.Value{
		"api_version": cty.StringVal("client.authentication.k8s.io/v1"),
		"command":     cty.StringVal("aws"),
		"args":        cty.NullVal(cty.List(cty.String)),
		"env":         cty.NullVal(cty.Map(cty.String)),
	})

	spec, ok := ctyToNative(exec).(map[string]interface{})
	if !ok {
		t.Fatalf("expected a map, got %T", ctyToNative(exec))
	}

	args, ok := spec["args"].([]interface{})
	if !ok {
		t.Fatalf("args: expected []interface{}, got %T", spec["args"])
	}
	if len(args) != 0 {
		t.Errorf("args: expected empty, got %v", args)
	}

	env, ok := spec["env"].(map[string]interface{})
	if !ok {
		t.Fatalf("env: expected map[string]interface{}, got %T", spec["env"])
	}
	if len(env) != 0 {
		t.Errorf("env: expected empty, got %v", env)
	}
}

func TestCtyConfigGetter(t *testing.T) {
	g := &ctyConfigGetter{v: cty.ObjectVal(map[string]cty.Value{
		"host":         cty.StringVal("https://cluster.example.com"),
		"insecure":     cty.False,
		"config_path":  cty.NullVal(cty.String),
		"config_paths": cty.ListVal([]cty.Value{cty.StringVal("/a"), cty.StringVal("/b")}),
	})}

	if v, ok := g.GetOk("host"); !ok || v.(string) != "https://cluster.example.com" {
		t.Errorf("host: got %v, %v", v, ok)
	}
	// False is the zero value for a bool, so GetOk reports it unset - matching
	// (*schema.ResourceData).GetOk.
	if _, ok := g.GetOk("insecure"); ok {
		t.Error("insecure: expected false for a zero value")
	}
	if _, ok := g.GetOk("config_path"); ok {
		t.Error("config_path: expected false for a null value")
	}
	if _, ok := g.GetOk("absent"); ok {
		t.Error("absent: expected false for a missing attribute")
	}
	paths, ok := g.Get("config_paths").([]interface{})
	if !ok || len(paths) != 2 {
		t.Errorf("config_paths: got %v", g.Get("config_paths"))
	}
}

func injectedTestResource(withUpdate bool) *schema.Resource {
	noop := func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		return nil
	}
	r := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {Type: schema.TypeString, Required: true, ForceNew: true},
		},
		CreateContext: noop,
		ReadContext:   noop,
		DeleteContext: noop,
	}
	if withUpdate {
		r.UpdateContext = noop
	}
	injectBootstrapConnection(r)
	return r
}

func bootstrapConnectionState() *terraform.InstanceState {
	return &terraform.InstanceState{
		ID: "default/example",
		Attributes: map[string]string{
			"id":                     "default/example",
			"name":                   "example",
			"bootstrap_connection.#": "1",
		},
	}
}

func TestConnectionBlockIsWriteOnlyThroughout(t *testing.T) {
	collections := map[string]bool{"config_paths": true, "exec.args": true}

	var check func(prefix string, attrs map[string]*schema.Schema)
	check = func(prefix string, attrs map[string]*schema.Schema) {
		for name, s := range attrs {
			path := prefix + name

			if nested, ok := s.Elem.(*schema.Resource); ok {
				check(path+".", nested.Schema)
				continue
			}

			switch {
			case collections[path]:
				if s.Type != schema.TypeList && s.Type != schema.TypeSet {
					t.Errorf("%s is exempt from write-only but is not a list or set, so it need not be", path)
				}
				if s.WriteOnly {
					t.Errorf("%s is write-only, which the SDK rejects for lists and sets", path)
				}
			case !s.WriteOnly:
				t.Errorf("%s is not write-only", path)
			}
		}
	}

	check("", bootstrapConnectionSchema().Elem.(*schema.Resource).Schema)
}

func TestBootstrapConnectionUpdateRequiresTheProvider(t *testing.T) {
	r := injectedTestResource(true)

	d := r.Data(bootstrapConnectionState())
	diags := r.UpdateContext(context.Background(), d, providerMetadata{})
	if !diags.HasError() {
		t.Fatal("expected an error when the provider has no configuration")
	}
	if diags[0].Summary != util.ProviderRequiredSummary {
		t.Errorf("got summary %q", diags[0].Summary)
	}

	configured := providerMetadata{config: &restclient.Config{Host: "https://cluster.example.com"}}
	if diags := r.UpdateContext(context.Background(), d, configured); diags.HasError() {
		t.Errorf("unexpected diagnostics: %v", diags)
	}
}

func TestProviderResourcesAcceptBootstrapConnection(t *testing.T) {
	for name, r := range Provider().ResourcesMap {
		if _, ok := r.Schema[bootstrapConnectionKey]; !ok {
			t.Errorf("%s: missing %q block", name, bootstrapConnectionKey)
		}
		if !hasUpdate(r) {
			t.Errorf("%s: no update defined after injection", name)
		}
	}
}

func TestAccKubernetesBootstrapConnection_basic(t *testing.T) {
	var conf corev1.Namespace
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_namespace_v1.test"
	kubeconfig := os.Getenv("KUBE_CONFIG_PATH")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if kubeconfig == "" {
				t.Skip("KUBE_CONFIG_PATH must be set for the bootstrap_connection acceptance test")
			}
		},
		ProviderFactories: testAccProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			// The block is write-only throughout.
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesBootstrapConnectionConfig_basic(name, kubeconfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "bootstrap_connection.#", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "bootstrap_connection.0.config_path"),
				),
			},
			{
				Config:   testAccKubernetesBootstrapConnectionConfig_edited(name, kubeconfig),
				PlanOnly: true,
			},
		},
	})
}

func testAccKubernetesBootstrapConnectionConfig_basic(name, kubeconfig string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  bootstrap_connection {
    config_path = %q
  }

  metadata {
    name = %q
  }
}
`, kubeconfig, name)
}

func testAccKubernetesBootstrapConnectionConfig_edited(name, kubeconfig string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  bootstrap_connection {
    config_path = %q
    insecure    = false
  }

  metadata {
    name = %q
  }
}
`, kubeconfig, name)
}
