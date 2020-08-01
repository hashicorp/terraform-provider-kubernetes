package kubernetes

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesClusterRole() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesClusterRoleRead,

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchemaRBAC("clusterRole", false, false),
			"rules": {
				Type:        schema.TypeList,
				Description: "List of PolicyRules for this ClusterRole",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: policyRuleSchema(),
				},
			},
		},
	}
}

func dataSourceKubernetesClusterRoleRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	cRole, err := conn.RbacV1().ClusterRoles().Get(metadata.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received cluster role: %#v", cRole)
	err = d.Set("metadata", flattenMetadata(cRole.ObjectMeta, d))
	if err != nil {
		return err
	}
	d.Set("rules", flattenClusterRoleRules(cRole.Rules))
	d.SetId(cRole.Name)

	return nil
}
