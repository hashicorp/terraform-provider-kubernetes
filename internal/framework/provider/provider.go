// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure KubernetesProvider satisfies various provider interfaces.
var _ provider.Provider = &KubernetesProvider{}

// KubernetesProvider defines the provider implementation.
type KubernetesProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// KubernetesProviderModel describes the provider data model.
type KubernetesProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Insecure types.Bool   `tfsdk:"insecure"`

	TLSServerName        types.String `tfsdk:"tls_server_name"`
	ClientCertificate    types.String `tfsdk:"client_certificate"`
	ClientKey            types.String `tfsdk:"client_key"`
	ClusterCACertificate types.String `tfsdk:"cluster_ca_certificate"`

	ConfigPaths []types.String `tfsdk:"config_paths"`
	ConfigPath  types.String   `tfsdk:"config_path"`

	ConfigContext         types.String `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String `tfsdk:"config_context_cluster"`

	Token types.String `tfsdk:"token"`

	ProxyURL types.String `tfsdk:"proxy_url"`

	ConfigDataBase64 types.String `tfsdk:"config_data_base64"`

	IgnoreAnnotations types.List `tfsdk:"ignore_annotations"`
	IgnoreLabels      types.List `tfsdk:"ignore_labels"`

	Exec []struct {
		APIVersion types.String            `tfsdk:"api_version"`
		Command    types.String            `tfsdk:"command"`
		Env        map[string]types.String `tfsdk:"env"`
		Args       []types.String          `tfsdk:"args"`
	} `tfsdk:"exec"`

	Experiments []struct {
		ManifestResource types.Bool `tfsdk:"manifest_resource"`
	} `tfsdk:"experiments"`
}

func (p *KubernetesProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
	resp.Version = p.version
}

func (p *KubernetesProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The hostname (in form of URI) of Kubernetes master.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
				Optional:    true,
			},
			"insecure": schema.BoolAttribute{
				Description: "Whether server should be accessed without verifying the TLS certificate.",
				Optional:    true,
			},
			"tls_server_name": schema.StringAttribute{
				Description: "Server name passed to the server for SNI and is used in the client to check server certificates against.",
				Optional:    true,
			},
			"client_certificate": schema.StringAttribute{
				Description: "PEM-encoded client certificate for TLS authentication.",
				Optional:    true,
			},
			"client_key": schema.StringAttribute{
				Description: "PEM-encoded client certificate key for TLS authentication.",
				Optional:    true,
			},
			"cluster_ca_certificate": schema.StringAttribute{
				Description: "PEM-encoded root certificates bundle for TLS authentication.",
				Optional:    true,
			},
			"config_paths": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
				Optional:    true,
			},
			"config_path": schema.StringAttribute{
				Description: "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
				Optional:    true,
			},
			"config_context": schema.StringAttribute{
				Description: "",
				Optional:    true,
			},
			"config_context_auth_info": schema.StringAttribute{
				Description: "",
				Optional:    true,
			},
			"config_context_cluster": schema.StringAttribute{
				Description: "",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "Token to authenticate an service account",
				Optional:    true,
			},
			"proxy_url": schema.StringAttribute{
				Description: "URL to the proxy to be used for all API requests",
				Optional:    true,
			},
			"config_data_base64": schema.StringAttribute{
				Description: "Kubeconfig content in base64 format",
				Optional:    true,
			},
			"ignore_annotations": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "List of Kubernetes metadata annotations to ignore across all resources handled by this provider for situations where external systems are managing certain resource annotations. Each item is a regular expression.",
				Optional:    true,
			},
			"ignore_labels": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "List of Kubernetes metadata labels to ignore across all resources handled by this provider for situations where external systems are managing certain resource labels. Each item is a regular expression.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"exec": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"api_version": schema.StringAttribute{
							Required: true,
						},
						"command": schema.StringAttribute{
							Required: true,
						},
						"env": schema.MapAttribute{
							ElementType: types.StringType,
							Optional:    true,
						},
						"args": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
						},
					},
				},
			},
			"experiments": schema.ListNestedBlock{
				Description: "Enable and disable experimental features.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"manifest_resource": schema.BoolAttribute{
							Description: "Enable the `kubernetes_manifest` resource.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (p *KubernetesProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *KubernetesProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) provider.Provider {
	return &KubernetesProvider{
		version: version,
	}
}
