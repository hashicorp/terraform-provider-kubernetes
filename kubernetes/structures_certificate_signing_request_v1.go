// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	certificates "k8s.io/api/certificates/v1"
)

func expandCertificateSigningRequestV1Spec(csr []interface{}) *certificates.CertificateSigningRequestSpec {
	obj := &certificates.CertificateSigningRequestSpec{}
	if len(csr) == 0 || csr[0] == nil {
		return obj
	}
	in := csr[0].(map[string]interface{})
	obj.Request = []byte(in["request"].(string))
	if v, ok := in["usages"].(*schema.Set); ok && v.Len() > 0 {
		obj.Usages = expandCertificateSigningRequestV1Usages(v.List())
	}
	if v, ok := in["signer_name"].(string); ok && v != "" {
		obj.SignerName = v
	}
	return obj
}

func expandCertificateSigningRequestV1Usages(s []interface{}) []certificates.KeyUsage {
	out := make([]certificates.KeyUsage, len(s))
	for i, v := range s {
		out[i] = certificates.KeyUsage(v.(string))
	}
	return out
}
