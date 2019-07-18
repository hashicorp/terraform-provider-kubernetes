package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	api "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
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
			"metadata": metadataSchemaClusterRole(),
			"rule": {
				Type:        schema.TypeList,
				Description: "List of PolicyRules for this ClusterRole",
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: policyRuleSchema(),
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
		Rules:      expandClusterRoleRules(d.Get("rule").([]interface{})),
	}
	log.Printf("[INFO] Creating new cluster role: %#v", cRole)
	out, err := conn.RbacV1().ClusterRoles().Create(&cRole)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new cluster role: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesClusterRoleRead(d, meta)
}

func resourceKubernetesClusterRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("rule") {
		diffOps := patchRbacRule(d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating ClusterRole %q: %v", name, string(data))
	out, err := conn.Rbac().ClusterRoles().Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update ClusterRole: %s", err)
	}
	log.Printf("[INFO] Submitted updated ClusterRole: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesClusterRoleRead(d, meta)
}

func resourceKubernetesClusterRoleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Reading cluster role %s", name)
	cRole, err := conn.RbacV1().ClusterRoles().Get(name, metav1.GetOptions{})
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
	d.Set("rule", flattenClusterRoleRules(cRole.Rules))

	return nil
}

func resourceKubernetesClusterRoleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Deleting cluster role: %#v", name)
	err := conn.RbacV1().ClusterRoles().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] cluster role %s deleted", name)

	return nil
}

func resourceKubernetesClusterRoleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Checking cluster role %s", name)
	_, err := conn.RbacV1().ClusterRoles().Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
