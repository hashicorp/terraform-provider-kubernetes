package eks

import (
	"strings"
	"testing"
)

const expected = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: baaaahhhhh=
    server: https://yo-own-eks.us-west-none.eks.amazonaws.com
  name: arn:aws:us-west-none:11111:cluster/yo-own-eks
contexts:
- context:
    cluster: arn:aws:us-west-none:11111:cluster/yo-own-eks
    user: arn:aws:us-west-none:11111:cluster/yo-own-eks
  name: arn:aws:us-west-none:11111:cluster/yo-own-eks
current-context: arn:aws:us-west-none:11111:cluster/yo-own-eks
kind: Config
preferences: {}
users:
- name: arn:aws:us-west-none:11111:cluster/yo-own-eks
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - token
      - -i
      - stack-eks-cluster-dev
      command: aws-iam-authenticator
`

func TestEksConfigWritesTemplateCorrectly(t *testing.T) {
	info := &ClusterInfo{
		Arn:                  "arn:aws:us-west-none:11111:cluster/yo-own-eks",
		Endpoint:             "https://yo-own-eks.us-west-none.eks.amazonaws.com",
		CertificateAuthority: "baaaahhhhh=",
	}

	var dest strings.Builder

	err := RenderConfig(info, &dest)
	if err != nil {
		t.Errorf("Couldn't write config to test: %s", err.Error())
	}
	if actual := dest.String(); actual != expected {
		t.Errorf("Did not render template correctly. Wanted:\n'%s'\nGot:\n'%s'", expected, actual)
	}
}
