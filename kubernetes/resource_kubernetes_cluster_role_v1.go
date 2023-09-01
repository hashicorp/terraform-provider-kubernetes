// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesClusterRoleV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesClusterRoleV1Create,
		ReadContext:   resourceKubernetesClusterRoleV1Read,
		UpdateContext: resourceKubernetesClusterRoleV1Update,
		DeleteContext: resourceKubernetesClusterRoleV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchemaRBAC("clusterRole", true, false),
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

func resourceKubernetesClusterRoleV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	cRole := rbacv1.ClusterRole{
		ObjectMeta: metadata,
		Rules:      expandClusterRoleRules(d.Get("rule").([]interface{})),
	}

	if v, ok := d.GetOk("aggregation_rule"); ok {
		cRole.AggregationRule = expandClusterRoleAggregationRule(v.([]interface{}))
	}

	log.Printf("[INFO] Creating new cluster role: %#v", cRole)
	out, err := conn.RbacV1().ClusterRoles().Create(ctx, &cRole, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new cluster role: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesClusterRoleV1Read(ctx, d, meta)
}

func resourceKubernetesClusterRoleV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

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
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating ClusterRole %q: %v", name, string(data))
	out, err := conn.RbacV1().ClusterRoles().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update ClusterRole: %s", err)
	}
	log.Printf("[INFO] Submitted updated ClusterRole: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesClusterRoleV1Read(ctx, d, meta)
}

func resourceKubernetesClusterRoleV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesClusterRoleV1Exists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	log.Printf("[INFO] Reading cluster role %s", name)
	cRole, err := conn.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received cluster role: %#v", cRole)
	err = d.Set("metadata", flattenMetadata(cRole.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("rule", flattenClusterRoleRules(cRole.Rules))
	if err != nil {
		return diag.FromErr(err)
	}
	if cRole.AggregationRule != nil {
		err = d.Set("aggregation_rule", flattenClusterRoleAggregationRule(cRole.AggregationRule))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func resourceKubernetesClusterRoleV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	log.Printf("[INFO] Deleting cluster role: %#v", name)
	err = conn.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] cluster role %s deleted", name)

	return nil
}

func resourceKubernetesClusterRoleV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

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
