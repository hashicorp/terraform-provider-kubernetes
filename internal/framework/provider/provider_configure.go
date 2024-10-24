// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/client"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mitchellh/go-homedir"

	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func (p *KubernetesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	if os.Getenv("TF_X_KUBERNETES_CODEGEN_PLUGIN6") != "1" {
		// NOTE don't configure the client unless the plugin6 experiment is enabled
		return
	}

	var data KubernetesProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := newKubernetesClientConfig(ctx, data)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("failed to initilize Kubernetes client configuration", err.Error()))
	}

	// FIXME make a helper function for this
	ignoreLabels := make([]string, len(data.IgnoreLabels))
	for i, s := range data.IgnoreLabels {
		ignoreLabels[i] = s.ValueString()
	}
	ignoreAnnotations := make([]string, len(data.IgnoreAnnotations))
	for i, s := range data.IgnoreAnnotations {
		ignoreAnnotations[i] = s.ValueString()
	}

	resp.ResourceData = client.NewKubernetesClientGetter(cfg, ignoreLabels, ignoreAnnotations)
}

func newKubernetesClientConfig(ctx context.Context, data KubernetesProviderModel) (*restclient.Config, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}
	if v := data.ConfigPath.ValueString(); v != "" {
		configPaths = []string{v}
	} else if len(data.ConfigPaths) > 0 {
		for _, p := range data.ConfigPaths {
			configPaths = append(configPaths, p.ValueString())
		}
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		configPaths = filepath.SplitList(v)
	}

	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}

			tflog.Debug(ctx, "Using kubeconfig file", map[string]interface{}{"path": path})
			expandedPaths = append(expandedPaths, path)
		}
		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		ctxSuffix := "; default context"

		kubectx := data.ConfigContext.ValueString()
		authInfo := data.ConfigContextAuthInfo.ValueString()
		cluster := data.ConfigContextCluster.ValueString()
		if kubectx != "" || authInfo != "" || cluster != "" {
			ctxSuffix = "; overridden context"
			if kubectx != "" {
				overrides.CurrentContext = kubectx
				ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
				tflog.Debug(ctx, "Using custom current context", map[string]interface{}{"context": overrides.CurrentContext})
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfo != "" {
				overrides.Context.AuthInfo = authInfo
				ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
			}
			if cluster != "" {
				overrides.Context.Cluster = cluster
				ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
			}
			tflog.Debug(ctx, "Using overridden context", map[string]interface{}{"context": overrides.Context})
		}
	}

	// Overriding with static configuration
	overrides.ClusterInfo.InsecureSkipTLSVerify = data.Insecure.ValueBool()
	overrides.ClusterInfo.TLSServerName = data.TLSServerName.ValueString()
	overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(data.ClusterCACertificate.ValueString()).Bytes()
	overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(data.ClientCertificate.ValueString()).Bytes()

	if v := data.Host.ValueString(); v != "" {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := restclient.DefaultServerURL(v, "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host: %s", err)
		}

		overrides.ClusterInfo.Server = host.String()
	}

	overrides.AuthInfo.Username = data.Username.ValueString()
	overrides.AuthInfo.Password = data.Password.ValueString()
	overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(data.ClientKey.ValueString()).Bytes()
	overrides.AuthInfo.Token = data.Token.ValueString()

	overrides.ClusterDefaults.ProxyURL = data.ProxyURL.ValueString()

	if len(data.Exec) > 0 {
		execData := data.Exec[0]

		exec := &clientcmdapi.ExecConfig{}
		exec.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
		exec.APIVersion = execData.APIVersion.ValueString()
		exec.Command = execData.Command.ValueString()
		exec.Args = expandStringSlice(execData.Args)
		for kk, vv := range execData.Env {
			exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: kk, Value: vv.ValueString()})
		}

		overrides.AuthInfo.Exec = exec
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	cfg, err := cc.ClientConfig()
	if err != nil {
		tflog.Warn(ctx, "Invalid provider configuration was supplied. Provider operations likely to fail", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, nil
	}
	return cfg, nil
}

func expandStringSlice(s []types.String) []string {
	v := []string{}
	for _, vv := range s {
		v = append(v, vv.ValueString())
	}
	return v
}
