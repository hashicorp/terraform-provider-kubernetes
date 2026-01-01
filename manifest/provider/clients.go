// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/openapi"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	// this is how client-go expects auth plugins to be loaded
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// keys into the global state storage
const (
	OAPIFoundry string = "OPENAPIFOUNDRY"
)

// getDynamicClient returns a configured unstructured (dynamic) client instance
func (ps *RawProviderServer) getDynamicClient() (dynamic.Interface, error) {
	if ps.clientConfig == nil {
		return nil, fmt.Errorf("cannot create dynamic client: no client config")
	}

	return ps.dynamicClient.Get(func() (dynamic.Interface, error) {
		return dynamic.NewForConfig(ps.clientConfig)
	})
}

// getDiscoveryClient returns a configured discovery client instance.
func (ps *RawProviderServer) getDiscoveryClient() (discovery.DiscoveryInterface, error) {
	if ps.clientConfig == nil {
		return nil, fmt.Errorf("cannot create discovery client: no client config")
	}

	return ps.discoveryClient.Get(func() (discovery.DiscoveryInterface, error) {
		return discovery.NewDiscoveryClientForConfig(ps.clientConfig)
	})
}

// getRestMapper returns a RESTMapper client instance
func (ps *RawProviderServer) getRestMapper() (meta.RESTMapper, error) {
	return ps.restMapper.Get(func() (meta.RESTMapper, error) {
		dc, err := ps.getDiscoveryClient()
		if err != nil {
			return nil, err
		}

		cacheClient := memory.NewMemCacheClient(dc)
		return restmapper.NewDeferredDiscoveryRESTMapper(cacheClient), nil
	})
}

// getRestClient returns a raw REST client instance
func (ps *RawProviderServer) getRestClient() (rest.Interface, error) {
	if ps.clientConfig == nil {
		return nil, fmt.Errorf("cannot create REST client: no client config")
	}

	return ps.restClient.Get(func() (rest.Interface, error) {
		return rest.UnversionedRESTClientFor(ps.clientConfig)
	})
}

// getOAPIv2Foundry returns an interface to request tftype types from an OpenAPIv2 spec
func (ps *RawProviderServer) getOAPIv2Foundry() (openapi.Foundry, error) {
	return ps.OAPIFoundry.Get(func() (openapi.Foundry, error) {
		rc, err := ps.getRestClient()
		if err != nil {
			return nil, fmt.Errorf("failed get OpenAPI spec: %s", err)
		}

		rq := rc.Verb("GET").Timeout(30*time.Second).AbsPath("openapi", "v2")
		rs, err := rq.DoRaw(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed get OpenAPI spec: %s", err)
		}

		oapif, err := openapi.NewFoundryFromSpecV2(rs)
		if err != nil {
			return nil, fmt.Errorf("failed construct OpenAPI foundry: %s", err)
		}

		return oapif, nil
	})
}

func loggingTransport(rt http.RoundTripper) http.RoundTripper {
	return &loggingRountTripper{
		ot: rt,
		lt: logging.NewSubsystemLoggingHTTPTransport("Kubernetes API", rt),
	}
}

type loggingRountTripper struct {
	ot http.RoundTripper
	lt http.RoundTripper
}

func (t *loggingRountTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Path == "/openapi/v2" {
		// don't trace-log the OpenAPI spec document, it's really big
		return t.ot.RoundTrip(req)
	}
	return t.lt.RoundTrip(req)
}

func (ps *RawProviderServer) checkValidCredentials(ctx context.Context) []*tfprotov5.Diagnostic {
	diagnostics, _ := ps.checkValidCredentialsResult.Get(func() (diags []*tfprotov5.Diagnostic, err error) {
		rc, err := ps.getRestClient()
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to construct REST client",
				Detail:   err.Error(),
			})
			return
		}
		vpath := []string{"/apis"}
		rs := rc.Get().AbsPath(vpath...).Do(ctx)
		if rs.Error() != nil {
			switch {
			case apierrors.IsUnauthorized(rs.Error()):
				diags = append(diags, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Invalid credentials",
					Detail:   fmt.Sprintf("The credentials configured in the provider block are not accepted by the API server. Error: %s\n\nSet TF_LOG=debug and look for '[InvalidClientConfiguration]' in the log to see actual configuration.", rs.Error().Error()),
				})
			default:
				diags = append(diags, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Invalid configuration for API client",
					Detail:   rs.Error().Error(),
				})
			}
			ps.logger.Debug("[InvalidClientConfiguration]", "Config", dump(ps.clientConfig))
		}
		return
	})

	return diagnostics
}
