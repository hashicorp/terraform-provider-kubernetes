package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
)

func dataSourceKubernetesServiceAccountToken() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesServiceAccountTokenRead,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service account token", false),

			"data": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ca_crt": {
							Type:        schema.TypeString,
							Description: "CA certificate for the apiserver - this is 'ca.crt' in the underlying secret",
							Computed:    true,
						},
						"namespace": {
							Type:        schema.TypeString,
							Description: "Namespace that the service account token is application for.",
							Computed:    true,
						},
						"token": {
							Type:        schema.TypeString,
							Description: "Bearer token used to authenticate against the apiserver.",
							Computed:    true,
							Sensitive:   true,
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesServiceAccountTokenRead(d *schema.ResourceData, meta interface{}) error {
	om := meta_v1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading service account token %s", name)
	secret, err := conn.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return err
	}

	if secret.Type != core_v1.SecretTypeServiceAccountToken {
		return fmt.Errorf("Incorrect secret type: %v", secret.Type)
	}

	log.Printf("[INFO] Received service account token: %#v", secret)
	err = d.Set("metadata", flattenMetadata(secret.ObjectMeta))
	if err != nil {
		return err
	}

	data := byteMapToStringMap(secret.Data)

	att := make(map[string]interface{})

	att["ca_crt"] = data[core_v1.ServiceAccountRootCAKey]
	att["namespace"] = data[core_v1.ServiceAccountNamespaceKey]
	att["token"] = data[core_v1.ServiceAccountTokenKey]
	d.Set("data", []interface{}{att})

	return err
}
