// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/openapi"
	"github.com/mitchellh/hashstructure"

	"github.com/hashicorp/terraform-provider-kubernetes/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type resourceClients struct {
	ps           *RawProviderServer
	clientConfig *rest.Config
	key          uint64
}

func connectionFromConfig(cfg *tfprotov5.DynamicValue, rt tftypes.Type) (tftypes.Value, error) {
	if cfg == nil {
		return tftypes.Value{}, fmt.Errorf("no config available")
	}
	cfgState, err := cfg.Unmarshal(rt)
	if err != nil {
		return tftypes.Value{}, err
	}
	cfgVal := make(map[string]tftypes.Value)
	if err := cfgState.As(&cfgVal); err != nil {
		return tftypes.Value{}, err
	}
	v, ok := cfgVal["bootstrap_connection"]
	if !ok {
		return tftypes.Value{}, fmt.Errorf("config has no \"bootstrap_connection\" attribute")
	}
	return v, nil
}

func hasConnectionBlock(resourceConfig map[string]tftypes.Value) bool {
	connVal, ok := resourceConfig["bootstrap_connection"]
	if !ok || connVal.IsNull() || !connVal.IsKnown() {
		return false
	}
	var connBlock []tftypes.Value
	if err := connVal.As(&connBlock); err != nil {
		return false
	}
	return len(connBlock) > 0 && !connBlock[0].IsNull()
}

func connectionBlockUsable(cfg *tfprotov5.DynamicValue, rt tftypes.Type) bool {
	connVal, err := connectionFromConfig(cfg, rt)
	if err != nil {
		return false
	}
	if !connVal.IsFullyKnown() {
		return false
	}
	return hasConnectionBlock(map[string]tftypes.Value{"bootstrap_connection": connVal})
}

func (ps *RawProviderServer) providerClients(stateOrConfig map[string]tftypes.Value) (*resourceClients, []*tfprotov5.Diagnostic) {
	if ps.clientConfig == nil && hasConnectionBlock(stateOrConfig) {
		return nil, []*tfprotov5.Diagnostic{{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  util.ProviderRequiredSummary,
			Detail:   util.ProviderRequiredDetail,
		}}
	}
	return &resourceClients{ps: ps}, nil
}

func (ps *RawProviderServer) clientsFor(resourceConfig map[string]tftypes.Value) (clients *resourceClients, connectionUnknown bool, diags []*tfprotov5.Diagnostic) {
	connVal, ok := resourceConfig["bootstrap_connection"]
	if !ok || connVal.IsNull() {
		return &resourceClients{ps: ps}, false, nil
	}
	if !connVal.IsFullyKnown() {
		return nil, true, nil
	}

	var connBlock []tftypes.Value
	if err := connVal.As(&connBlock); err != nil {
		return nil, false, []*tfprotov5.Diagnostic{{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  `Failed to assert type of "bootstrap_connection" block`,
			Detail:   err.Error(),
		}}
	}
	if len(connBlock) == 0 || connBlock[0].IsNull() {
		return &resourceClients{ps: ps}, false, nil
	}

	var connMap map[string]tftypes.Value
	if err := connBlock[0].As(&connMap); err != nil {
		return nil, false, []*tfprotov5.Diagnostic{{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  `Failed to extract "bootstrap_connection" block attributes`,
			Detail:   err.Error(),
		}}
	}

	clientConfig, fp, buildDiags := buildClientConfig(connMap, ps.logger, noEnvLookup)
	if len(buildDiags) > 0 {
		return nil, false, buildDiags
	}
	if clientConfig == nil {
		return nil, false, []*tfprotov5.Diagnostic{{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  util.UnusableConnectionSummary,
			Detail:   util.UnusableConnectionDetail,
		}}
	}

	key, err := hashstructure.Hash(fp, nil)
	if err != nil {
		return nil, false, []*tfprotov5.Diagnostic{{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  `Failed to hash "bootstrap_connection" block`,
			Detail:   err.Error(),
		}}
	}

	return &resourceClients{ps: ps, clientConfig: clientConfig, key: key}, false, nil
}

func (rc *resourceClients) DynamicClient() (dynamic.Interface, error) {
	if rc.clientConfig == nil {
		return rc.ps.getDynamicClient()
	}
	return rc.ps.dynamicClients.Get(rc.key, func() (dynamic.Interface, error) {
		return dynamic.NewForConfig(rc.clientConfig)
	})
}

func (rc *resourceClients) DiscoveryClient() (discovery.DiscoveryInterface, error) {
	if rc.clientConfig == nil {
		return rc.ps.getDiscoveryClient()
	}
	return rc.ps.discoveryClients.Get(rc.key, func() (discovery.DiscoveryInterface, error) {
		return discovery.NewDiscoveryClientForConfig(rc.clientConfig)
	})
}

func (rc *resourceClients) RestMapper() (meta.RESTMapper, error) {
	if rc.clientConfig == nil {
		return rc.ps.getRestMapper()
	}
	return rc.ps.restMappers.Get(rc.key, func() (meta.RESTMapper, error) {
		dc, err := rc.DiscoveryClient()
		if err != nil {
			return nil, err
		}
		cacheClient := memory.NewMemCacheClient(dc)
		return restmapper.NewDeferredDiscoveryRESTMapper(cacheClient), nil
	})
}

func (rc *resourceClients) RestClient() (rest.Interface, error) {
	if rc.clientConfig == nil {
		return rc.ps.getRestClient()
	}
	return rc.ps.restClients.Get(rc.key, func() (rest.Interface, error) {
		return rest.UnversionedRESTClientFor(rc.clientConfig)
	})
}

func (rc *resourceClients) OAPIFoundry() (openapi.Foundry, error) {
	if rc.clientConfig == nil {
		return rc.ps.getOAPIv2Foundry()
	}
	return rc.ps.oapiFoundries.Get(rc.key, func() (openapi.Foundry, error) {
		rcl, err := rc.RestClient()
		if err != nil {
			return nil, fmt.Errorf("failed get OpenAPI spec: %s", err)
		}

		rq := rcl.Verb("GET").Timeout(30*time.Second).AbsPath("openapi", "v2")
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

func (rc *resourceClients) CheckValidCredentials(ctx context.Context) []*tfprotov5.Diagnostic {
	if rc.clientConfig == nil {
		return rc.ps.checkValidCredentials(ctx)
	}
	diagnostics, _ := rc.ps.checkValidCredentialsResults.Get(rc.key, func() (diags []*tfprotov5.Diagnostic, err error) {
		rcl, err := rc.RestClient()
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to construct REST client",
				Detail:   err.Error(),
			})
			return
		}
		vpath := []string{"/apis"}
		rs := rcl.Get().AbsPath(vpath...).Do(ctx)
		if rs.Error() != nil {
			switch {
			case apierrors.IsUnauthorized(rs.Error()):
				diags = append(diags, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Invalid credentials",
					Detail:   fmt.Sprintf("The credentials configured in the resource's 'connection' block are not accepted by the API server. Error: %s\n\nSet TF_LOG=debug and look for '[InvalidClientConfiguration]' in the log to see actual configuration.", rs.Error().Error()),
				})
			default:
				diags = append(diags, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Invalid configuration for API client",
					Detail:   rs.Error().Error(),
				})
			}
			rc.ps.logger.Debug("[InvalidClientConfiguration]", "Config", dump(rc.clientConfig))
		}
		return
	})
	return diagnostics
}
