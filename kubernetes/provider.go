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

	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

const defaultFieldManagerName = "Terraform"

func Provider() *schema.Provider {
	conditionsMessage := "Specifying more than one authentication method can lead to unpredictable behavior." +
		" This option will be removed in a future release. Please update your configuration."
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_HOST", nil),
				Description:       "The hostname (in form of URI) of Kubernetes master.",
				ConflictsWith:     []string{"config_path", "config_paths"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: conditionsMessage,
				// TODO: enable this when AtLeastOneOf works with optional attributes.
				// https://github.com/hashicorp/terraform-plugin-sdk/issues/705
				// AtLeastOneOf: []string{"token", "exec", "username", "password", "client_certificate", "client_key"},
			},
			"username": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_USER", nil),
				Description:       "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
				ConflictsWith:     []string{"config_path", "config_paths", "exec", "token", "client_certificate", "client_key"},
				RequiredWith:      []string{"password", "host"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: conditionsMessage,
			},
			"password": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_PASSWORD", nil),
				Description:       "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
				ConflictsWith:     []string{"config_path", "config_paths", "exec", "token", "client_certificate", "client_key"},
				RequiredWith:      []string{"username", "host"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: conditionsMessage,
			},
			"insecure": {
				Type:              schema.TypeBool,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_INSECURE", nil),
				Description:       "Whether server should be accessed without verifying the TLS certificate.",
				ConflictsWith:     []string{"cluster_ca_certificate", "client_key", "client_certificate", "exec"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: conditionsMessage,
			},
			"client_certificate": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_CLIENT_CERT_DATA", nil),
				Description:       "PEM-encoded client certificate for TLS authentication.",
				ConflictsWith:     []string{"config_path", "config_paths", "username", "password", "insecure"},
				RequiredWith:      []string{"client_key", "cluster_ca_certificate", "host"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: conditionsMessage,
			},
			"client_key": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_CLIENT_KEY_DATA", nil),
				Description:       "PEM-encoded client certificate key for TLS authentication.",
				ConflictsWith:     []string{"config_path", "config_paths", "username", "password", "exec", "insecure"},
				RequiredWith:      []string{"client_certificate", "cluster_ca_certificate", "host"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: conditionsMessage,
			},
			"cluster_ca_certificate": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_CLUSTER_CA_CERT_DATA", nil),
				Description:       "PEM-encoded root certificates bundle for TLS authentication.",
				ConflictsWith:     []string{"config_path", "config_paths", "insecure"},
				RequiredWith:      []string{"host"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: conditionsMessage,
				// TODO: enable this when AtLeastOneOf works with optional attributes.
				// https://github.com/hashicorp/terraform-plugin-sdk/issues/705
				// AtLeastOneOf:  []string{"token", "exec", "client_certificate", "client_key"},
			},
			"config_paths": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				DefaultFunc: configPathsEnv,
				Optional:    true,
				Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
				// config_paths conflicts with every attribute except for "insecure", since all of these options will be read from the kubeconfig.
				ConflictsWith:     []string{"config_path", "exec", "token", "host", "client_certificate", "client_key", "cluster_ca_certificate", "username", "password"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: conditionsMessage,
			},
			"config_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CONFIG_PATH", nil),
				Description: "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
				// config_path conflicts with every attribute except for "insecure", since all of these options will be read from the kubeconfig.
				ConflictsWith:     []string{"config_paths", "exec", "token", "host", "client_certificate", "client_key", "cluster_ca_certificate", "username", "password"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: conditionsMessage,
			},
			"config_context": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_CTX", nil),
				Description:       "Context to choose from the kube config file. ",
				ConflictsWith:     []string{"exec", "token", "client_certificate", "client_key", "username", "password"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: "This functionality will be removed in a later release. Please update your configuration.",
				// TODO: enable this when AtLeastOneOf works with optional attributes.
				// AtLeastOneOf:  []string{"config_path", "config_paths"},
			},
			"config_context_auth_info": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_CTX_AUTH_INFO", nil),
				Description:       "Authentication info context of the kube config (name of the kubeconfig user, --user flag in kubectl).",
				ConflictsWith:     []string{"exec", "token", "client_certificate", "client_key", "username", "password"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: "This functionality will be removed in a later release. Please update your configuration.",
				// TODO: enable this when AtLeastOneOf works with optional attributes.
				// AtLeastOneOf:  []string{"config_path", "config_paths"},
			},
			"config_context_cluster": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_CTX_CLUSTER", nil),
				Description:       "Cluster context of the kube config (name of the kubeconfig cluster, --cluster flag in kubectl).",
				ConflictsWith:     []string{"exec", "token", "client_certificate", "client_key", "username", "password"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: "Specifying more than one authentication method can lead to unpredictable behavior. This option will be removed in a future release. Please update your configuration.",
				// TODO: enable this when AtLeastOneOf works with optional attributes.
				// AtLeastOneOf:  []string{"config_path", "config_paths"},
			},
			"token": {
				Type:              schema.TypeString,
				Optional:          true,
				DefaultFunc:       schema.EnvDefaultFunc("KUBE_TOKEN", nil),
				Description:       "Bearer token for authenticating the Kubernetes API.",
				ConflictsWith:     []string{"config_path", "config_paths", "exec", "client_certificate", "client_key", "username", "password"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: "Specifying more than one authentication method can lead to unpredictable behavior. This option will be removed in a future release. Please update your configuration.",
				RequiredWith:      []string{"host"},
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
				Description:       "Configuration block to use an exec-based credential plugin, e.g. call an external command to receive user credentials.",
				ConflictsWith:     []string{"config_path", "config_paths", "token", "client_certificate", "client_key", "username", "password", "insecure"},
				RequiredWith:      []string{"host", "cluster_ca_certificate"},
				ConditionsMode:    schema.SchemaConditionsModeWarning,
				ConditionsMessage: "Specifying more than one authentication method can lead to unpredictable behavior. This option will be removed in a future release. Please update your configuration.",
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
			"kubernetes_config_map":                 dataSourceKubernetesConfigMap(),
			"kubernetes_config_map_v1":              dataSourceKubernetesConfigMap(),
			"kubernetes_namespace":                  dataSourceKubernetesNamespace(),
			"kubernetes_namespace_v1":               dataSourceKubernetesNamespace(),
			"kubernetes_all_namespaces":             dataSourceKubernetesAllNamespaces(),
			"kubernetes_secret":                     dataSourceKubernetesSecret(),
			"kubernetes_secret_v1":                  dataSourceKubernetesSecret(),
			"kubernetes_endpoints_v1":               dataSourceKubernetesEndpointsV1(),
			"kubernetes_service":                    dataSourceKubernetesService(),
			"kubernetes_service_v1":                 dataSourceKubernetesService(),
			"kubernetes_pod":                        dataSourceKubernetesPod(),
			"kubernetes_pod_v1":                     dataSourceKubernetesPod(),
			"kubernetes_service_account":            dataSourceKubernetesServiceAccount(),
			"kubernetes_service_account_v1":         dataSourceKubernetesServiceAccount(),
			"kubernetes_persistent_volume_claim":    dataSourceKubernetesPersistentVolumeClaim(),
			"kubernetes_persistent_volume_claim_v1": dataSourceKubernetesPersistentVolumeClaim(),
			"kubernetes_nodes":                      dataSourceKubernetesNodes(),

			// networking
			"kubernetes_ingress":    dataSourceKubernetesIngress(),
			"kubernetes_ingress_v1": dataSourceKubernetesIngressV1(),

			// storage
			"kubernetes_storage_class":    dataSourceKubernetesStorageClass(),
			"kubernetes_storage_class_v1": dataSourceKubernetesStorageClass(),

			// admission control
			"kubernetes_mutating_webhook_configuration_v1": dataSourceKubernetesMutatingWebhookConfiguration(),
		},

		ResourcesMap: map[string]*schema.Resource{
			// core
			"kubernetes_namespace":                  resourceKubernetesNamespace(),
			"kubernetes_namespace_v1":               resourceKubernetesNamespace(),
			"kubernetes_service":                    resourceKubernetesService(),
			"kubernetes_service_v1":                 resourceKubernetesService(),
			"kubernetes_service_account":            resourceKubernetesServiceAccount(),
			"kubernetes_service_account_v1":         resourceKubernetesServiceAccount(),
			"kubernetes_default_service_account":    resourceKubernetesDefaultServiceAccount(),
			"kubernetes_default_service_account_v1": resourceKubernetesDefaultServiceAccount(),
			"kubernetes_config_map":                 resourceKubernetesConfigMap(),
			"kubernetes_config_map_v1":              resourceKubernetesConfigMap(),
			"kubernetes_config_map_v1_data":         resourceKubernetesConfigMapV1Data(),
			"kubernetes_secret":                     resourceKubernetesSecret(),
			"kubernetes_secret_v1":                  resourceKubernetesSecret(),
			"kubernetes_pod":                        resourceKubernetesPod(),
			"kubernetes_pod_v1":                     resourceKubernetesPod(),
			"kubernetes_endpoints":                  resourceKubernetesEndpoints(),
			"kubernetes_endpoints_v1":               resourceKubernetesEndpoints(),
			"kubernetes_env":                        resourceKubernetesEnv(),
			"kubernetes_limit_range":                resourceKubernetesLimitRange(),
			"kubernetes_limit_range_v1":             resourceKubernetesLimitRange(),
			"kubernetes_node_taint":                 resourceKubernetesNodeTaint(),
			"kubernetes_persistent_volume":          resourceKubernetesPersistentVolume(),
			"kubernetes_persistent_volume_v1":       resourceKubernetesPersistentVolume(),
			"kubernetes_persistent_volume_claim":    resourceKubernetesPersistentVolumeClaim(),
			"kubernetes_persistent_volume_claim_v1": resourceKubernetesPersistentVolumeClaim(),
			"kubernetes_replication_controller":     resourceKubernetesReplicationController(),
			"kubernetes_replication_controller_v1":  resourceKubernetesReplicationController(),
			"kubernetes_resource_quota":             resourceKubernetesResourceQuota(),
			"kubernetes_resource_quota_v1":          resourceKubernetesResourceQuota(),

			// api registration
			"kubernetes_api_service":    resourceKubernetesAPIService(),
			"kubernetes_api_service_v1": resourceKubernetesAPIService(),

			// apps
			"kubernetes_deployment":      resourceKubernetesDeployment(),
			"kubernetes_deployment_v1":   resourceKubernetesDeployment(),
			"kubernetes_daemonset":       resourceKubernetesDaemonSet(),
			"kubernetes_daemon_set_v1":   resourceKubernetesDaemonSet(),
			"kubernetes_stateful_set":    resourceKubernetesStatefulSet(),
			"kubernetes_stateful_set_v1": resourceKubernetesStatefulSet(),

			// batch
			"kubernetes_job":         resourceKubernetesJob(),
			"kubernetes_job_v1":      resourceKubernetesJob(),
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
			"kubernetes_role":                    resourceKubernetesRole(),
			"kubernetes_role_v1":                 resourceKubernetesRole(),
			"kubernetes_role_binding":            resourceKubernetesRoleBinding(),
			"kubernetes_role_binding_v1":         resourceKubernetesRoleBinding(),
			"kubernetes_cluster_role":            resourceKubernetesClusterRole(),
			"kubernetes_cluster_role_v1":         resourceKubernetesClusterRole(),
			"kubernetes_cluster_role_binding":    resourceKubernetesClusterRoleBinding(),
			"kubernetes_cluster_role_binding_v1": resourceKubernetesClusterRoleBinding(),

			// networking
			"kubernetes_ingress":           resourceKubernetesIngress(),
			"kubernetes_ingress_v1":        resourceKubernetesIngressV1(),
			"kubernetes_ingress_class":     resourceKubernetesIngressClass(),
			"kubernetes_ingress_class_v1":  resourceKubernetesIngressClass(),
			"kubernetes_network_policy":    resourceKubernetesNetworkPolicy(),
			"kubernetes_network_policy_v1": resourceKubernetesNetworkPolicy(),

			// policy
			"kubernetes_pod_disruption_budget":       resourceKubernetesPodDisruptionBudget(),
			"kubernetes_pod_disruption_budget_v1":    resourceKubernetesPodDisruptionBudgetV1(),
			"kubernetes_pod_security_policy":         resourceKubernetesPodSecurityPolicy(),
			"kubernetes_pod_security_policy_v1beta1": resourceKubernetesPodSecurityPolicy(),

			// scheduling
			"kubernetes_priority_class":    resourceKubernetesPriorityClass(),
			"kubernetes_priority_class_v1": resourceKubernetesPriorityClass(),

			// admission control
			"kubernetes_validating_webhook_configuration":    resourceKubernetesValidatingWebhookConfiguration(),
			"kubernetes_validating_webhook_configuration_v1": resourceKubernetesValidatingWebhookConfigurationV1(),
			"kubernetes_mutating_webhook_configuration":      resourceKubernetesMutatingWebhookConfiguration(),
			"kubernetes_mutating_webhook_configuration_v1":   resourceKubernetesMutatingWebhookConfigurationV1(),

			// storage
			"kubernetes_storage_class":    resourceKubernetesStorageClass(),
			"kubernetes_storage_class_v1": resourceKubernetesStorageClass(),
			"kubernetes_csi_driver":       resourceKubernetesCSIDriver(),
			"kubernetes_csi_driver_v1":    resourceKubernetesCSIDriverV1(),

			// provider helper resources
			"kubernetes_labels":      resourceKubernetesLabels(),
			"kubernetes_annotations": resourceKubernetesAnnotations(),

			// authentication
			"kubernetes_token_request_v1": resourceKubernetesTokenRequestV1(),
		},
	}

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, d, p.TerraformVersion)
	}

	return p
}

// configPathsEnv fetches the value of the environment variable KUBE_CONFIG_PATHS, if defined.
func configPathsEnv() (interface{}, error) {
	value, exists := os.LookupEnv("KUBE_CONFIG_PATHS")
	if exists {
		log.Print("[DEBUG] using environment variable KUBE_CONFIG_PATHS to define config_paths")
		log.Printf("[DEBUG] value of KUBE_CONFIG_PATHS: %v", value)
		pathList := filepath.SplitList(value)
		configPaths := new([]interface{})
		for _, p := range pathList {
			*configPaths = append(*configPaths, p)
		}
		return *configPaths, nil
	}
	return nil, nil
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

	if err := checkConfigurationValid(k.configData); err != nil {
		return nil, err
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

var apiTokenMountPath = "/var/run/secrets/kubernetes.io/serviceaccount"

func inCluster() bool {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return false
	}

	if _, err := os.Stat(apiTokenMountPath); err != nil {
		return false
	}
	return true
}

var authDocumentationURL = "https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs#authentication"

func checkConfigurationValid(d *schema.ResourceData) error {
	if inCluster() {
		log.Printf("[DEBUG] Terraform appears to be running inside the Kubernetes cluster")
		return nil
	}

	if os.Getenv("KUBE_CONFIG_PATHS") != "" {
		return nil
	}

	atLeastOneOf := []string{
		"host",
		"config_path",
		"config_paths",
		"client_certificate",
		"token",
		"exec",
	}
	for _, a := range atLeastOneOf {
		if _, ok := d.GetOk(a); ok {
			return nil
		}
	}

	return fmt.Errorf(`provider not configured: you must configure a path to your kubeconfig
or explicitly supply credentials via the provider block or environment variables.

See our documentation at: %s`, authDocumentationURL)

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
			return logging.NewTransport("Kubernetes", rt)
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
	loader := &clientcmd.ClientConfigLoadingRules{
		WarnIfAllMissing: true,
	}

	configPaths := []string{}

	if v, ok := d.Get("config_path").(string); ok && v != "" {
		configPaths = []string{v}
	} else if v, ok := d.Get("config_paths").([]interface{}); ok && len(v) > 0 {
		for _, p := range v {
			configPaths = append(configPaths, p.(string))
		}
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
			log.Printf("[DEBUG] Using overidden context: %#v", overrides.Context)
		}
	}

	// Overriding with static configuration
	if v, ok := d.GetOk("insecure"); ok && v != "" {
		overrides.ClusterInfo.InsecureSkipTLSVerify = v.(bool)
	}
	if v, ok := d.GetOk("cluster_ca_certificate"); ok && v != "" {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("client_certificate"); ok && v != "" {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("host"); ok && v != "" {
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
	if v, ok := d.GetOk("username"); ok && v != "" {
		overrides.AuthInfo.Username = v.(string)
	}
	if v, ok := d.GetOk("password"); ok && v != "" {
		overrides.AuthInfo.Password = v.(string)
	}
	if v, ok := d.GetOk("client_key"); ok && v != "" {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("token"); ok && v != "" {
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
		useadmissionregistrationv1beta1 = ptrToBool(false)
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
	useadmissionregistrationv1beta1 = ptrToBool(true)
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
