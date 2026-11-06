// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	restclient "k8s.io/client-go/rest"

	"github.com/hashicorp/terraform-provider-kubernetes/util"
)

const bootstrapConnectionKey = util.BootstrapConnectionKey

type crudFunc = func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics

func bootstrapConnectionSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: util.BootstrapConnectionDescription,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"host": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Description: util.BootstrapConnectionHostDescription,
				},
				"username": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Description: util.BootstrapConnectionUsernameDescription,
				},
				"password": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Sensitive:   true,
					Description: util.BootstrapConnectionPasswordDescription,
				},
				"insecure": {
					Type:        schema.TypeBool,
					Optional:    true,
					WriteOnly:   true,
					Description: util.BootstrapConnectionInsecureDescription,
				},
				"tls_server_name": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Description: util.BootstrapConnectionTLSServerNameDescription,
				},
				"client_certificate": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Sensitive:   true,
					Description: util.BootstrapConnectionClientCertificateDescription,
				},
				"client_key": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Sensitive:   true,
					Description: util.BootstrapConnectionClientKeyDescription,
				},
				"cluster_ca_certificate": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Description: util.BootstrapConnectionClusterCACertificateDescription,
				},
				"config_paths": {
					Type:        schema.TypeList,
					Optional:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: util.BootstrapConnectionConfigPathsDescription,
				},
				"config_path": {
					Type:          schema.TypeString,
					Optional:      true,
					WriteOnly:     true,
					ConflictsWith: []string{bootstrapConnectionKey + ".0.config_paths"},
					Description:   util.BootstrapConnectionConfigPathDescription,
				},
				"config_context": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Description: util.BootstrapConnectionConfigContextDescription,
				},
				"config_context_auth_info": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Description: util.BootstrapConnectionConfigContextAuthInfoDescription,
				},
				"config_context_cluster": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Description: util.BootstrapConnectionConfigContextClusterDescription,
				},
				"token": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Sensitive:   true,
					Description: util.BootstrapConnectionTokenDescription,
				},
				"proxy_url": {
					Type:        schema.TypeString,
					Optional:    true,
					WriteOnly:   true,
					Description: util.BootstrapConnectionProxyURLDescription,
				},
				"exec": {
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Description: util.BootstrapConnectionExecDescription,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"api_version": {
								Type:        schema.TypeString,
								Required:    true,
								WriteOnly:   true,
								Description: util.BootstrapConnectionExecAPIVersionDescription,
							},
							"command": {
								Type:        schema.TypeString,
								Required:    true,
								WriteOnly:   true,
								Description: util.BootstrapConnectionExecCommandDescription,
							},
							"env": {
								Type:      schema.TypeMap,
								Optional:  true,
								WriteOnly: true,
								Sensitive: true,
								// Elem must stay unset: the SDK rejects WriteOnly on a
								// map with an Elem. Values default to strings.
								Description: util.BootstrapConnectionExecEnvDescription,
							},
							"args": {
								Type:        schema.TypeList,
								Optional:    true,
								Elem:        &schema.Schema{Type: schema.TypeString},
								Description: util.BootstrapConnectionExecArgsDescription,
							},
						},
					},
				},
			},
		},
	}
}

func injectBootstrapConnection(r *schema.Resource) {
	if r == nil {
		return
	}
	if _, exists := r.Schema[bootstrapConnectionKey]; exists {
		return
	}
	r.Schema[bootstrapConnectionKey] = bootstrapConnectionSchema()

	r.CreateContext = withBootstrapConnection(r.CreateContext)
	r.ReadContext = requiresProviderConnection(r.ReadContext)
	r.DeleteContext = requiresProviderConnection(r.DeleteContext)

	if !hasUpdate(r) {
		r.UpdateContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			return nil
		}
		return
	}
	r.UpdateContext = requiresProviderConnection(r.UpdateContext)
}

//nolint:staticcheck // SA1019: reading the deprecated fields is the point.
func hasUpdate(r *schema.Resource) bool {
	return r.UpdateContext != nil || r.Update != nil || r.UpdateWithoutTimeout != nil
}

func withBootstrapConnection(f crudFunc) crudFunc {
	if f == nil {
		return nil
	}
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		m, diags := bootstrapMeta(d, meta)
		if diags.HasError() {
			return diags
		}
		return append(diags, f(ctx, d, m)...)
	}
}

func requiresProviderConnection(f crudFunc) crudFunc {
	if f == nil {
		return nil
	}
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		if stateHasBootstrapConnection(d) && !ProviderIsConfigured(meta) {
			return diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       util.ProviderRequiredSummary,
				Detail:        util.ProviderRequiredDetail,
				AttributePath: cty.Path{}.GetAttr(bootstrapConnectionKey),
			}}
		}
		return f(ctx, d, meta)
	}
}

func bootstrapMeta(d *schema.ResourceData, meta interface{}) (interface{}, diag.Diagnostics) {
	block, ok := bootstrapConnectionBlock(d)
	if !ok {
		return meta, nil
	}

	pm, ok := meta.(providerMetadata)
	if !ok {
		return meta, nil
	}

	if !block.IsWhollyKnown() {
		return nil, diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  `"bootstrap_connection" block contains values that are not yet known`,
			Detail: "This should not happen at apply time. If this error is reproducible, please report this " +
				"issue to the provider maintainers.",
			AttributePath: cty.Path{}.GetAttr(bootstrapConnectionKey),
		}}
	}

	cfg, diags := initializeConfiguration(&ctyConfigGetter{v: block}, false)
	if diags.HasError() {
		return nil, diags
	}
	if cfg == nil {
		return nil, append(diags, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       `Invalid "bootstrap_connection" block`,
			Detail:        util.UnusableConnectionDetail,
			AttributePath: cty.Path{}.GetAttr(bootstrapConnectionKey),
		})
	}

	if pm.config != nil {
		cfg.UserAgent = pm.config.UserAgent
		cfg.WrapTransport = pm.config.WrapTransport
	}

	// Replace the config and drop any clients cached against the provider's.
	pm.config = cfg
	pm.mainClientset = nil
	pm.aggregatorClientset = nil
	pm.dynamicClient = nil
	pm.discoveryClient = nil

	return pm, diags
}

func bootstrapConnectionBlock(d *schema.ResourceData) (cty.Value, bool) {
	raw := d.GetRawConfig()
	if raw.IsNull() || !raw.IsKnown() {
		return cty.NilVal, false
	}
	t := raw.Type()
	if !t.IsObjectType() || !t.HasAttribute(bootstrapConnectionKey) {
		return cty.NilVal, false
	}
	list := raw.GetAttr(bootstrapConnectionKey)
	if list.IsNull() || !list.IsKnown() || list.LengthInt() == 0 {
		return cty.NilVal, false
	}
	block := list.Index(cty.NumberIntVal(0))
	if block.IsNull() {
		return cty.NilVal, false
	}
	return block, true
}

func stateHasBootstrapConnection(d *schema.ResourceData) bool {
	v, ok := d.GetOk(bootstrapConnectionKey)
	if !ok {
		return false
	}
	l, ok := v.([]interface{})
	return ok && len(l) > 0
}

func ProviderIsConfigured(meta interface{}) bool {
	pm, ok := meta.(providerMetadata)
	if !ok {
		// Unknown metadata type - don't second-guess it.
		return true
	}
	return pm.config != nil && pm.config.Host != ""
}

func BootstrapConnectionConfig(values map[string]interface{}) (*restclient.Config, error) {
	cfg, diags := initializeConfiguration(mapConfigGetter(values), false)
	for _, d := range diags {
		if d.Severity == diag.Error {
			return nil, fmt.Errorf("%s: %s", d.Summary, d.Detail)
		}
	}
	if cfg == nil {
		return nil, errors.New(util.UnusableConnectionDetail)
	}
	return cfg, nil
}

type mapConfigGetter map[string]interface{}

var _ configGetter = mapConfigGetter(nil)

func (g mapConfigGetter) Get(key string) interface{} {
	return g[key]
}

func (g mapConfigGetter) GetOk(key string) (interface{}, bool) {
	v, ok := g[key]
	if !ok {
		return nil, false
	}
	return v, !isZeroNative(v)
}

type ctyConfigGetter struct {
	v cty.Value
}

var _ configGetter = (*ctyConfigGetter)(nil)

func (g *ctyConfigGetter) attr(key string) (cty.Value, bool) {
	t := g.v.Type()
	if !t.IsObjectType() || !t.HasAttribute(key) {
		return cty.NilVal, false
	}
	av := g.v.GetAttr(key)
	if av.IsNull() || !av.IsKnown() {
		return cty.NilVal, false
	}
	return av, true
}

func (g *ctyConfigGetter) Get(key string) interface{} {
	av, ok := g.attr(key)
	if !ok {
		return nil
	}
	return ctyToNative(av)
}

func (g *ctyConfigGetter) GetOk(key string) (interface{}, bool) {
	av, ok := g.attr(key)
	if !ok {
		return nil, false
	}
	nv := ctyToNative(av)
	return nv, !isZeroNative(nv)
}

func ctyToNative(v cty.Value) interface{} {
	t := v.Type()
	unset := v.IsNull() || !v.IsKnown()

	switch {
	case t == cty.String:
		if unset {
			return ""
		}
		return v.AsString()

	case t == cty.Bool:
		if unset {
			return false
		}
		return v.True()

	case t == cty.Number:
		if unset {
			return 0
		}
		i, _ := v.AsBigFloat().Int64()
		return int(i)

	case t.IsListType(), t.IsSetType(), t.IsTupleType():
		out := []interface{}{}
		if unset {
			return out
		}
		for it := v.ElementIterator(); it.Next(); {
			_, ev := it.Element()
			out = append(out, ctyToNative(ev))
		}
		return out

	case t.IsObjectType():
		out := map[string]interface{}{}
		for name, at := range t.AttributeTypes() {
			if unset {
				out[name] = ctyToNative(cty.NullVal(at))
				continue
			}
			out[name] = ctyToNative(v.GetAttr(name))
		}
		return out

	case t.IsMapType():
		out := map[string]interface{}{}
		if unset {
			return out
		}
		for it := v.ElementIterator(); it.Next(); {
			kv, ev := it.Element()
			out[kv.AsString()] = ctyToNative(ev)
		}
		return out
	}

	return nil
}

func isZeroNative(v interface{}) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case bool:
		return !x
	case int:
		return x == 0
	case []interface{}:
		return len(x) == 0
	case map[string]interface{}:
		return len(x) == 0
	}
	return false
}
