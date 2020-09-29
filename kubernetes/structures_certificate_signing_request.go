package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"k8s.io/api/certificates/v1beta1"
)

func expandCertificateSigningRequestSpec(csr []interface{}) (*v1beta1.CertificateSigningRequestSpec, error) {
	obj := &v1beta1.CertificateSigningRequestSpec{}
	if len(csr) == 0 || csr[0] == nil {
		return obj, nil
	}
	in := csr[0].(map[string]interface{})
	obj.Request = []byte(in["request"].(string))
	if v, ok := in["usages"].(*schema.Set); ok && v.Len() > 0 {
		obj.Usages = expandCertificateSigningRequestUsages(v.List())
	}
	return obj, nil
}

func expandCertificateSigningRequestUsages(s []interface{}) []v1beta1.KeyUsage {
	out := make([]v1beta1.KeyUsage, len(s), len(s))
	for i, v := range s {
		out[i] = v1beta1.KeyUsage(v.(string))
	}
	return out
}
