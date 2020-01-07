package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesServiceAccount() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesServiceAccountRead,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service account", false),
			"image_pull_secret": {
				Type:        schema.TypeSet,
				Description: "A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. More info: http://kubernetes.io/docs/user-guide/secrets#manually-specifying-an-imagepullsecret",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
							Computed:    true,
						},
					},
				},
			},
			"secret": {
				Type:        schema.TypeSet,
				Description: "A list of secrets allowed to be used by pods running using this Service Account. More info: http://kubernetes.io/docs/user-guide/secrets",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
							Computed:    true,
						},
					},
				},
			},
			"automount_service_account_token": {
				Type:        schema.TypeBool,
				Description: "True to enable automatic mounting of the service account token",
				Computed:    true,
			},
			"default_secret_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceKubernetesServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	om := meta_v1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	return resourceKubernetesServiceAccountRead(d, meta)
}
