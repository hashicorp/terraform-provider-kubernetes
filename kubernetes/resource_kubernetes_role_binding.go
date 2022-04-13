package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesRoleBinding() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesRoleBindingCreate,
		ReadContext:   resourceKubernetesRoleBindingRead,
		UpdateContext: resourceKubernetesRoleBindingUpdate,
		DeleteContext: resourceKubernetesRoleBindingDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchemaRBAC("roleBinding", false, true),
			"role_ref": {
				Type:        schema.TypeList,
				Description: "RoleRef references the Role for this binding",
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: rbacRoleRefSchema(),
				},
			},
			"subject": {
				Type:        schema.TypeList,
				Description: "Subjects defines the entities to bind a Role to.",
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: rbacSubjectSchema(),
				},
			},
		},
	}
}

func resourceKubernetesRoleBindingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	binding := &api.RoleBinding{
		ObjectMeta: metadata,
		RoleRef:    expandRBACRoleRef(d.Get("role_ref").([]interface{})),
		Subjects:   expandRBACSubjects(d.Get("subject").([]interface{})),
	}
	log.Printf("[INFO] Creating new RoleBinding: %#v", binding)
	out, err := conn.RbacV1().RoleBindings(metadata.Namespace).Create(ctx, binding, metav1.CreateOptions{})

	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new RoleBinding: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesRoleBindingRead(ctx, d, meta)
}

func resourceKubernetesRoleBindingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesRoleBindingExists(ctx, d, meta)
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

	log.Printf("[INFO] Reading RoleBinding %s", name)
	binding, err := conn.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received RoleBinding: %#v", binding)
	err = d.Set("metadata", flattenMetadata(binding.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedRef := flattenRBACRoleRef(binding.RoleRef)
	log.Printf("[DEBUG] Flattened RoleBinding roleRef: %#v", flattenedRef)
	err = d.Set("role_ref", flattenedRef)
	if err != nil {
		return diag.FromErr(err)
	}

	flattenedSubjects := flattenRBACSubjects(binding.Subjects)
	log.Printf("[DEBUG] Flattened RoleBinding subjects: %#v", flattenedSubjects)
	err = d.Set("subject", flattenedSubjects)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesRoleBindingUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("subject") {
		diffOps := patchRbacSubject(d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating RoleBinding %q: %v", name, string(data))
	out, err := conn.RbacV1().RoleBindings(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update RoleBinding: %s", err)
	}
	log.Printf("[INFO] Submitted updated RoleBinding: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesRoleBindingRead(ctx, d, meta)
}

func resourceKubernetesRoleBindingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting RoleBinding: %#v", name)
	err = conn.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] RoleBinding %s deleted", name)

	return nil
}

func resourceKubernetesRoleBindingExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking RoleBinding %s", name)
	_, err = conn.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
