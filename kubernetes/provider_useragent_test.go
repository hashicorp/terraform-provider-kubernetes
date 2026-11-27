// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/useragent"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestProviderConfigureUserAgent asserts that the client the SDKv2 provider
// configures identifies itself to the API server, and that TF_APPEND_USER_AGENT
// reaches the wire.
func TestProviderConfigureUserAgent(t *testing.T) {
	cases := map[string]string{
		"without TF_APPEND_USER_AGENT": "",
		"with TF_APPEND_USER_AGENT":    "AcmeCorp/9.9.9",
	}

	for name, appendUA := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("KUBE_CONFIG_PATH", "")
			t.Setenv("KUBE_CONFIG_PATHS", "")
			t.Setenv("TF_APPEND_USER_AGENT", appendUA)

			var mu sync.Mutex
			var got string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				got = r.Header.Get("User-Agent")
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"kind":"Namespace","apiVersion":"v1","metadata":{"name":"default"}}`))
			}))
			t.Cleanup(srv.Close)

			d := schema.TestResourceDataRaw(t, Provider().Schema, map[string]interface{}{
				"host": srv.URL,
			})
			meta, diags := providerConfigure(context.Background(), d, "1.9.5")
			if diags.HasError() {
				t.Fatalf("providerConfigure: %+v", diags)
			}

			clientset, err := meta.(providerMetadata).MainClientset()
			if err != nil {
				t.Fatalf("clientset: %v", err)
			}
			if _, err := clientset.CoreV1().Namespaces().Get(context.Background(), "default", metav1.GetOptions{}); err != nil {
				t.Fatalf("namespace get: %v", err)
			}

			mu.Lock()
			defer mu.Unlock()
			if !strings.HasPrefix(got, "Terraform/1.9.5 ") {
				t.Errorf("unidentified User-Agent %q", got)
			}
			if want := "terraform-provider-kubernetes/" + useragent.ProviderVersion; !strings.Contains(got, want) {
				t.Errorf("User-Agent %q does not contain %q", got, want)
			}
			if appendUA != "" && !strings.HasSuffix(got, appendUA) {
				t.Errorf("User-Agent %q did not append TF_APPEND_USER_AGENT", got)
			}
		})
	}
}
