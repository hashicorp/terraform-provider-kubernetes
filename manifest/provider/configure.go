// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/runtime"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const minTFVersion string = "v0.14.8"

// ConfigureProvider function
func (s *RawProviderServer) ConfigureProvider(ctx context.Context, req *tfprotov5.ConfigureProviderRequest) (*tfprotov5.ConfigureProviderResponse, error) {
	response := &tfprotov5.ConfigureProviderResponse{}
	diags := []*tfprotov5.Diagnostic{}
	var providerConfig map[string]tftypes.Value
	var err error

	s.hostTFVersion = "v" + req.TerraformVersion

	// transform provider config schema into tftype.Type and unmarshal the given config into a tftypes.Value
	cfgType := GetObjectTypeFromSchema(GetProviderConfigSchema())
	cfgVal, err := req.Config.Unmarshal(cfgType)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to decode ConfigureProvider request parameter",
			Detail:   err.Error(),
		})
		return response, nil
	}
	err = cfgVal.As(&providerConfig)
	if err != nil {
		// invalid configuration schema - this shouldn't happen, bail out now
		response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Provider configuration: failed to extract 'config_path' value",
			Detail:   err.Error(),
		})
		return response, nil
	}

	providerEnabled := true
	if !providerConfig["experiments"].IsNull() && providerConfig["experiments"].IsKnown() {
		var experimentsBlock []tftypes.Value
		err = providerConfig["experiments"].As(&experimentsBlock)
		if err != nil {
			// invalid configuration schema - this shouldn't happen, bail out now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to extract 'experiments' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		if len(experimentsBlock) > 0 {
			var experimentsObj map[string]tftypes.Value
			err := experimentsBlock[0].As(&experimentsObj)
			if err != nil {
				// invalid configuration schema - this shouldn't happen, bail out now
				response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Provider configuration: failed to extract 'experiments' value",
					Detail:   err.Error(),
				})
				return response, nil
			}
			if !experimentsObj["manifest_resource"].IsNull() && experimentsObj["manifest_resource"].IsKnown() {
				err = experimentsObj["manifest_resource"].As(&providerEnabled)
				if err != nil {
					// invalid attribute type - this shouldn't happen, bail out for now
					response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Provider configuration: failed to extract 'manifest_resource' value",
						Detail:   err.Error(),
					})
					return response, nil
				}
			}
		}
	}
	if v := os.Getenv("TF_X_KUBERNETES_MANIFEST_RESOURCE"); v != "" {
		providerEnabled, err = strconv.ParseBool(v)
		if err != nil {
			if err != nil {
				// invalid attribute type - this shouldn't happen, bail out for now
				response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Provider configuration: failed to parse boolean from `TF_X_KUBERNETES_MANIFEST_RESOURCE` env var",
					Detail:   err.Error(),
				})
				return response, nil
			}
		}
	}
	s.providerEnabled = providerEnabled

	if !providerEnabled {
		// Configure should be a noop when not enabled in the provider block
		return response, nil
	}

	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	// Handle 'config_path' attribute
	//
	var configPath string
	if !providerConfig["config_path"].IsNull() && providerConfig["config_path"].IsKnown() {
		err = providerConfig["config_path"].As(&configPath)
		if err != nil {
			// invalid attribute - this shouldn't happen, bail out now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to extract 'config_path' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
	}
	// check environment - this overrides any value found in provider configuration
	if configPathEnv, ok := os.LookupEnv("KUBE_CONFIG_PATH"); ok && configPathEnv != "" {
		configPath = configPathEnv
	}
	if len(configPath) > 0 {
		configPathAbs, err := homedir.Expand(configPath)
		if err == nil {
			_, err = os.Stat(configPathAbs)
		}
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityInvalid,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   fmt.Sprintf("'config_path' refers to an invalid path: %q: %v", configPathAbs, err),
			})
		}
		loader.ExplicitPath = configPathAbs
	}
	// Handle 'config_paths' attribute
	//
	var precedence []string
	if !providerConfig["config_paths"].IsNull() && providerConfig["config_paths"].IsKnown() {
		var configPaths []tftypes.Value
		err = providerConfig["config_paths"].As(&configPaths)
		if err != nil {
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to extract 'config_paths' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		for _, p := range configPaths {
			var pp string
			p.As(&pp)
			precedence = append(precedence, pp)
		}
	}
	//
	// check environment for KUBE_CONFIG_PATHS
	if configPathsEnv, ok := os.LookupEnv("KUBE_CONFIG_PATHS"); ok && configPathsEnv != "" {
		precedence = filepath.SplitList(configPathsEnv)
	}
	if len(precedence) > 0 {
		for i, p := range precedence {
			absPath, err := homedir.Expand(p)
			if err == nil {
				_, err = os.Stat(absPath)
			}
			if err != nil {
				diags = append(diags, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityInvalid,
					Summary:  "Invalid attribute in provider configuration",
					Detail:   fmt.Sprintf("'config_paths' refers to an invalid path: %q: %v", absPath, err),
				})
			}
			precedence[i] = absPath
		}
		loader.Precedence = precedence
	}

	// Handle 'client_certificate' attribute
	//
	var clientCertificate string
	if !providerConfig["client_certificate"].IsNull() && providerConfig["client_certificate"].IsKnown() {
		err = providerConfig["client_certificate"].As(&clientCertificate)
		if err != nil {
			response.Diagnostics = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityInvalid,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'client_certificate' type cannot be asserted: " + err.Error(),
			})
			return response, nil
		}
	}
	if clientCrtEnv, ok := os.LookupEnv("KUBE_CLIENT_CERT_DATA"); ok && clientCrtEnv != "" {
		clientCertificate = clientCrtEnv
	}
	if len(clientCertificate) > 0 {
		ccPEM, _ := pem.Decode([]byte(clientCertificate))
		if ccPEM == nil || ccPEM.Type != "CERTIFICATE" {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityInvalid,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'client_certificate' is not a valid PEM encoded certificate",
			})
		}
		overrides.AuthInfo.ClientCertificateData = []byte(clientCertificate)
	}

	// Handle 'cluster_ca_certificate' attribute
	//
	var clusterCaCertificate string
	if !providerConfig["cluster_ca_certificate"].IsNull() && providerConfig["cluster_ca_certificate"].IsKnown() {
		err = providerConfig["cluster_ca_certificate"].As(&clusterCaCertificate)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to extract 'cluster_ca_certificate' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
	}
	if clusterCAEnv, ok := os.LookupEnv("KUBE_CLUSTER_CA_CERT_DATA"); ok && clusterCAEnv != "" {
		clusterCaCertificate = clusterCAEnv
	}
	if len(clusterCaCertificate) > 0 {
		ca, _ := pem.Decode([]byte(clusterCaCertificate))
		if ca == nil || ca.Type != "CERTIFICATE" {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityInvalid,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'cluster_ca_certificate' is not a valid PEM encoded certificate",
			})
		}
		overrides.ClusterInfo.CertificateAuthorityData = []byte(clusterCaCertificate)
	}

	// Handle 'insecure' attribute
	//
	var insecure bool
	if !providerConfig["insecure"].IsNull() && providerConfig["insecure"].IsKnown() {
		err = providerConfig["insecure"].As(&insecure)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'insecure' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
	}
	if insecureEnv, ok := os.LookupEnv("KUBE_INSECURE"); ok && insecureEnv != "" {
		iv, err := strconv.ParseBool(insecureEnv)
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityInvalid,
				Summary:  "Invalid provider configuration",
				Detail:   "Environment variable KUBE_INSECURE contains invalid value: " + err.Error(),
			})
		} else {
			insecure = iv
		}
	}
	overrides.ClusterInfo.InsecureSkipTLSVerify = insecure

	// Handle 'tls_server_name' attribute
	//
	var tlsServerName string
	if !providerConfig["tls_server_name"].IsNull() && providerConfig["tls_server_name"].IsKnown() {
		err = providerConfig["tls_server_name"].As(&tlsServerName)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'tls_server_name' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		overrides.ClusterInfo.TLSServerName = tlsServerName
	}
	if tlsServerName, ok := os.LookupEnv("KUBE_TLS_SERVER_NAME"); ok && tlsServerName != "" {
		overrides.ClusterInfo.TLSServerName = tlsServerName
	}

	hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
	hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
	defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify

	// Handle 'host' attribute
	//
	var host string
	if !providerConfig["host"].IsNull() && providerConfig["host"].IsKnown() {
		err = providerConfig["host"].As(&host)
		if err != nil {
			// invalid attribute path - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to extract 'host' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
	}
	// check environment - this overrides any value found in provider configuration
	if hostEnv, ok := os.LookupEnv("KUBE_HOST"); ok && hostEnv != "" {
		host = hostEnv
	}
	if len(host) > 0 {
		_, err = url.ParseRequestURI(host)
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityInvalid,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'host' is not a valid URL",
			})
		}
		hostURL, _, err := rest.DefaultServerURL(host, "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			response.Diagnostics = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityInvalid,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "Invalid value for 'host': " + err.Error(),
			})
			return response, nil
		}
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		overrides.ClusterInfo.Server = hostURL.String()
	}

	// Handle 'client_key' attribute
	//
	var clientKey string
	if !providerConfig["client_key"].IsNull() && providerConfig["client_key"].IsKnown() {
		err = providerConfig["client_key"].As(&clientKey)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: ",
				Detail:   "Failed to extract 'client_key' value" + err.Error(),
			})
			return response, nil
		}
	}
	// check environment - this overrides any value found in provider configuration
	if clientKeyEnv, ok := os.LookupEnv("KUBE_CLIENT_KEY_DATA"); ok && clientKeyEnv != "" {
		clientKey = clientKeyEnv
	}
	if len(clientKey) > 0 {
		ck, _ := pem.Decode([]byte(clientKey))
		if ck == nil || !strings.Contains(ck.Type, "PRIVATE KEY") {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityInvalid,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'client_key' is not a valid PEM encoded private key",
			})
		}
		overrides.AuthInfo.ClientKeyData = []byte(clientKey)
	}

	if len(diags) > 0 {
		response.Diagnostics = diags
		return response, nil
	}

	// Handle 'config_context' attribute
	//
	var cfgContext string
	if !providerConfig["config_context"].IsNull() && providerConfig["config_context"].IsKnown() {
		err = providerConfig["config_context"].As(&cfgContext)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'config_context' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		overrides.CurrentContext = cfgContext
	}
	if cfgContext, ok := os.LookupEnv("KUBE_CTX"); ok && cfgContext != "" {
		overrides.CurrentContext = cfgContext
	}

	overrides.Context = clientcmdapi.Context{}

	// Handle 'config_context_cluster' attribute
	//
	var cfgCtxCluster string
	if !providerConfig["config_context_cluster"].IsNull() && providerConfig["config_context_cluster"].IsKnown() {
		err = providerConfig["config_context_cluster"].As(&cfgCtxCluster)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'config_context_cluster' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		overrides.Context.Cluster = cfgCtxCluster
	}
	if cfgCtxCluster, ok := os.LookupEnv("KUBE_CTX_CLUSTER"); ok && cfgCtxCluster != "" {
		overrides.Context.Cluster = cfgCtxCluster
	}

	// Handle 'config_context_user' attribute
	//
	var cfgContextAuthInfo *string
	if !providerConfig["config_context_user"].IsNull() && providerConfig["config_context_user"].IsKnown() {
		err = providerConfig["config_context_user"].As(&cfgContextAuthInfo)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'config_context_user' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		if cfgContextAuthInfo != nil {
			overrides.Context.AuthInfo = *cfgContextAuthInfo
		}
	}
	if cfgContextAuthInfoEnv, ok := os.LookupEnv("KUBE_CTX_AUTH_INFO"); ok && cfgContextAuthInfoEnv != "" {
		overrides.Context.AuthInfo = cfgContextAuthInfoEnv
	}

	var username string
	if !providerConfig["username"].IsNull() && providerConfig["username"].IsKnown() {
		err = providerConfig["username"].As(&username)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'username' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		overrides.AuthInfo.Username = username
	}
	if username, ok := os.LookupEnv("KUBE_USERNAME"); ok && username != "" {
		overrides.AuthInfo.Username = username
	}

	var password string
	if !providerConfig["password"].IsNull() && providerConfig["password"].IsKnown() {
		err = providerConfig["password"].As(&password)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'password' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		overrides.AuthInfo.Password = password
	}
	if password, ok := os.LookupEnv("KUBE_PASSWORD"); ok && password != "" {
		overrides.AuthInfo.Password = password
	}

	var token string
	if !providerConfig["token"].IsNull() && providerConfig["token"].IsKnown() {
		err = providerConfig["token"].As(&token)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'token' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		overrides.AuthInfo.Token = token
	}
	if token, ok := os.LookupEnv("KUBE_TOKEN"); ok && token != "" {
		overrides.AuthInfo.Token = token
	}

	var proxyURL string
	if !providerConfig["proxy_url"].IsNull() && providerConfig["proxy_url"].IsKnown() {
		err = providerConfig["proxy_url"].As(&proxyURL)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'proxy_url' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		overrides.ClusterDefaults.ProxyURL = proxyURL
	}
	if proxyUrl, ok := os.LookupEnv("KUBE_PROXY_URL"); ok && proxyUrl != "" {
		overrides.ClusterDefaults.ProxyURL = proxyURL
	}

	if !providerConfig["exec"].IsNull() && providerConfig["exec"].IsKnown() {
		var execBlock []tftypes.Value
		err = providerConfig["exec"].As(&execBlock)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'exec' value",
				Detail:   err.Error(),
			})
			return response, nil
		}
		execCfg := clientcmdapi.ExecConfig{}
		execCfg.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
		if len(execBlock) > 0 {
			var execObj map[string]tftypes.Value
			err := execBlock[0].As(&execObj)
			if err != nil {
				response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  `Provider configuration: failed to assert type of "exec" block`,
					Detail:   err.Error(),
				})
				return response, nil
			}
			if !execObj["api_version"].IsNull() && execObj["api_version"].IsKnown() {
				var apiv string
				err = execObj["api_version"].As(&apiv)
				if err != nil {
					// invalid attribute type - this shouldn't happen, bail out for now
					response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Provider configuration: failed to assert type of 'api_version' value",
						Detail:   err.Error(),
					})
					return response, nil
				}
				execCfg.APIVersion = apiv
			}
			if !execObj["command"].IsNull() && execObj["command"].IsKnown() {
				var cmd string
				err = execObj["command"].As(&cmd)
				if err != nil {
					// invalid attribute type - this shouldn't happen, bail out for now
					response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Provider configuration: failed to assert type of 'command' value",
						Detail:   err.Error(),
					})
					return response, nil
				}
				execCfg.Command = cmd
			}
			if !execObj["args"].IsNull() && execObj["args"].IsFullyKnown() {
				var xcmdArgs []tftypes.Value
				err = execObj["args"].As(&xcmdArgs)
				if err != nil {
					// invalid attribute type - this shouldn't happen, bail out for now
					response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Provider configuration: failed to assert type of 'args' value",
						Detail:   err.Error(),
					})
					return response, nil
				}
				execCfg.Args = make([]string, 0, len(xcmdArgs))
				for _, arg := range xcmdArgs {
					var v string
					err := arg.As(&v)
					if err != nil {
						// invalid attribute type - this shouldn't happen, bail out for now
						response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
							Severity: tfprotov5.DiagnosticSeverityError,
							Summary:  "Provider configuration: failed to assert type of element in 'args' value",
							Detail:   err.Error(),
						})
						return response, nil
					}
					execCfg.Args = append(execCfg.Args, v)
				}
			}
			if !execObj["env"].IsNull() && execObj["env"].IsFullyKnown() {
				var xcmdEnvs map[string]tftypes.Value
				err = execObj["env"].As(&xcmdEnvs)
				if err != nil {
					// invalid attribute type - this shouldn't happen, bail out for now
					response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Provider configuration: failed to assert type of element in 'env' value",
						Detail:   err.Error(),
					})
					return response, nil
				}
				execCfg.Env = make([]clientcmdapi.ExecEnvVar, 0, len(xcmdEnvs))
				for k, v := range xcmdEnvs {
					var vs string
					err = v.As(&vs)
					if err != nil {
						// invalid attribute type - this shouldn't happen, bail out for now
						response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
							Severity: tfprotov5.DiagnosticSeverityError,
							Summary:  "Provider configuration: failed to assert type of element in 'env' value",
							Detail:   err.Error(),
						})
						return response, nil
					}
					execCfg.Env = append(execCfg.Env, clientcmdapi.ExecEnvVar{
						Name:  k,
						Value: vs,
					})
				}
			}
			overrides.AuthInfo.Exec = &execCfg
		}
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	clientConfig, err := cc.ClientConfig()
	if err != nil {
		s.logger.Error("[Configure]", "Failed to load config:", dump(cc))
		if errors.Is(err, clientcmd.ErrEmptyConfig) {
			// this is a terrible fix for if the configuration is a calculated value
			return response, nil
		}
		response.Diagnostics = append(response.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Provider configuration: cannot load Kubernetes client config",
			Detail:   err.Error(),
		})
		return response, nil
	}

	if s.logger.IsTrace() {
		clientConfig.WrapTransport = loggingTransport
	}

	codec := runtime.NoopEncoder{Decoder: scheme.Codecs.UniversalDecoder()}
	clientConfig.NegotiatedSerializer = serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{Serializer: codec})

	s.logger.Trace("[Configure]", "[ClientConfig]", dump(*clientConfig))
	s.clientConfig = clientConfig

	return response, nil
}

func (s *RawProviderServer) canExecute() (resp []*tfprotov5.Diagnostic) {
	if !s.providerEnabled {
		resp = append(resp, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Experimental feature not enabled.",
			Detail:   "The `kubernetes_manifest` resource is an experimental feature and must be explicitly enabled in the provider configuration block.",
		})
	}
	if semver.IsValid(s.hostTFVersion) && semver.Compare(s.hostTFVersion, minTFVersion) < 0 {
		resp = append(resp, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Incompatible terraform version",
			Detail:   fmt.Sprintf("The `kubernetes_manifest` resource requires Terraform %s or above", minTFVersion),
		})
	}
	return
}
