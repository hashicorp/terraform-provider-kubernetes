package kubernetes

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-homedir"
	kubernetes "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
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
			"config_path": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{
						"KUBE_CONFIG",
						"KUBECONFIG",
					},
					"~/.kube/config"),
				Description: "Path to the kube config file, defaults to ~/.kube/config",
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
			"load_config_file": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_LOAD_CONFIG_FILE", true),
				Description: "Load local kubeconfig.",
			},
			"eks_cluster_name": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_EKS_CLUSTER_NAME", ""),
				Description: "Name of eks cluster to try to autoload",
			},
			"eks_cluster_region": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_REGION_DO_I_REALLY_WANT_THIS", ""),
				Description: "adjlsfkdslfds",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"kubernetes_secret":        dataSourceKubernetesSecret(),
			"kubernetes_service":       dataSourceKubernetesService(),
			"kubernetes_storage_class": dataSourceKubernetesStorageClass(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"kubernetes_cluster_role":              resourceKubernetesClusterRole(),
			"kubernetes_cluster_role_binding":      resourceKubernetesClusterRoleBinding(),
			"kubernetes_config_map":                resourceKubernetesConfigMap(),
			"kubernetes_daemonset":                 resourceKubernetesDaemonSet(),
			"kubernetes_deployment":                resourceKubernetesDeployment(),
			"kubernetes_endpoint":                  resourceKubernetesEndpoint(),
			"kubernetes_horizontal_pod_autoscaler": resourceKubernetesHorizontalPodAutoscaler(),
			"kubernetes_limit_range":               resourceKubernetesLimitRange(),
			"kubernetes_namespace":                 resourceKubernetesNamespace(),
			"kubernetes_network_policy":            resourceKubernetesNetworkPolicy(),
			"kubernetes_persistent_volume":         resourceKubernetesPersistentVolume(),
			"kubernetes_persistent_volume_claim":   resourceKubernetesPersistentVolumeClaim(),
			"kubernetes_pod":                       resourceKubernetesPod(),
			"kubernetes_replication_controller":    resourceKubernetesReplicationController(),
			"kubernetes_role_binding":              resourceKubernetesRoleBinding(),
			"kubernetes_resource_quota":            resourceKubernetesResourceQuota(),
			"kubernetes_role":                      resourceKubernetesRole(),
			"kubernetes_secret":                    resourceKubernetesSecret(),
			"kubernetes_service":                   resourceKubernetesService(),
			"kubernetes_service_account":           resourceKubernetesServiceAccount(),
			"kubernetes_stateful_set":              resourceKubernetesStatefulSet(),
			"kubernetes_storage_class":             resourceKubernetesStorageClass(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	var cfg *restclient.Config
	var err error
	if d.Get("load_config_file").(bool) {
		// Config file loading
		cfg, err = tryLoadingConfigFile(d)
	}

	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &restclient.Config{}
	}

	// Overriding with static configuration
	cfg.UserAgent = fmt.Sprintf("HashiCorp/1.0 Terraform/%s", terraform.VersionString())

	if v, ok := d.GetOk("host"); ok {
		cfg.Host = v.(string)
	}
	if v, ok := d.GetOk("username"); ok {
		cfg.Username = v.(string)
	}
	if v, ok := d.GetOk("password"); ok {
		cfg.Password = v.(string)
	}
	if v, ok := d.GetOk("insecure"); ok {
		cfg.Insecure = v.(bool)
	}
	if v, ok := d.GetOk("cluster_ca_certificate"); ok {
		cfg.CAData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("client_certificate"); ok {
		cfg.CertData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("client_key"); ok {
		cfg.KeyData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("token"); ok {
		cfg.BearerToken = v.(string)
	}

	k, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("Failed to configure: %s", err)
	}

	return k, nil
}

func tryLoadingConfigFile(d *schema.ResourceData) (*restclient.Config, error) {
	path, err := homedir.Expand(d.Get("config_path").(string))
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] eks_cluster_name is %v\n", d.Get("eks_cluster_name"))
	// but eks_cluster_name overrides it all! bwahahaha
	if d.Get("eks_cluster_name") != nil {
		writeVolatileEksConfigFile(d.Get("eks_cluster_name").(string), d.Get("eks_cluster_region").(string))
		path = configpath
	}

	loader := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: path,
	}

	overrides := &clientcmd.ConfigOverrides{}
	ctxSuffix := "; default context"

	ctx, ctxOk := d.GetOk("config_context")
	authInfo, authInfoOk := d.GetOk("config_context_auth_info")
	cluster, clusterOk := d.GetOk("config_context_cluster")
	if ctxOk || authInfoOk || clusterOk {
		ctxSuffix = "; overriden context"
		if ctxOk {
			overrides.CurrentContext = ctx.(string)
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

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	cfg, err := cc.ClientConfig()
	if err != nil {
		if pathErr, ok := err.(*os.PathError); ok && os.IsNotExist(pathErr.Err) {
			log.Printf("[INFO] Unable to load config file as it doesn't exist at %q", path)
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to load config (%s%s): %s", path, ctxSuffix, err)
	}

	log.Printf("[INFO] Successfully loaded config file (%s%s)", path, ctxSuffix)
	return cfg, nil
}

const configpath = "/tmp/ekstfprovider"
const kubeconfigTemplate = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: {{ .CertificateAuthority }}
    server: {{ .Endpoint }}
  name: {{ .Arn }}
contexts:
- context:
    cluster: {{ .Arn }}
    user: {{ .Arn }}
  name: {{ .Arn }}
current-context: {{ .Arn }}
kind: Config
preferences: {}
users:
- name: {{ .Arn }}
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - token
      - -i
      - stack-eks-cluster-dev
      command: aws-iam-authenticator
`

type ClusterInfo struct {
	Arn                  string
	Endpoint             string
	CertificateAuthority string
}

func getEksInfo(session *session.Session) (*ClusterInfo, error) {
	eksClient := eks.New(session)

	dci := &eks.DescribeClusterInput{Name: aws.String("stack-eks-cluster-dev")}
	dco, err := eksClient.DescribeCluster(dci)
	if err != nil {
		return nil, err
	}

	info := &ClusterInfo{}
	info.Arn = *dco.Cluster.Arn
	info.Endpoint = *dco.Cluster.Endpoint
	info.CertificateAuthority = *dco.Cluster.CertificateAuthority.Data
	return info, nil
}

func renderConfig(info *ClusterInfo, dest io.Writer) error {
	tmpl, err := template.New("kubeconfig").Parse(kubeconfigTemplate)
	if err != nil {
		return err
	}

	err = tmpl.Execute(dest, *info)
	if err != nil {
		return err
	}

	return nil
}

func getAwsSession(region string) *session.Session {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	return sess
}

func writeVolatileEksConfigFile(clustername, awsregion string) {
	session := getAwsSession(awsregion)
	info, _ := getEksInfo(session)
	f, err := os.Create(configpath)
	if err != nil {
		log.Printf("[ERROR] can't write temporary kubeconfig: %s\n", err.Error())
	}
	defer f.Close()
	err = renderConfig(info, f)
	if err != nil {
		log.Printf("[ERROR] couldn't write config: %s\n", err.Error())
	}
}
