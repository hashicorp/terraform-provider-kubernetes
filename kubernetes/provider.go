// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	gversion "github.com/hashicorp/go-version"
	"github.com/mitchellh/go-homedir"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"

	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

const defaultFieldManagerName = "Terraform"

func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_HOST", ""),
				Description: "The hostname (in form of URI) of Kubernetes master.",
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_USER", ""),
				Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_PASSWORD", ""),
				Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_INSECURE", false),
				Description: "Whether server should be accessed without verifying the TLS certificate.",
			},
			"tls_server_name": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_TLS_SERVER_NAME", ""),
				Description: "Server name passed to the server for SNI and is used in the client to check server certificates against.",
			},
			"client_certificate": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CLIENT_CERT_DATA", ""),
				Description: "PEM-encoded client certificate for TLS authentication.",
			},
			"client_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CLIENT_KEY_DATA", ""),
				Description: "PEM-encoded client certificate key for TLS authentication.",
			},
			"cluster_ca_certificate": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CLUSTER_CA_CERT_DATA", ""),
				Description: "PEM-encoded root certificates bundle for TLS authentication.",
			},
			"config_paths": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
			},
			"config_path": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("KUBE_CONFIG_PATH", nil),
				Description:   "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
				ConflictsWith: []string{"config_paths"},
			},
			"config_context": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX", ""),
			},
			"config_context_auth_info": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX_AUTH_INFO", ""),
				Description: "",
			},
			"config_context_cluster": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX_CLUSTER", ""),
				Description: "",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_TOKEN", ""),
				Description: "Token to authenticate an service account",
			},
			"proxy_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "URL to the proxy to be used for all API requests",
				DefaultFunc: schema.EnvDefaultFunc("KUBE_PROXY_URL", ""),
			},
			"exec": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_version": {
							Type:     schema.TypeString,
							Required: true,
							ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
								apiVersion := val.(string)
								if apiVersion == "client.authentication.k8s.io/v1alpha1" {
									return diag.Diagnostics{{
										Severity: diag.Warning,
										Summary:  "v1alpha1 of the client authentication API is deprecated, use v1beta1 or above",
										Detail:   "v1alpha1 of the client authentication API will be removed in Kubernetes client versions 1.24 and above. You may need to update your exec plugin to use the latest version.",
									}}
								}
								return nil
							},
						},
						"command": {
							Type:     schema.TypeString,
							Required: true,
						},
						"env": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"args": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
				Description: "",
			},
			"experiments": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Enable and disable experimental features.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"manifest_resource": {
							Type:     schema.TypeBool,
							Optional: true,
							DefaultFunc: func() (interface{}, error) {
								if v := os.Getenv("TF_X_KUBERNETES_MANIFEST_RESOURCE"); v != "" {
									vv, err := strconv.ParseBool(v)
									if err != nil {
										return true, err
									}
									return vv, nil
								}
								return true, nil
							},
							Description: "Enable the `kubernetes_manifest` resource.",
						},
					},
				},
			},
			"ignore_annotations": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsValidRegExp,
				},
				Optional:    true,
				Description: "List of Kubernetes metadata annotations to ignore across all resources handled by this provider for situations where external systems are managing certain resource annotations. Each item is a regular expression.",
			},
			"ignore_labels": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsValidRegExp,
				},
				Optional:    true,
				Description: "List of Kubernetes metadata labels to ignore across all resources handled by this provider for situations where external systems are managing certain resource labels. Each item is a regular expression.",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			// core
			"kubernetes_config_map":                 dataSourceKubernetesConfigMapV1(),
			"kubernetes_config_map_v1":              dataSourceKubernetesConfigMapV1(),
			"kubernetes_namespace":                  dataSourceKubernetesNamespaceV1(),
			"kubernetes_namespace_v1":               dataSourceKubernetesNamespaceV1(),
			"kubernetes_all_namespaces":             dataSourceKubernetesAllNamespaces(),
			"kubernetes_secret":                     dataSourceKubernetesSecretV1(),
			"kubernetes_secret_v1":                  dataSourceKubernetesSecretV1(),
			"kubernetes_endpoints_v1":               dataSourceKubernetesEndpointsV1(),
			"kubernetes_service":                    dataSourceKubernetesServiceV1(),
			"kubernetes_service_v1":                 dataSourceKubernetesServiceV1(),
			"kubernetes_pod":                        dataSourceKubernetesPodV1(),
			"kubernetes_pod_v1":                     dataSourceKubernetesPodV1(),
			"kubernetes_service_account":            dataSourceKubernetesServiceAccountV1(),
			"kubernetes_service_account_v1":         dataSourceKubernetesServiceAccountV1(),
			"kubernetes_persistent_volume_v1":       dataSourceKubernetesPersistentVolumeV1(),
			"kubernetes_persistent_volume_claim":    dataSourceKubernetesPersistentVolumeClaimV1(),
			"kubernetes_persistent_volume_claim_v1": dataSourceKubernetesPersistentVolumeClaimV1(),
			"kubernetes_nodes":                      dataSourceKubernetesNodes(),

			// networking
			"kubernetes_ingress":    dataSourceKubernetesIngress(),
			"kubernetes_ingress_v1": dataSourceKubernetesIngressV1(),

			// storage
			"kubernetes_storage_class":    dataSourceKubernetesStorageClassV1(),
			"kubernetes_storage_class_v1": dataSourceKubernetesStorageClassV1(),

			// admission control
			"kubernetes_mutating_webhook_configuration_v1": dataSourceKubernetesMutatingWebhookConfigurationV1(),
		},

		ResourcesMap: map[string]*schema.Resource{
			// core
			"kubernetes_namespace":                  resourceKubernetesNamespaceV1(),
			"kubernetes_namespace_v1":               resourceKubernetesNamespaceV1(),
			"kubernetes_service":                    resourceKubernetesServiceV1(),
			"kubernetes_service_v1":                 resourceKubernetesServiceV1(),
			"kubernetes_service_account":            resourceKubernetesServiceAccountV1(),
			"kubernetes_service_account_v1":         resourceKubernetesServiceAccountV1(),
			"kubernetes_default_service_account":    resourceKubernetesDefaultServiceAccountV1(),
			"kubernetes_default_service_account_v1": resourceKubernetesDefaultServiceAccountV1(),
			"kubernetes_config_map":                 resourceKubernetesConfigMapV1(),
			"kubernetes_config_map_v1":              resourceKubernetesConfigMapV1(),
			"kubernetes_config_map_v1_data":         resourceKubernetesConfigMapV1Data(),
			"kubernetes_secret":                     resourceKubernetesSecretV1(),
			"kubernetes_secret_v1":                  resourceKubernetesSecretV1(),
			"kubernetes_pod":                        resourceKubernetesPodV1(),
			"kubernetes_pod_v1":                     resourceKubernetesPodV1(),
			"kubernetes_endpoints":                  resourceKubernetesEndpointsV1(),
			"kubernetes_endpoints_v1":               resourceKubernetesEndpointsV1(),
			"kubernetes_endpoint_slice_v1":          resourceKubernetesEndpointSliceV1(),
			"kubernetes_env":                        resourceKubernetesEnv(),
			"kubernetes_limit_range":                resourceKubernetesLimitRangeV1(),
			"kubernetes_limit_range_v1":             resourceKubernetesLimitRangeV1(),
			"kubernetes_node_taint":                 resourceKubernetesNodeTaint(),
			"kubernetes_persistent_volume":          resourceKubernetesPersistentVolumeV1(),
			"kubernetes_persistent_volume_v1":       resourceKubernetesPersistentVolumeV1(),
			"kubernetes_persistent_volume_claim":    resourceKubernetesPersistentVolumeClaimV1(),
			"kubernetes_persistent_volume_claim_v1": resourceKubernetesPersistentVolumeClaimV1(),
			"kubernetes_replication_controller":     resourceKubernetesReplicationControllerV1(),
			"kubernetes_replication_controller_v1":  resourceKubernetesReplicationControllerV1(),
			"kubernetes_resource_quota":             resourceKubernetesResourceQuotaV1(),
			"kubernetes_resource_quota_v1":          resourceKubernetesResourceQuotaV1(),

			// api registration
			"kubernetes_api_service":    resourceKubernetesAPIServiceV1(),
			"kubernetes_api_service_v1": resourceKubernetesAPIServiceV1(),

			// apps
			"kubernetes_deployment":      resourceKubernetesDeploymentV1(),
			"kubernetes_deployment_v1":   resourceKubernetesDeploymentV1(),
			"kubernetes_daemonset":       resourceKubernetesDaemonSetV1(),
			"kubernetes_daemon_set_v1":   resourceKubernetesDaemonSetV1(),
			"kubernetes_stateful_set":    resourceKubernetesStatefulSetV1(),
			"kubernetes_stateful_set_v1": resourceKubernetesStatefulSetV1(),

			// batch
			"kubernetes_job":         resourceKubernetesJobV1(),
			"kubernetes_job_v1":      resourceKubernetesJobV1(),
			"kubernetes_cron_job":    resourceKubernetesCronJobV1Beta1(),
			"kubernetes_cron_job_v1": resourceKubernetesCronJobV1(),

			// autoscaling
			"kubernetes_horizontal_pod_autoscaler":         resourceKubernetesHorizontalPodAutoscaler(),
			"kubernetes_horizontal_pod_autoscaler_v1":      resourceKubernetesHorizontalPodAutoscalerV1(),
			"kubernetes_horizontal_pod_autoscaler_v2beta2": resourceKubernetesHorizontalPodAutoscalerV2Beta2(),
			"kubernetes_horizontal_pod_autoscaler_v2":      resourceKubernetesHorizontalPodAutoscalerV2(),

			// certificates
			"kubernetes_certificate_signing_request":    resourceKubernetesCertificateSigningRequest(),
			"kubernetes_certificate_signing_request_v1": resourceKubernetesCertificateSigningRequestV1(),

			// rbac
			"kubernetes_role":                    resourceKubernetesRoleV1(),
			"kubernetes_role_v1":                 resourceKubernetesRoleV1(),
			"kubernetes_role_binding":            resourceKubernetesRoleBindingV1(),
			"kubernetes_role_binding_v1":         resourceKubernetesRoleBindingV1(),
			"kubernetes_cluster_role":            resourceKubernetesClusterRoleV1(),
			"kubernetes_cluster_role_v1":         resourceKubernetesClusterRoleV1(),
			"kubernetes_cluster_role_binding":    resourceKubernetesClusterRoleBindingV1(),
			"kubernetes_cluster_role_binding_v1": resourceKubernetesClusterRoleBindingV1(),

			// networking
			"kubernetes_ingress":           resourceKubernetesIngressV1Beta1(),
			"kubernetes_ingress_v1":        resourceKubernetesIngressV1(),
			"kubernetes_ingress_class":     resourceKubernetesIngressClassV1(),
			"kubernetes_ingress_class_v1":  resourceKubernetesIngressClassV1(),
			"kubernetes_network_policy":    resourceKubernetesNetworkPolicyV1(),
			"kubernetes_network_policy_v1": resourceKubernetesNetworkPolicyV1(),

			// policy
			"kubernetes_pod_disruption_budget":       resourceKubernetesPodDisruptionBudget(),
			"kubernetes_pod_disruption_budget_v1":    resourceKubernetesPodDisruptionBudgetV1(),
			"kubernetes_pod_security_policy":         resourceKubernetesPodSecurityPolicyV1Beta1(),
			"kubernetes_pod_security_policy_v1beta1": resourceKubernetesPodSecurityPolicyV1Beta1(),

			// scheduling
			"kubernetes_priority_class":    resourceKubernetesPriorityClassV1(),
			"kubernetes_priority_class_v1": resourceKubernetesPriorityClassV1(),

			// admission control
			"kubernetes_validating_webhook_configuration":    resourceKubernetesValidatingWebhookConfigurationV1Beta1(),
			"kubernetes_validating_webhook_configuration_v1": resourceKubernetesValidatingWebhookConfigurationV1(),
			"kubernetes_mutating_webhook_configuration":      resourceKubernetesMutatingWebhookConfiguration(),
			"kubernetes_mutating_webhook_configuration_v1":   resourceKubernetesMutatingWebhookConfigurationV1(),

			// storage
			"kubernetes_storage_class":    resourceKubernetesStorageClassV1(),
			"kubernetes_storage_class_v1": resourceKubernetesStorageClassV1(),
			"kubernetes_csi_driver":       resourceKubernetesCSIDriverV1Beta1(),
			"kubernetes_csi_driver_v1":    resourceKubernetesCSIDriverV1(),

			// provider helper resources
			"kubernetes_labels":      resourceKubernetesLabels(),
			"kubernetes_annotations": resourceKubernetesAnnotations(),

			// authentication
			"kubernetes_token_request_v1": resourceKubernetesTokenRequestV1(),

			//node
			"kubernetes_runtime_class_v1": resourceKubernetesRuntimeClassV1(),
		},
	}

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, d, p.TerraformVersion)
	}

	return p
}

type KubeClientsets interface {
	MainClientset() (*kubernetes.Clientset, error)
	AggregatorClientset() (*aggregator.Clientset, error)
	DynamicClient() (dynamic.Interface, error)
	DiscoveryClient() (discovery.DiscoveryInterface, error)
}

type kubeClientsets struct {
	// TODO: this struct has become overloaded we should
	// rename this or break it into smaller structs
	config              *restclient.Config
	mainClientset       *kubernetes.Clientset
	aggregatorClientset *aggregator.Clientset
	dynamicClient       dynamic.Interface
	discoveryClient     discovery.DiscoveryInterface

	IgnoreAnnotations []string
	IgnoreLabels      []string
}

func (k kubeClientsets) MainClientset() (*kubernetes.Clientset, error) {
	if k.mainClientset != nil {
		return k.mainClientset, nil
	}

	if k.config != nil {
		kc, err := kubernetes.NewForConfig(k.config)
		if err != nil {
			return nil, fmt.Errorf("Failed to configure client: %s", err)
		}
		k.mainClientset = kc
	}
	return k.mainClientset, nil
}

func (k kubeClientsets) AggregatorClientset() (*aggregator.Clientset, error) {
	if k.aggregatorClientset != nil {
		return k.aggregatorClientset, nil
	}
	if k.config != nil {
		ac, err := aggregator.NewForConfig(k.config)
		if err != nil {
			return nil, fmt.Errorf("Failed to configure client: %s", err)
		}
		k.aggregatorClientset = ac
	}
	return k.aggregatorClientset, nil
}

func (k kubeClientsets) DynamicClient() (dynamic.Interface, error) {
	if k.dynamicClient != nil {
		return k.dynamicClient, nil
	}

	if k.config != nil {
		kc, err := dynamic.NewForConfig(k.config)
		if err != nil {
			return nil, fmt.Errorf("Failed to configure dynamic client: %s", err)
		}
		k.dynamicClient = kc
	}
	return k.dynamicClient, nil
}

func (k kubeClientsets) DiscoveryClient() (discovery.DiscoveryInterface, error) {
	if k.discoveryClient != nil {
		return k.discoveryClient, nil
	}

	if k.config != nil {
		kc, err := discovery.NewDiscoveryClientForConfig(k.config)
		if err != nil {
			return nil, fmt.Errorf("Failed to configure discovery client: %s", err)
		}
		k.discoveryClient = kc
	}
	return k.discoveryClient, nil
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	// Config initialization
	cfg, err := initializeConfiguration(d)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	if cfg == nil {
		// This is a TEMPORARY measure to work around https://github.com/hashicorp/terraform/issues/24055
		// IMPORTANT: this will NOT enable a workaround of issue: https://github.com/hashicorp/terraform/issues/4149
		// IMPORTANT: if the supplied configuration is incomplete or invalid
		///IMPORTANT: provider operations will fail or attempt to connect to localhost endpoints
		cfg = &restclient.Config{}
	}

	cfg.UserAgent = fmt.Sprintf("HashiCorp/1.0 Terraform/%s", terraformVersion)

	if logging.IsDebugOrHigher() {
		log.Printf("[DEBUG] Enabling HTTP requests/responses tracing")
		cfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return logging.NewSubsystemLoggingHTTPTransport("Kubernetes", rt)
		}
	}

	ignoreAnnotations := []string{}
	ignoreLabels := []string{}

	if v, ok := d.Get("ignore_annotations").([]interface{}); ok {
		ignoreAnnotations = expandStringSlice(v)
	}
	if v, ok := d.Get("ignore_labels").([]interface{}); ok {
		ignoreLabels = expandStringSlice(v)
	}

	m := kubeClientsets{
		config:              cfg,
		mainClientset:       nil,
		aggregatorClientset: nil,
		IgnoreAnnotations:   ignoreAnnotations,
		IgnoreLabels:        ignoreLabels,
	}
	return m, diag.Diagnostics{}
}

func initializeConfiguration(d *schema.ResourceData) (*restclient.Config, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}

	if v, ok := d.Get("config_path").(string); ok && v != "" {
		configPaths = []string{v}
	} else if v, ok := d.Get("config_paths").([]interface{}); ok && len(v) > 0 {
		for _, p := range v {
			configPaths = append(configPaths, p.(string))
		}
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		// NOTE we have to do this here because the schema
		// does not yet allow you to set a default for a TypeList
		configPaths = filepath.SplitList(v)
	}

	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}

			log.Printf("[DEBUG] Using kubeconfig: %s", path)
			expandedPaths = append(expandedPaths, path)
		}

		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		ctxSuffix := "; default context"

		kubectx, ctxOk := d.GetOk("config_context")
		authInfo, authInfoOk := d.GetOk("config_context_auth_info")
		cluster, clusterOk := d.GetOk("config_context_cluster")
		if ctxOk || authInfoOk || clusterOk {
			ctxSuffix = "; overridden context"
			if ctxOk {
				overrides.CurrentContext = kubectx.(string)
				ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
				log.Printf("[DEBUG] Using custom current context: %q", overrides.CurrentContext)
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfoOk {
				overrides.Context.AuthInfo = authInfo.(string)
				ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
			}
			if clusterOk {
				overrides.Context.Cluster = cluster.(string)
				ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
			}
			log.Printf("[DEBUG] Using overridden context: %#v", overrides.Context)
		}
	}

	// Overriding with static configuration
	if v, ok := d.GetOk("insecure"); ok {
		overrides.ClusterInfo.InsecureSkipTLSVerify = v.(bool)
	}
	if v, ok := d.GetOk("tls_server_name"); ok {
		overrides.ClusterInfo.TLSServerName = v.(string)
	}
	if v, ok := d.GetOk("cluster_ca_certificate"); ok {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("client_certificate"); ok {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("host"); ok {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := restclient.DefaultServerURL(v.(string), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse host: %s", err)
		}

		overrides.ClusterInfo.Server = host.String()
	}
	if v, ok := d.GetOk("username"); ok {
		overrides.AuthInfo.Username = v.(string)
	}
	if v, ok := d.GetOk("password"); ok {
		overrides.AuthInfo.Password = v.(string)
	}
	if v, ok := d.GetOk("client_key"); ok {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("token"); ok {
		overrides.AuthInfo.Token = v.(string)
	}

	if v, ok := d.GetOk("exec"); ok {
		exec := &clientcmdapi.ExecConfig{}
		if spec, ok := v.([]interface{})[0].(map[string]interface{}); ok {
			exec.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
			exec.APIVersion = spec["api_version"].(string)
			exec.Command = spec["command"].(string)
			exec.Args = expandStringSlice(spec["args"].([]interface{}))
			for kk, vv := range spec["env"].(map[string]interface{}) {
				exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: kk, Value: vv.(string)})
			}
		} else {
			return nil, fmt.Errorf("Failed to parse exec")
		}
		overrides.AuthInfo.Exec = exec
	}

	if v, ok := d.GetOk("proxy_url"); ok {
		overrides.ClusterDefaults.ProxyURL = v.(string)
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	cfg, err := cc.ClientConfig()
	if err != nil {
		log.Printf("[WARN] Invalid provider configuration was supplied. Provider operations likely to fail: %v", err)
		return nil, nil
	}

	return cfg, nil
}

var useadmissionregistrationv1beta1 *bool

func useAdmissionregistrationV1beta1(conn *kubernetes.Clientset) (bool, error) {
	if useadmissionregistrationv1beta1 != nil {
		return *useadmissionregistrationv1beta1, nil
	}

	d := conn.Discovery()

	group := "admissionregistration.k8s.io"

	v1, err := apimachineryschema.ParseGroupVersion(fmt.Sprintf("%s/v1", group))
	if err != nil {
		return false, err
	}

	err = discovery.ServerSupportsVersion(d, v1)
	if err == nil {
		log.Printf("[INFO] Using %s/v1", group)
		useadmissionregistrationv1beta1 = ptr.To(false)
		return false, nil
	}

	v1beta1, err := apimachineryschema.ParseGroupVersion(fmt.Sprintf("%s/v1beta1", group))
	if err != nil {
		return false, err
	}

	err = discovery.ServerSupportsVersion(d, v1beta1)
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Using %s/v1beta1", group)
	useadmissionregistrationv1beta1 = ptr.To(true)
	return true, nil
}

func getServerVersion(connection *kubernetes.Clientset) (*gversion.Version, error) {
	sv, err := connection.ServerVersion()
	if err != nil {
		return nil, err
	}

	return gversion.NewVersion(sv.String())
}

func serverVersionGreaterThanOrEqual(connection *kubernetes.Clientset, version string) (bool, error) {
	sv, err := getServerVersion(connection)
	if err != nil {
		return false, err
	}
	// server version that we need to compare with
	cv, err := gversion.NewVersion(version)
	if err != nil {
		return false, err
	}

	return sv.GreaterThanOrEqual(cv), nil
}
