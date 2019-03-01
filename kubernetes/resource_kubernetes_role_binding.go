package kubernetes

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	api "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"log"
)

func resourceKubernetesRoleBinding() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesRoleBindingCreate,
		Read:   resourceKubernetesRoleBindingRead,
		Exists: resourceKubernetesRoleBindingExists,
		Update: resourceKubernetesRoleBindingUpdate,
		Delete: resourceKubernetesRoleBindingDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("roleBinding", false),
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

func resourceKubernetesRoleBindingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	binding := &api.RoleBinding{
		ObjectMeta: metadata,
		RoleRef:    expandRBACRoleRef(d.Get("role_ref").([]interface{})),
		Subjects:   expandRBACSubjects(d.Get("subject").([]interface{})),
	}
	log.Printf("[INFO] Creating new RoleBinding: %#v", binding)
	out, err := conn.RbacV1().RoleBindings(metadata.Namespace).Create(binding)

	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new RoleBinding: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesRoleBindingRead(d, meta)
}

func resourceKubernetesRoleBindingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading RoleBinding %s", name)
	binding, err := conn.RbacV1().RoleBindings(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}

	log.Printf("[INFO] Received RoleBinding: %#v", binding)
	err = d.Set("metadata", flattenMetadata(binding.ObjectMeta))
	if err != nil {
		return err
	}

	flattenedRef := flattenRBACRoleRef(binding.RoleRef)
	log.Printf("[DEBUG] Flattened RoleBinding roleRef: %#v", flattenedRef)
	err = d.Set("role_ref", flattenedRef)
	if err != nil {
		return err
	}

	flattenedSubjects := flattenRBACSubjects(binding.Subjects)
	log.Printf("[DEBUG] Flattened RoleBinding subjects: %#v", flattenedSubjects)
	err = d.Set("subject", flattenedSubjects)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesRoleBindingUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("subject") {
		diffOps := patchRbacSubject(d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating RoleBinding %q: %v", name, string(data))
	out, err := conn.RbacV1().RoleBindings(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update RoleBinding: %s", err)
	}
	log.Printf("[INFO] Submitted updated RoleBinding: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesRoleBindingRead(d, meta)
}

func resourceKubernetesRoleBindingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting RoleBinding: %#v", name)
	err = conn.RbacV1().RoleBindings(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] RoleBinding %s deleted", name)

	return nil
}

func resourceKubernetesRoleBindingExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking RoleBinding %s", name)
	_, err = conn.RbacV1().RoleBindings(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
