// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestBatchCLI_AmbientConfigurationIsolation(t *testing.T) {
	// Bogus inherited paths and credentials must not be loaded, validated, or
	// passed to client-go when the fixture supplies its explicit loopback host.
	inherited := map[string]string{
		"KUBECONFIG":                "does-not-exist-qa-kubeconfig",
		"KUBE_CONFIG_PATH":          "does-not-exist-qa-kubeconfig",
		"KUBE_CONFIG_PATHS":         "does-not-exist-qa-kubeconfig",
		"KUBE_HOST":                 "https://invalid.invalid",
		"KUBE_TOKEN":                "not-a-real-token",
		"KUBE_USER":                 "not-a-real-user",
		"KUBE_PASSWORD":             "not-a-real-password",
		"KUBE_CLIENT_CERT_DATA":     "not-a-real-certificate",
		"KUBE_CLIENT_KEY_DATA":      "not-a-real-key",
		"KUBE_CLUSTER_CA_CERT_DATA": "not-a-real-ca",
		"KUBE_CTX":                  "does-not-exist",
		"KUBE_CTX_AUTH_INFO":        "does-not-exist",
		"KUBE_CTX_CLUSTER":          "does-not-exist",
		"KUBE_PROXY_URL":            "http://invalid.invalid",
		"KUBERNETES_SERVICE_HOST":   "invalid.invalid",
		"KUBERNETES_SERVICE_PORT":   "443",
	}
	for name, value := range inherited {
		t.Setenv(name, value)
	}
	batchCLIEnvironment(t)
	for name := range inherited {
		if value := os.Getenv(name); value != "" {
			t.Fatalf("ambient provider configuration %s was not isolated", name)
		}
	}
	api := newBatchCLIAPI(t)
	var uid string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             api.checkDestroy,
		Steps: []resource.TestStep{{
			Config: batchCLIConfig(api, "jobs", batchCLIOptions{}),
			Check:  api.check("jobs", &uid, 1, 0, nil),
		}},
	})
}

func TestBatchCLI_InvalidConfiguration(t *testing.T) {
	batchCLIEnvironment(t)
	for _, test := range []struct {
		kind    string
		options batchCLIOptions
		error   string
	}{
		{kind: "jobs", options: batchCLIOptions{ttl: "-1"}, error: "Invalid job TTL"},
		{kind: "cronjobs", options: batchCLIOptions{schedule: "not a schedule"}, error: "Invalid Cron expression"},
	} {
		t.Run(test.kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{{
					Config:      batchCLIConfig(api, test.kind, test.options),
					ExpectError: regexp.MustCompile(test.error),
				}},
			})
			api.mu.Lock()
			defer api.mu.Unlock()
			if len(api.calls) != 0 {
				t.Errorf("invalid configuration reached the API: %v", api.calls)
			}
		})
	}
}

func TestBatchCLI_HTTPConnectionFailure(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			// A closed loopback listener is a negative control: the same real
			// mux/client-go path must fail, not silently skip or use kubeconfig.
			api.server.Close()
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{{
					Config:      batchCLIConfig(api, kind, batchCLIOptions{}),
					ExpectError: regexp.MustCompile(`(?i)connection refused|actively refused`),
				}},
			})
		})
	}
}
