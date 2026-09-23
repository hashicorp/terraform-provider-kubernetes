// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/useragent"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type recordedRequest struct {
	path      string
	userAgent string
}

// TestConfigureProviderUserAgent asserts that every client the manifest provider
// builds sends the provider's User-Agent. The clients default differently when
// rest.Config.UserAgent is left empty -- the raw REST client sends Go's
// "Go-http-client/1.1", while the dynamic and discovery clients substitute the
// client-go default -- so each flavour is checked against the wire rather than
// trusting that one assignment covers them all.
func TestConfigureProviderUserAgent(t *testing.T) {
	cases := map[string]string{
		"without TF_APPEND_USER_AGENT": "",
		"with TF_APPEND_USER_AGENT":    "AcmeCorp/9.9.9",
	}

	for name, appendUA := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("KUBE_CONFIG_PATH", "")
			t.Setenv("KUBE_CONFIG_PATHS", "")
			t.Setenv("TF_APPEND_USER_AGENT", appendUA)

			srv, recorded := fakeClusterServer(t)
			s := configureProviderAgainst(t, srv.URL)

			// Raw REST client: used for the OpenAPI spec fetch and the
			// credential check.
			restClient, err := s.getRestClient()
			if err != nil {
				t.Fatalf("rest client: %v", err)
			}
			if res := restClient.Get().AbsPath("/apis").Do(context.Background()); res.Error() != nil {
				t.Fatalf("rest client request: %v", res.Error())
			}

			// Discovery client: used to build the RESTMapper.
			discoveryClient, err := s.getDiscoveryClient()
			if err != nil {
				t.Fatalf("discovery client: %v", err)
			}
			if _, err := discoveryClient.ServerVersion(); err != nil {
				t.Fatalf("discovery client request: %v", err)
			}

			// Dynamic client: what kubernetes_manifest does CRUD with.
			dynamicClient, err := s.getDynamicClient()
			if err != nil {
				t.Fatalf("dynamic client: %v", err)
			}
			gvr := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
			if _, err := dynamicClient.Resource(gvr).List(context.Background(), metav1.ListOptions{}); err != nil {
				t.Fatalf("dynamic client request: %v", err)
			}

			reqs := recorded()
			if len(reqs) != 3 {
				t.Fatalf("expected one request per client, recorded %d: %+v", len(reqs), reqs)
			}

			wantProvider := "terraform-provider-kubernetes/" + useragent.ProviderVersion
			for _, r := range reqs {
				if !strings.HasPrefix(r.userAgent, "Terraform/1.9.5 ") {
					t.Errorf("request to %s sent unidentified User-Agent %q", r.path, r.userAgent)
				}
				if !strings.Contains(r.userAgent, wantProvider) {
					t.Errorf("request to %s omitted %q: %q", r.path, wantProvider, r.userAgent)
				}
				if appendUA != "" && !strings.HasSuffix(r.userAgent, appendUA) {
					t.Errorf("request to %s did not append TF_APPEND_USER_AGENT: %q", r.path, r.userAgent)
				}
			}
		})
	}
}

// configureProviderAgainst runs the real ConfigureProvider RPC with every
// attribute null except host, as Terraform would for
// `provider "kubernetes" { host = ... }`.
func configureProviderAgainst(t *testing.T, host string) *RawProviderServer {
	t.Helper()

	cfgType := GetObjectTypeFromSchema(GetProviderConfigSchema())
	attrTypes := cfgType.(tftypes.Object).AttributeTypes

	vals := make(map[string]tftypes.Value, len(attrTypes))
	for name, ty := range attrTypes {
		vals[name] = tftypes.NewValue(ty, nil)
	}
	vals["host"] = tftypes.NewValue(tftypes.String, host)

	cfg, err := tfprotov5.NewDynamicValue(cfgType, tftypes.NewValue(cfgType, vals))
	if err != nil {
		t.Fatalf("encoding provider config: %v", err)
	}

	s := &RawProviderServer{logger: hclog.NewNullLogger()}
	resp, err := s.ConfigureProvider(context.Background(), &tfprotov5.ConfigureProviderRequest{
		TerraformVersion: "1.9.5",
		Config:           &cfg,
	})
	if err != nil {
		t.Fatalf("ConfigureProvider: %v", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov5.DiagnosticSeverityError {
			t.Fatalf("ConfigureProvider diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}
	if s.clientConfig == nil {
		t.Fatal("ConfigureProvider produced no client config")
	}
	return s
}

// fakeClusterServer serves just enough of the Kubernetes API for client-go to
// build clients and issue one request each, recording the User-Agent of every
// request it receives.
func fakeClusterServer(t *testing.T) (*httptest.Server, func() []recordedRequest) {
	t.Helper()

	responses := map[string]string{
		"/version":           `{"major":"1","minor":"33","gitVersion":"v1.33.0"}`,
		"/api":               `{"kind":"APIVersions","versions":["v1"]}`,
		"/apis":              `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`,
		"/api/v1/namespaces": `{"kind":"NamespaceList","apiVersion":"v1","metadata":{"resourceVersion":"1"},"items":[]}`,
	}

	var mu sync.Mutex
	var recorded []recordedRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		recorded = append(recorded, recordedRequest{path: r.URL.Path, userAgent: r.Header.Get("User-Agent")})
		mu.Unlock()

		body, ok := responses[r.URL.Path]
		if !ok {
			body = "{}"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv, func() []recordedRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]recordedRequest(nil), recorded...)
	}
}
