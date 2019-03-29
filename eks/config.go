package eks

import (
	"html/template"
	"io"
)

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

func RenderConfig(info *ClusterInfo, dest io.Writer) error {
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
