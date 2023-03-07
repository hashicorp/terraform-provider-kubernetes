// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

var (
	cascadeDeletePolicy = metav1.DeletePropagationForeground
	deleteOptions       = metav1.DeleteOptions{
		PropagationPolicy: &cascadeDeletePolicy,
	}
)
