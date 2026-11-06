// Copyright IBM Corp. 2017, 2026
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

	"github.com/hashicorp/go-hclog"
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

type credentialFingerprint struct {
	ConfigPath            string
	ConfigPaths           []string
	ClientCertificate     string
	ClusterCACertificate  string
	Insecure              bool
	TLSServerName         string
	Host                  string
	ClientKey             string
	ConfigContext         string
	ConfigContextCluster  string
	ConfigContextAuthInfo string
	Username              string
	Password              string
	Token                 string
	ProxyURL              string
	ExecAPIVersion        string
	ExecCommand           string
	ExecArgs              []string
	ExecEnv               map[string]string
}

type envLookup func(string) (string, bool)

func osEnvLookup(name string) (string, bool) { return os.LookupEnv(name) }

func noEnvLookup(string) (string, bool) { return "", false }

func buildClientConfig(providerConfig map[string]tftypes.Value, logger hclog.Logger, lookupEnv envLookup) (*rest.Config, credentialFingerprint, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	var fp credentialFingerprint
	var err error

	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	// Handle 'config_path' attribute
	//
	var configPath string
	if !providerConfig["config_path"].IsNull() && providerConfig["config_path"].IsKnown() {
		err = providerConfig["config_path"].As(&configPath)
		if err != nil {
			// invalid attribute - this shouldn't happen, bail out now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to extract 'config_path' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
	} else if configPathEnv, ok := lookupEnv("KUBE_CONFIG_PATH"); ok && configPathEnv != "" {
		configPath = configPathEnv
	}
	if len(configPath) > 0 {
		configPathAbs, err := homedir.Expand(configPath)
		if err == nil {
			_, err = os.Stat(configPathAbs)
		}
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   fmt.Sprintf("'config_path' refers to an invalid path: %q: %v", configPathAbs, err),
			})
		}
		loader.ExplicitPath = configPathAbs
		fp.ConfigPath = configPathAbs
	}
	// Handle 'config_paths' attribute
	//
	var precedence []string
	if !providerConfig["config_paths"].IsNull() && providerConfig["config_paths"].IsKnown() {
		var configPaths []tftypes.Value
		err = providerConfig["config_paths"].As(&configPaths)
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to extract 'config_paths' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		for _, p := range configPaths {
			var pp string
			p.As(&pp)
			precedence = append(precedence, pp)
		}
	} else if configPathsEnv, ok := lookupEnv("KUBE_CONFIG_PATHS"); ok && configPathsEnv != "" {
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
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Invalid attribute in provider configuration",
					Detail:   fmt.Sprintf("'config_paths' refers to an invalid path: %q: %v", absPath, err),
				})
			}
			precedence[i] = absPath
		}
		loader.Precedence = precedence
		fp.ConfigPaths = precedence
	}

	// Handle 'client_certificate' attribute
	//
	var clientCertificate string
	if !providerConfig["client_certificate"].IsNull() && providerConfig["client_certificate"].IsKnown() {
		err = providerConfig["client_certificate"].As(&clientCertificate)
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'client_certificate' type cannot be asserted: " + err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
	} else if clientCrtEnv, ok := lookupEnv("KUBE_CLIENT_CERT_DATA"); ok && clientCrtEnv != "" {
		clientCertificate = clientCrtEnv
	}
	if len(clientCertificate) > 0 {
		ccPEM, _ := pem.Decode([]byte(clientCertificate))
		if ccPEM == nil || ccPEM.Type != "CERTIFICATE" {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'client_certificate' is not a valid PEM encoded certificate",
			})
		}
		overrides.AuthInfo.ClientCertificateData = []byte(clientCertificate)
		fp.ClientCertificate = clientCertificate
	}

	// Handle 'cluster_ca_certificate' attribute
	//
	var clusterCaCertificate string
	if !providerConfig["cluster_ca_certificate"].IsNull() && providerConfig["cluster_ca_certificate"].IsKnown() {
		err = providerConfig["cluster_ca_certificate"].As(&clusterCaCertificate)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to extract 'cluster_ca_certificate' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
	} else if clusterCAEnv, ok := lookupEnv("KUBE_CLUSTER_CA_CERT_DATA"); ok && clusterCAEnv != "" {
		clusterCaCertificate = clusterCAEnv
	}
	if len(clusterCaCertificate) > 0 {
		ca, _ := pem.Decode([]byte(clusterCaCertificate))
		if ca == nil || ca.Type != "CERTIFICATE" {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'cluster_ca_certificate' is not a valid PEM encoded certificate",
			})
		}
		overrides.ClusterInfo.CertificateAuthorityData = []byte(clusterCaCertificate)
		fp.ClusterCACertificate = clusterCaCertificate
	}

	// Handle 'insecure' attribute
	//
	var insecure bool
	if !providerConfig["insecure"].IsNull() && providerConfig["insecure"].IsKnown() {
		err = providerConfig["insecure"].As(&insecure)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'insecure' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
	} else if insecureEnv, ok := lookupEnv("KUBE_INSECURE"); ok && insecureEnv != "" {
		iv, err := strconv.ParseBool(insecureEnv)
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid provider configuration",
				Detail:   "Environment variable KUBE_INSECURE contains invalid value: " + err.Error(),
			})
		} else {
			insecure = iv
		}
	}
	overrides.ClusterInfo.InsecureSkipTLSVerify = insecure
	fp.Insecure = insecure

	// Handle 'tls_server_name' attribute
	//
	var tlsServerName string
	if !providerConfig["tls_server_name"].IsNull() && providerConfig["tls_server_name"].IsKnown() {
		err = providerConfig["tls_server_name"].As(&tlsServerName)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'tls_server_name' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		overrides.ClusterInfo.TLSServerName = tlsServerName
	} else if tlsServerName, ok := lookupEnv("KUBE_TLS_SERVER_NAME"); ok && tlsServerName != "" {
		overrides.ClusterInfo.TLSServerName = tlsServerName
	}
	fp.TLSServerName = overrides.ClusterInfo.TLSServerName

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
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to extract 'host' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
	} else if hostEnv, ok := lookupEnv("KUBE_HOST"); ok && hostEnv != "" {
		host = hostEnv
	}
	if len(host) > 0 {
		_, err = url.ParseRequestURI(host)
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'host' is not a valid URL",
			})
		}
		hostURL, _, err := rest.DefaultServerURL(host, "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "Invalid value for 'host': " + err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		overrides.ClusterInfo.Server = hostURL.String()
	}
	fp.Host = overrides.ClusterInfo.Server

	// Handle 'client_key' attribute
	//
	var clientKey string
	if !providerConfig["client_key"].IsNull() && providerConfig["client_key"].IsKnown() {
		err = providerConfig["client_key"].As(&clientKey)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: ",
				Detail:   "Failed to extract 'client_key' value" + err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
	} else if clientKeyEnv, ok := lookupEnv("KUBE_CLIENT_KEY_DATA"); ok && clientKeyEnv != "" {
		clientKey = clientKeyEnv
	}
	if len(clientKey) > 0 {
		ck, _ := pem.Decode([]byte(clientKey))
		if ck == nil || !strings.Contains(ck.Type, "PRIVATE KEY") {
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'client_key' is not a valid PEM encoded private key",
			})
		}
		overrides.AuthInfo.ClientKeyData = []byte(clientKey)
		fp.ClientKey = clientKey
	}

	if len(diags) > 0 {
		return nil, credentialFingerprint{}, diags
	}

	// Handle 'config_context' attribute
	//
	var cfgContext string
	if !providerConfig["config_context"].IsNull() && providerConfig["config_context"].IsKnown() {
		err = providerConfig["config_context"].As(&cfgContext)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'config_context' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		overrides.CurrentContext = cfgContext
	} else if cfgContext, ok := lookupEnv("KUBE_CTX"); ok && cfgContext != "" {
		overrides.CurrentContext = cfgContext
	}
	fp.ConfigContext = overrides.CurrentContext

	overrides.Context = clientcmdapi.Context{}

	// Handle 'config_context_cluster' attribute
	//
	var cfgCtxCluster string
	if !providerConfig["config_context_cluster"].IsNull() && providerConfig["config_context_cluster"].IsKnown() {
		err = providerConfig["config_context_cluster"].As(&cfgCtxCluster)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'config_context_cluster' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		overrides.Context.Cluster = cfgCtxCluster
	} else if cfgCtxCluster, ok := lookupEnv("KUBE_CTX_CLUSTER"); ok && cfgCtxCluster != "" {
		overrides.Context.Cluster = cfgCtxCluster
	}
	fp.ConfigContextCluster = overrides.Context.Cluster

	// Handle 'config_context_auth_info' attribute
	//
	var cfgContextAuthInfo *string
	if !providerConfig["config_context_auth_info"].IsNull() && providerConfig["config_context_auth_info"].IsKnown() {
		err = providerConfig["config_context_auth_info"].As(&cfgContextAuthInfo)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'config_context_auth_info' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		if cfgContextAuthInfo != nil {
			overrides.Context.AuthInfo = *cfgContextAuthInfo
		}
	} else if cfgContextAuthInfoEnv, ok := lookupEnv("KUBE_CTX_AUTH_INFO"); ok && cfgContextAuthInfoEnv != "" {
		overrides.Context.AuthInfo = cfgContextAuthInfoEnv
	}
	fp.ConfigContextAuthInfo = overrides.Context.AuthInfo

	var username string
	if !providerConfig["username"].IsNull() && providerConfig["username"].IsKnown() {
		err = providerConfig["username"].As(&username)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'username' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		overrides.AuthInfo.Username = username
	} else if username, ok := lookupEnv("KUBE_USERNAME"); ok && username != "" {
		overrides.AuthInfo.Username = username
	}
	fp.Username = overrides.AuthInfo.Username

	var password string
	if !providerConfig["password"].IsNull() && providerConfig["password"].IsKnown() {
		err = providerConfig["password"].As(&password)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'password' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		overrides.AuthInfo.Password = password
	} else if password, ok := lookupEnv("KUBE_PASSWORD"); ok && password != "" {
		overrides.AuthInfo.Password = password
	}
	fp.Password = overrides.AuthInfo.Password

	var token string
	if !providerConfig["token"].IsNull() && providerConfig["token"].IsKnown() {
		err = providerConfig["token"].As(&token)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'token' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		overrides.AuthInfo.Token = token
	} else if token, ok := lookupEnv("KUBE_TOKEN"); ok && token != "" {
		overrides.AuthInfo.Token = token
	}
	fp.Token = overrides.AuthInfo.Token

	var proxyURL string
	if !providerConfig["proxy_url"].IsNull() && providerConfig["proxy_url"].IsKnown() {
		err = providerConfig["proxy_url"].As(&proxyURL)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'proxy_url' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		overrides.ClusterDefaults.ProxyURL = proxyURL
	} else if proxyURL, ok := lookupEnv("KUBE_PROXY_URL"); ok && proxyURL != "" {
		overrides.ClusterDefaults.ProxyURL = proxyURL
	}
	fp.ProxyURL = overrides.ClusterDefaults.ProxyURL

	if !providerConfig["exec"].IsNull() && providerConfig["exec"].IsKnown() {
		var execBlock []tftypes.Value
		err = providerConfig["exec"].As(&execBlock)
		if err != nil {
			// invalid attribute type - this shouldn't happen, bail out for now
			diags = append(diags, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Provider configuration: failed to assert type of 'exec' value",
				Detail:   err.Error(),
			})
			return nil, credentialFingerprint{}, diags
		}
		execCfg := clientcmdapi.ExecConfig{}
		execCfg.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
		if len(execBlock) > 0 {
			var execObj map[string]tftypes.Value
			err := execBlock[0].As(&execObj)
			if err != nil {
				diags = append(diags, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  `Provider configuration: failed to assert type of "exec" block`,
					Detail:   err.Error(),
				})
				return nil, credentialFingerprint{}, diags
			}
			if !execObj["api_version"].IsNull() && execObj["api_version"].IsKnown() {
				var apiv string
				err = execObj["api_version"].As(&apiv)
				if err != nil {
					// invalid attribute type - this shouldn't happen, bail out for now
					diags = append(diags, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Provider configuration: failed to assert type of 'api_version' value",
						Detail:   err.Error(),
					})
					return nil, credentialFingerprint{}, diags
				}
				execCfg.APIVersion = apiv
			}
			if !execObj["command"].IsNull() && execObj["command"].IsKnown() {
				var cmd string
				err = execObj["command"].As(&cmd)
				if err != nil {
					// invalid attribute type - this shouldn't happen, bail out for now
					diags = append(diags, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Provider configuration: failed to assert type of 'command' value",
						Detail:   err.Error(),
					})
					return nil, credentialFingerprint{}, diags
				}
				execCfg.Command = cmd
			}
			if !execObj["args"].IsNull() && execObj["args"].IsFullyKnown() {
				var xcmdArgs []tftypes.Value
				err = execObj["args"].As(&xcmdArgs)
				if err != nil {
					// invalid attribute type - this shouldn't happen, bail out for now
					diags = append(diags, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Provider configuration: failed to assert type of 'args' value",
						Detail:   err.Error(),
					})
					return nil, credentialFingerprint{}, diags
				}
				execCfg.Args = make([]string, 0, len(xcmdArgs))
				for _, arg := range xcmdArgs {
					var v string
					err := arg.As(&v)
					if err != nil {
						// invalid attribute type - this shouldn't happen, bail out for now
						diags = append(diags, &tfprotov5.Diagnostic{
							Severity: tfprotov5.DiagnosticSeverityError,
							Summary:  "Provider configuration: failed to assert type of element in 'args' value",
							Detail:   err.Error(),
						})
						return nil, credentialFingerprint{}, diags
					}
					execCfg.Args = append(execCfg.Args, v)
				}
			}
			if !execObj["env"].IsNull() && execObj["env"].IsFullyKnown() {
				var xcmdEnvs map[string]tftypes.Value
				err = execObj["env"].As(&xcmdEnvs)
				if err != nil {
					// invalid attribute type - this shouldn't happen, bail out for now
					diags = append(diags, &tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Provider configuration: failed to assert type of element in 'env' value",
						Detail:   err.Error(),
					})
					return nil, credentialFingerprint{}, diags
				}
				execCfg.Env = make([]clientcmdapi.ExecEnvVar, 0, len(xcmdEnvs))
				for k, v := range xcmdEnvs {
					var vs string
					err = v.As(&vs)
					if err != nil {
						// invalid attribute type - this shouldn't happen, bail out for now
						diags = append(diags, &tfprotov5.Diagnostic{
							Severity: tfprotov5.DiagnosticSeverityError,
							Summary:  "Provider configuration: failed to assert type of element in 'env' value",
							Detail:   err.Error(),
						})
						return nil, credentialFingerprint{}, diags
					}
					execCfg.Env = append(execCfg.Env, clientcmdapi.ExecEnvVar{
						Name:  k,
						Value: vs,
					})
				}
			}
			overrides.AuthInfo.Exec = &execCfg
			fp.ExecAPIVersion = execCfg.APIVersion
			fp.ExecCommand = execCfg.Command
			fp.ExecArgs = append([]string{}, execCfg.Args...)
			if len(execCfg.Env) > 0 {
				fp.ExecEnv = make(map[string]string, len(execCfg.Env))
				for _, e := range execCfg.Env {
					fp.ExecEnv[e.Name] = e.Value
				}
			}
		}
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	clientConfig, err := cc.ClientConfig()
	if err != nil {
		if logger != nil {
			logger.Error("[Configure]", "Failed to load config:", dump(cc))
		}
		if errors.Is(err, clientcmd.ErrEmptyConfig) {
			// this is a terrible fix for if the configuration is a calculated value
			return nil, credentialFingerprint{}, nil
		}
		diags = append(diags, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Provider configuration: cannot load Kubernetes client config",
			Detail:   err.Error(),
		})
		return nil, credentialFingerprint{}, diags
	}

	if logger != nil && logger.IsTrace() {
		clientConfig.WrapTransport = loggingTransport
	}

	codec := runtime.NoopEncoder{Decoder: scheme.Codecs.UniversalDecoder()}
	clientConfig.NegotiatedSerializer = serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{Serializer: codec})

	if logger != nil {
		logger.Trace("[Configure]", "[ClientConfig]", dump(*clientConfig))
	}

	return clientConfig, fp, diags
}

// ConfigureProvider function
func (s *RawProviderServer) ConfigureProvider(ctx context.Context, req *tfprotov5.ConfigureProviderRequest) (*tfprotov5.ConfigureProviderResponse, error) {
	response := &tfprotov5.ConfigureProviderResponse{}
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

	clcp := req.ClientCapabilities
	if !cfgVal.IsFullyKnown() && clcp != nil && clcp.DeferralAllowed {
		// need to deferr actions
		s.clientConfigUnknown = true
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

	clientConfig, _, diags := buildClientConfig(providerConfig, s.logger, osEnvLookup)
	if len(diags) > 0 {
		response.Diagnostics = diags
		return response, nil
	}
	if clientConfig == nil {
		return response, nil
	}

	s.clientConfig = clientConfig

	return response, nil
}

func (s *RawProviderServer) canExecute() (resp []*tfprotov5.Diagnostic) {
	if semver.IsValid(s.hostTFVersion) && semver.Compare(s.hostTFVersion, minTFVersion) < 0 {
		resp = append(resp, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Incompatible terraform version",
			Detail:   fmt.Sprintf("The `kubernetes_manifest` resource requires Terraform %s or above", minTFVersion),
		})
	}
	return
}
