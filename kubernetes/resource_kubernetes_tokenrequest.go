package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	apiv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesTokenRequest() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesTokenRequestCreate,
		ReadContext:   resourceKubernetesTokenRequestRead,
		UpdateContext: resourceKubernetesTokenRequestUpdate,
		DeleteContext: resourceKubernetesTokenDelete,

		Schema: map[string]*schema.Schema{
			"serviceAcccount": {
				Type:        schema.TypeString,
				Description: "Service Account name that will receive the token request.",
				Required:    true,
			},
			"metadata": namespacedMetadataSchema("token request", true),
			"spec": {
				Type:        schema.TypeList,
				Description: apiv1.TokenRequest{}.Spec.SwaggerDoc()["spec"],
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: tokenRequestSpecFields(),
				},
			},
		},
	}
}

func resourceKubernetesTokenRequestCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandTokenRequestSpec(d.Get("spec").([]interface{}))
	saName := d.Get("serviceAcccount").(string)

	request := apiv1.TokenRequest{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new TokenRequest: %#v", request)
	// TODO: ask how we would get the service account name, probably add an attribute to it.
	out, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).CreateToken(ctx, saName, &request, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new TokenRequest: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesTokenRequestRead(ctx, d, meta)
}

func resourceKubernetesTokenRequestRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// exists, err := resourceKubernetesRoleExists(ctx, d, meta)
	// if err != nil {
	// 	return diag.FromErr(err)
	// }
	// if !exists {
	// 	d.SetId("")
	// 	return diag.Diagnostics{}
	// }
	// conn, err := meta.(KubeClientsets).MainClientset()
	// if err != nil {
	// 	return diag.FromErr(err)
	// }

	// namespace, name, err := idParts(d.Id())
	// if err != nil {
	// 	return diag.FromErr(err)
	// }

	// log.Printf("[INFO] Reading role %s", name)
	// role, err := conn.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	// if err != nil {
	// 	log.Printf("[DEBUG] Received error: %#v", err)
	// 	return diag.FromErr(err)
	// }

	// log.Printf("[INFO] Received role: %#v", role)
	// err = d.Set("metadata", flattenMetadata(role.ObjectMeta, d, meta))
	// if err != nil {
	// 	return diag.FromErr(err)
	// }

	// err = d.Set("rule", flattenRules(&role.Rules))
	// if err != nil {
	// 	return diag.FromErr(err)
	// }

	return nil
}

func resourceKubernetesTokenRequestUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	// data, err := ops.MarshalJSON()
	// if err != nil {
	// 	return diag.Errorf("Failed to marshal update operations: %s", err)
	// }
	// log.Printf("[INFO] Updating role %q: %v", name, string(data))
	// out, err := conn.RbacV1().Roles(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	// if err != nil {
	// 	return diag.Errorf("Failed to update role: %s", err)
	// }
	// log.Printf("[INFO] Submitted updated role: %#v", out)
	// d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesRoleRead(ctx, d, meta)
}

func resourceKubernetesTokenDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting Token: %#v", name)
	err = conn.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Token %s deleted", name)

	return nil
}
