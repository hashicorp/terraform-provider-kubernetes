// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package admissionregistrationv1

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	"github.com/hashicorp/terraform-provider-kubernetes/util"

	k8s "k8s.io/client-go/kubernetes"
)

type BootstrapConnectionModel struct {
	Host                  types.String         `tfsdk:"host"`
	Username              types.String         `tfsdk:"username"`
	Password              types.String         `tfsdk:"password"`
	Insecure              types.Bool           `tfsdk:"insecure"`
	TLSServerName         types.String         `tfsdk:"tls_server_name"`
	ClientCertificate     types.String         `tfsdk:"client_certificate"`
	ClientKey             types.String         `tfsdk:"client_key"`
	ClusterCACertificate  types.String         `tfsdk:"cluster_ca_certificate"`
	ConfigPaths           []types.String       `tfsdk:"config_paths"`
	ConfigPath            types.String         `tfsdk:"config_path"`
	ConfigContext         types.String         `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String         `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String         `tfsdk:"config_context_cluster"`
	Token                 types.String         `tfsdk:"token"`
	ProxyURL              types.String         `tfsdk:"proxy_url"`
	Exec                  []BootstrapExecModel `tfsdk:"exec"`
}

type BootstrapExecModel struct {
	APIVersion types.String            `tfsdk:"api_version"`
	Command    types.String            `tfsdk:"command"`
	Env        map[string]types.String `tfsdk:"env"`
	Args       []types.String          `tfsdk:"args"`
}

// output is for kubernetes.BootstrapConnectionConfig
func (m BootstrapConnectionModel) values() map[string]interface{} {
	v := map[string]interface{}{
		"host":                     m.Host.ValueString(),
		"username":                 m.Username.ValueString(),
		"password":                 m.Password.ValueString(),
		"insecure":                 m.Insecure.ValueBool(),
		"tls_server_name":          m.TLSServerName.ValueString(),
		"client_certificate":       m.ClientCertificate.ValueString(),
		"client_key":               m.ClientKey.ValueString(),
		"cluster_ca_certificate":   m.ClusterCACertificate.ValueString(),
		"config_paths":             stringSliceValues(m.ConfigPaths),
		"config_path":              m.ConfigPath.ValueString(),
		"config_context":           m.ConfigContext.ValueString(),
		"config_context_auth_info": m.ConfigContextAuthInfo.ValueString(),
		"config_context_cluster":   m.ConfigContextCluster.ValueString(),
		"token":                    m.Token.ValueString(),
		"proxy_url":                m.ProxyURL.ValueString(),
		"exec":                     []interface{}{},
	}

	if len(m.Exec) > 0 {
		e := m.Exec[0]
		env := map[string]interface{}{}
		for k, ev := range e.Env {
			env[k] = ev.ValueString()
		}
		v["exec"] = []interface{}{map[string]interface{}{
			"api_version": e.APIVersion.ValueString(),
			"command":     e.Command.ValueString(),
			"env":         env,
			"args":        stringSliceValues(e.Args),
		}}
	}

	return v
}

func stringSliceValues(in []types.String) []interface{} {
	out := []interface{}{}
	for _, v := range in {
		out = append(out, v.ValueString())
	}
	return out
}

func bootstrapClientset(sdkv2Meta func() any, conn []BootstrapConnectionModel) (*k8s.Clientset, error) {
	if len(conn) == 0 {
		return sdkv2Meta().(kubernetes.KubeClientsets).MainClientset()
	}
	cfg, err := kubernetes.BootstrapConnectionConfig(conn[0].values())
	if err != nil {
		return nil, err
	}
	return k8s.NewForConfig(cfg)
}

func bootstrapConnectionBlock() schema.Block {
	return schema.ListNestedBlock{
		MarkdownDescription: util.BootstrapConnectionDescription,
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"exec": schema.ListNestedBlock{
					MarkdownDescription: util.BootstrapConnectionExecDescription,
					Validators: []validator.List{
						listvalidator.SizeAtMost(1),
					},
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"api_version": schema.StringAttribute{
								MarkdownDescription: util.BootstrapConnectionExecAPIVersionDescription,
								Required:            true,
								WriteOnly:           true,
							},
							"command": schema.StringAttribute{
								MarkdownDescription: util.BootstrapConnectionExecCommandDescription,
								Required:            true,
								WriteOnly:           true,
							},
							"env": schema.MapAttribute{
								MarkdownDescription: util.BootstrapConnectionExecEnvDescription,
								ElementType:         types.StringType,
								Optional:            true,
								WriteOnly:           true,
								Sensitive:           true,
							},
							"args": schema.ListAttribute{
								MarkdownDescription: util.BootstrapConnectionExecArgsDescription,
								ElementType:         types.StringType,
								Optional:            true,
								WriteOnly:           true,
							},
						},
					},
				},
			},
			Attributes: map[string]schema.Attribute{
				"host": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionHostDescription,
					Optional:            true,
					WriteOnly:           true,
				},
				"username": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionUsernameDescription,
					Optional:            true,
					WriteOnly:           true,
				},
				"password": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionPasswordDescription,
					Optional:            true,
					WriteOnly:           true,
					Sensitive:           true,
				},
				"insecure": schema.BoolAttribute{
					MarkdownDescription: util.BootstrapConnectionInsecureDescription,
					Optional:            true,
					WriteOnly:           true,
				},
				"tls_server_name": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionTLSServerNameDescription,
					Optional:            true,
					WriteOnly:           true,
				},
				"client_certificate": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionClientCertificateDescription,
					Optional:            true,
					WriteOnly:           true,
					Sensitive:           true,
				},
				"client_key": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionClientKeyDescription,
					Optional:            true,
					WriteOnly:           true,
					Sensitive:           true,
				},
				"cluster_ca_certificate": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionClusterCACertificateDescription,
					Optional:            true,
					WriteOnly:           true,
				},
				"config_paths": schema.ListAttribute{
					MarkdownDescription: util.BootstrapConnectionConfigPathsDescription,
					ElementType:         types.StringType,
					Optional:            true,
					WriteOnly:           true,
				},
				"config_path": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionConfigPathDescription,
					Optional:            true,
					WriteOnly:           true,
				},
				"config_context": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionConfigContextDescription,
					Optional:            true,
					WriteOnly:           true,
				},
				"config_context_auth_info": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionConfigContextAuthInfoDescription,
					Optional:            true,
					WriteOnly:           true,
				},
				"config_context_cluster": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionConfigContextClusterDescription,
					Optional:            true,
					WriteOnly:           true,
				},
				"token": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionTokenDescription,
					Optional:            true,
					WriteOnly:           true,
					Sensitive:           true,
				},
				"proxy_url": schema.StringAttribute{
					MarkdownDescription: util.BootstrapConnectionProxyURLDescription,
					Optional:            true,
					WriteOnly:           true,
				},
			},
		},
	}
}
