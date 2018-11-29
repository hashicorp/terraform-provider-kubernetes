package kubernetes

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func resourceKubernetesKubeSystemNamespace() *schema.Resource {
	// reuse kubernetes_namespace schema, and methods for READ, UPDATE
	kubeSystemNamespace := resourceKubernetesNamespace()
	kubeSystemNamespace.Create = resourceKubernetesKubeSystemNamespaceCreate
	kubeSystemNamespace.Delete = resourceKubernetesKubeSystemNamespaceDelete

	// cidr_block is a computed value for Default VPCs
	kubeSystemNamespace.Schema["cidr_block"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// instance_tenancy is a computed value for Default VPCs
	kubeSystemNamespace.Schema["instance_tenancy"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// assign_generated_ipv6_cidr_block is a computed value for Default VPCs
	kubeSystemNamespace.Schema["assign_generated_ipv6_cidr_block"] = &schema.Schema{
		Type:     schema.TypeBool,
		Computed: true,
	}

	return kubeSystemNamespace
}

func resourceKubernetesKubeSystemNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, err := conn.CoreV1().Namespaces().Get("kube-system", meta_v1.GetOptions{})
	if err != nil {
		return err
	}

	d.SetId(namespace.Name)

	return resourceKubernetesNamespaceUpdate(d, meta)
}

func resourceKubernetesKubeSystemNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Cannot destroy the kube-system namespace. Terraform will remove this resource from the state file, however resources may remain.")
	return nil
}
