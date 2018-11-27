package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	api "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
)

func resourceKubernetesClusterRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesClusterRoleCreate,
		Read:   resourceKubernetesClusterRoleRead,
		Exists: resourceKubernetesClusterRoleExists,
		Update: resourceKubernetesClusterRoleUpdate,
		Delete: resourceKubernetesClusterRoleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("cluster role", true),
			"rule": {
				Type:        schema.TypeList,
				Description: "List of PolicyRules for this ClusterRole",
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: policyRuleFields(),
				},
			},
		},
	}
}

func resourceKubernetesClusterRoleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	cRole := api.ClusterRole{
		ObjectMeta: metadata,
		Rules:      expandClusterRoleRule(d.Get("rule").([]interface{})),
	}
	log.Printf("[INFO] Creating new cluster role: %#v", cRole)
	out, err := conn.RbacV1().ClusterRoles().Create(&cRole)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new cluster role: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesClusterRoleRead(d, meta)
}

func resourceKubernetesClusterRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	_, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	cRole := api.ClusterRole{
		ObjectMeta: metadata,
		Rules:      expandClusterRoleRule(d.Get("rule").([]interface{})),
	}

	log.Printf("[INFO] Updating cluster role %q: %v", name, cRole)
	out, err := conn.RbacV1().ClusterRoles().Update(&cRole)
	if err != nil {
		return fmt.Errorf("Failed to update cluster role: %s", err)
	}
	log.Printf("[INFO] Submitted updated cluster role: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesClusterRoleRead(d, meta)
}

func resourceKubernetesClusterRoleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	_, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Reading cluster role %s", name)
	cRole, err := conn.RbacV1().ClusterRoles().Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received cluster role: %#v", cRole)
	err = d.Set("metadata", flattenMetadata(cRole.ObjectMeta))
	if err != nil {
		return err
	}
	d.Set("rule", flattenClusterRoleRules(cRole.Rules))

	return nil
}

func resourceKubernetesClusterRoleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	_, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Deleting cluster role: %#v", name)
	err = conn.RbacV1().ClusterRoles().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] cluster role %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesClusterRoleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	_, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking cluster role %s", name)
	_, err = conn.RbacV1().ClusterRoles().Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
