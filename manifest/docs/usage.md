## Provider configuration

The provider accepts the following configuration attributes under the `provider` block.

* `config_path` - (string) (env-var: `KUBE_CONFIG_PATH`) Path to a `kubeconfig` file.
* `host` - (string) (env-var: `KUBE_HOST`) URL to the base of the API server.
* `cluster_ca_certificate` - (string) (env-var: `KUBE_CLUSTER_CA_CERT_DATA`) PEM-encoded CA TLS certificate (including intermediates, if any).
* `client_certificate` - (string) (env-var: `KUBE_CLIENT_CERT_DATA`) PEM-encoded client TLS certificate (including intermediates, if any).
* `client_key` - (string) (env-var: `KUBE_CLIENT_KEY_DATA`) PEM-encoded private key for the above certificate.
* `username` - (string) (env-var: `KUBE_USERNAME`) Basic authentication username.
* `password` - (string) (env-var: `KUBE_PASSWORD`) Basic authentication password.
* `config_context` - (string) (env-var: `KUBE_CTX`) Context to select from the loaded `kubeconfig` file.
* `config_context_user` - (string) (env-var: `KUBE_CTX_USER`) User entry to associate to the current context (from kubeconfig).
* `config_context_cluster` - (string) (env-var: `KUBE_CTX_CLUSTER`) Cluster entry to associate to the current context (from kubeconfig).
* `token` - (string) (env-var: `KUBE_TOKEN`) Token is a bearer token used by the client for request authentication.
* `insecure` - (boolean) (env-var: `KUBE_INSECURE`) Disregard invalid TLS certificates _(default false)_.
* `exec` - (object) Exec-based authentication plugin.
  * `api_version` - (string) Version of the "client.authentication.k8s.io" API which the plugin implements.
  * `command` - (string) The plugin executable (absolute path, or expects the plugin to be in OS PATH).
  * `env` - (map string to string) Environment values to set on the plugin process.
  * `args` - (list of strings) Command line arguments to the plugin command.

All attributes are optional, but you must either set a config path or static credentials. An empty provider block will not be a functional configuration.

Due to the internal design of this provider, access to a responsive API server is required both during PLAN and APPLY. The provider makes calls to the Kubernetes API to retrieve metadata and type information during all stages of Terraform operations.

### Credentials

For authentication, the provider can be configured with identity credentials sourced from either a `kubeconfig` file, explicit values in the `provider` block, or a combination of both.

If the `config_path` attribute is set to the path of a `kubeconfig` file, the provider will load it and use the credential values in it. When `config_path` is not set **NO EXTERNAL KUBECONFIG WILL BE LOADED**. Specifically, $KUBECONFIG environment variable is **NOT** considered.

Take note of the `current-context` configured in the file. You can override it using the `config_context` provider attribute.

If both `kubeconfig` and static credentials are defined in the `provider` block, the provider will prefer any attributes specified by the static credentials and ignore the corresponding attributes in the `kubeconfig`.

There are five options for providing identity information to the provider for authentication purposes:

* a kubeconfig
* a client certificate & key pair
* a static token
* a username & password pair
* an authentication plugin, such as `oidc` or `exec` (see examples folder).
