// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package util

const (
	BootstrapConnectionKey = "bootstrap_connection"

	BootstrapConnectionDescription = "Connection settings used while this resource is being created, replacing the " +
		"provider connection configuration in whole. After creation the provider connection configuration will be " +
		"used. This is to support creating resources when the credentials are not " +
		"available at plan time (e.g. when creating a cluster and configuring it in the same stack). For managing " +
		"resources in multiple clusters use aliased providers instead."

	ProviderRequiredSummary = "Provider configuration required for this operation"

	ProviderRequiredDetail = `This resource has a "bootstrap_connection" block, but the provider block it belongs to has no ` +
		"Kubernetes configuration.\n\n" +
		`A "bootstrap_connection" block only supplies credentials while the resource is being created. ` +
		"Reading or destroying the resource afterwards uses the provider, so that it always " +
		"authenticates with freshly evaluated credentials rather than whatever was recorded in state at " +
		"the last apply.\n\n" +
		`Configure a provider for the target cluster - typically an aliased "kubernetes" provider whose ` +
		`settings come from a data source - and associate this resource with it using the "provider" ` +
		"meta-argument."

	UnusableConnectionSummary = `Invalid "bootstrap_connection" block`

	UnusableConnectionDetail = `The "bootstrap_connection" block did not resolve to a usable Kubernetes client ` +
		`configuration. Note that, unlike the provider block, a "bootstrap_connection" block does not read KUBE_* ` +
		`environment variables: it must specify the cluster's address and credentials itself (for example "host" ` +
		`plus "cluster_ca_certificate" and "token", or "config_path").`

	BootstrapConnectionHostDescription                  = "The hostname (in form of URI) of the Kubernetes API."
	BootstrapConnectionUsernameDescription              = "The username to use for HTTP basic authentication when accessing the Kubernetes API."
	BootstrapConnectionPasswordDescription              = "The password to use for HTTP basic authentication when accessing the Kubernetes API."
	BootstrapConnectionInsecureDescription              = "Whether the server should be accessed without verifying the TLS certificate. Defaults to `false`."
	BootstrapConnectionTLSServerNameDescription         = "Server name passed to the server for SNI and is used in the client to check server certificates against."
	BootstrapConnectionClientCertificateDescription     = "PEM-encoded client certificate for TLS authentication."
	BootstrapConnectionClientKeyDescription             = "PEM-encoded client certificate key for TLS authentication."
	BootstrapConnectionClusterCACertificateDescription  = "PEM-encoded root certificates bundle for TLS authentication."
	BootstrapConnectionConfigPathsDescription           = "A list of paths to the kube config files."
	BootstrapConnectionConfigPathDescription            = "A path to a kube config file."
	BootstrapConnectionConfigContextDescription         = "Context to choose from the config file."
	BootstrapConnectionConfigContextAuthInfoDescription = "Authentication info context of the kube config (name of the kubeconfig user, `--user` flag in `kubectl`)."
	BootstrapConnectionConfigContextClusterDescription  = "Cluster context of the kube config (name of the kubeconfig cluster, `--cluster` flag in `kubectl`)."
	BootstrapConnectionTokenDescription                 = "Token of your service account."
	BootstrapConnectionProxyURLDescription              = `URL to the proxy to be used for all API requests. URLs with "http", "https", and "socks5" schemes are supported.`

	BootstrapConnectionExecDescription           = "Configuration block to use an exec-based credential plugin, e.g. call an external command to receive user credentials."
	BootstrapConnectionExecAPIVersionDescription = "API version to use when decoding the ExecCredentials resource, e.g. `client.authentication.k8s.io/v1`."
	BootstrapConnectionExecCommandDescription    = "Command to execute."
	BootstrapConnectionExecEnvDescription        = "Map of environment variables to set when executing the plugin."
	BootstrapConnectionExecArgsDescription       = "List of arguments to pass when executing the plugin."
)
