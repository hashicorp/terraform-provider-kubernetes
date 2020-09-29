package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
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
			"metadata": metadataSchemaRBAC("clusterRole", false, false),
			"rule": {
				Type:        schema.TypeList,
				Description: "List of PolicyRules for this ClusterRole",
				Optional:    true,
				Computed:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: policyRuleSchema(),
				},
			},
			"aggregation_rule": {
				Type:        schema.TypeList,
				Description: "Describes how to build the Rules for this ClusterRole.",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_role_selectors": {
							Type:        schema.TypeList,
							Description: "A list of selectors which will be used to find ClusterRoles and create the rules.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(true),
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesClusterRoleCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	cRole := api.ClusterRole{
		ObjectMeta: metadata,
		Rules:      expandClusterRoleRules(d.Get("rule").([]interface{})),
	}

	if v, ok := d.GetOk("aggregation_rule"); ok {
		cRole.AggregationRule = expandClusterRoleAggregationRule(v.([]interface{}))
	}

	log.Printf("[INFO] Creating new cluster role: %#v", cRole)
	out, err := conn.RbacV1().ClusterRoles().Create(ctx, &cRole, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new cluster role: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesClusterRoleRead(d, meta)
}

func resourceKubernetesClusterRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("rule") {
		diffOps := patchRbacRule(d)
		ops = append(ops, diffOps...)
	}
	if d.HasChange("aggregation_rule") {
		diffOps := patchRbacAggregationRule(d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating ClusterRole %q: %v", name, string(data))
	out, err := conn.RbacV1().ClusterRoles().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("Failed to update ClusterRole: %s", err)
	}
	log.Printf("[INFO] Submitted updated ClusterRole: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesClusterRoleRead(d, meta)
}

func resourceKubernetesClusterRoleRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Reading cluster role %s", name)
	cRole, err := conn.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
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
	err = d.Set("rule", flattenClusterRoleRules(cRole.Rules))
	if err != nil {
		return err
	}
	if cRole.AggregationRule != nil {
		err = d.Set("aggregation_rule", flattenClusterRoleAggregationRule(cRole.AggregationRule))
		if err != nil {
			return err
		}
	}
	return nil
}

func resourceKubernetesClusterRoleDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Deleting cluster role: %#v", name)
	err = conn.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] cluster role %s deleted", name)

	return nil
}

func resourceKubernetesClusterRoleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Checking cluster role %s", name)
	_, err = conn.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
