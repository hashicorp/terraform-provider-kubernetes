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

func resourceKubernetesClusterRoleBindingV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesClusterRoleBindingV1Create,
		ReadContext:   resourceKubernetesClusterRoleBindingV1Read,
		UpdateContext: resourceKubernetesClusterRoleBindingV1Update,
		DeleteContext: resourceKubernetesClusterRoleBindingV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchemaRBAC("clusterRoleBinding", true, false),
			"role_ref": {
				Type:        schema.TypeList,
				Description: "RoleRef references the Cluster Role for this binding",
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: rbacRoleRefSchema(),
				},
			},
			"subject": {
				Type:        schema.TypeList,
				Description: "Subjects defines the entities to bind a ClusterRole to.",
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: rbacSubjectSchema(),
				},
			},
		},
	}
}

func resourceKubernetesClusterRoleBindingV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metadata,
		RoleRef:    expandRBACRoleRef(d.Get("role_ref").([]interface{})),
		Subjects:   expandRBACSubjects(d.Get("subject").([]interface{})),
	}
	log.Printf("[INFO] Creating new ClusterRoleBinding: %#v", binding)
	binding, err = conn.RbacV1().ClusterRoleBindings().Create(ctx, binding, metav1.CreateOptions{})

	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new ClusterRoleBinding: %#v", binding)
	d.SetId(binding.Name)

	return resourceKubernetesClusterRoleBindingV1Read(ctx, d, meta)
}

func resourceKubernetesClusterRoleBindingV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesClusterRoleBindingV1Exists(ctx, d, meta)
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
	log.Printf("[INFO] Reading ClusterRoleBinding %s", name)
	binding, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received ClusterRoleBinding: %#v", binding)
	err = d.Set("metadata", flattenMetadata(binding.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedRef := flattenRBACRoleRef(binding.RoleRef)
	log.Printf("[DEBUG] Flattened ClusterRoleBinding roleRef: %#v", flattenedRef)
	err = d.Set("role_ref", flattenedRef)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSubjects := flattenRBACSubjects(binding.Subjects)
	log.Printf("[DEBUG] Flattened ClusterRoleBinding subjects: %#v", flattenedSubjects)
	err = d.Set("subject", flattenedSubjects)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesClusterRoleBindingV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("subject") {
		diffOps := patchRbacSubject(d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating ClusterRoleBinding %q: %v", name, string(data))
	out, err := conn.RbacV1().ClusterRoleBindings().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update ClusterRoleBinding: %s", err)
	}
	log.Printf("[INFO] Submitted updated ClusterRoleBinding: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesClusterRoleBindingV1Read(ctx, d, meta)
}

func resourceKubernetesClusterRoleBindingV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	log.Printf("[INFO] Deleting ClusterRoleBinding: %#v", name)
	err = conn.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}
	log.Printf("[INFO] ClusterRoleBinding %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesClusterRoleBindingV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()
	log.Printf("[INFO] Checking ClusterRoleBinding %s", name)
	_, err = conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
