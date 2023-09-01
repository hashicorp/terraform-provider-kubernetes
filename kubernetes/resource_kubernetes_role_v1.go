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

func resourceKubernetesRoleV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesRoleV1Create,
		ReadContext:   resourceKubernetesRoleV1Read,
		UpdateContext: resourceKubernetesRoleV1Update,
		DeleteContext: resourceKubernetesRoleV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchemaRBAC("role", true, true),
			"rule": {
				Type:        schema.TypeList,
				Description: "Rule defining a set of permissions for the role",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_groups": {
							Type:        schema.TypeSet,
							Description: "Name of the APIGroup that contains the resources",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"resources": {
							Type:        schema.TypeSet,
							Description: "List of resources that the rule applies to",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"resource_names": {
							Type:        schema.TypeSet,
							Description: "White list of names that the rule applies to",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"verbs": {
							Type:        schema.TypeSet,
							Description: "List of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesRoleV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	rules := expandRules(d.Get("rule").([]interface{}))

	role := rbacv1.Role{
		ObjectMeta: metadata,
		Rules:      *rules,
	}
	log.Printf("[INFO] Creating new role: %#v", role)
	out, err := conn.RbacV1().Roles(metadata.Namespace).Create(ctx, &role, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new role: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesRoleV1Read(ctx, d, meta)
}

func resourceKubernetesRoleV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesRoleV1Exists(ctx, d, meta)
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

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading role %s", name)
	role, err := conn.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received role: %#v", role)
	err = d.Set("metadata", flattenMetadata(role.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("rule", flattenRules(&role.Rules))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesRoleV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("rule") {
		rules := expandRules(d.Get("rule").([]interface{}))

		ops = append(ops, &ReplaceOperation{
			Path:  "/rules",
			Value: rules,
		})
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating role %q: %v", name, string(data))
	out, err := conn.RbacV1().Roles(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update role: %s", err)
	}
	log.Printf("[INFO] Submitted updated role: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesRoleV1Read(ctx, d, meta)
}

func resourceKubernetesRoleV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting role: %#v", name)
	err = conn.RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Role %s deleted", name)

	return nil
}

func resourceKubernetesRoleV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking role %s", name)
	_, err = conn.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func expandRules(rules []interface{}) *[]rbacv1.PolicyRule {
	var objects []rbacv1.PolicyRule

	for _, rule := range rules {
		in := rule.(map[string]interface{})
		obj := rbacv1.PolicyRule{}
		if v, ok := in["api_groups"].(*schema.Set); ok && v.Len() > 0 {
			obj.APIGroups = sliceOfString(v.List())
		}
		if v, ok := in["resources"].(*schema.Set); ok && v.Len() > 0 {
			obj.Resources = sliceOfString(v.List())
		}
		if v, ok := in["resource_names"].(*schema.Set); ok && v.Len() > 0 {
			obj.ResourceNames = sliceOfString(v.List())
		}
		if v, ok := in["verbs"].(*schema.Set); ok && v.Len() > 0 {
			obj.Verbs = sliceOfString(v.List())
		}
		objects = append(objects, obj)
	}

	return &objects
}

func flattenRules(in *[]rbacv1.PolicyRule) []interface{} {
	var flattened []interface{}
	for _, rule := range *in {
		att := make(map[string]interface{})
		if len(rule.APIGroups) > 0 {
			att["api_groups"] = newStringSet(schema.HashString, rule.APIGroups)
		}
		if len(rule.Resources) > 0 {
			att["resources"] = newStringSet(schema.HashString, rule.Resources)
		}
		if len(rule.ResourceNames) > 0 {
			att["resource_names"] = newStringSet(schema.HashString, rule.ResourceNames)
		}
		if len(rule.Verbs) > 0 {
			att["verbs"] = newStringSet(schema.HashString, rule.Verbs)
		}
		flattened = append(flattened, att)
	}

	return flattened
}
