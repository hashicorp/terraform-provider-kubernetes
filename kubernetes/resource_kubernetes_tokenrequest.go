package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	apiv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesTokenRequestV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesTokenRequestCreateV1,
		ReadContext:   resourceKubernetesTokenRequestReadV1,
		UpdateContext: resourceKubernetesTokenRequestUpdateV1,
		DeleteContext: resourceKubernetesTokenDeleteV1,

		Schema: map[string]*schema.Schema{
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
			"token": {
				Type:        schema.TypeString,
				Description: "Token is the opaque bearer token.",
				Computed:    true,
			},
		},
	}
}

func resourceKubernetesTokenRequestCreateV1(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandTokenRequestSpec(d.Get("spec").([]interface{}))
	saName := d.Get("metadata.0.name").(string)

	request := apiv1.TokenRequest{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new TokenRequest: %#v", request)
	out, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).CreateToken(ctx, saName, &request, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("token", out.Status.Token)

	log.Printf("[INFO] Submitted new TokenRequest: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesTokenRequestReadV1(ctx, d, meta)
}

func resourceKubernetesTokenRequestReadV1(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	return nil
}

func resourceKubernetesTokenRequestUpdateV1(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	return resourceKubernetesRoleRead(ctx, d, meta)
}

func resourceKubernetesTokenDeleteV1(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	return nil
}
