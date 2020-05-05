package kubernetes

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

var (
	cascadeDeletePolicy = metav1.DeletePropagationForeground
	deleteOptions       = metav1.DeleteOptions{
		PropagationPolicy: &cascadeDeletePolicy,
	}
)
