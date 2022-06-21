package v1

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	providermetav1 "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/meta/v1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/structures"

	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func ResourceKubernetesDefaultServiceAccount() *schema.Resource {
	serviceAccountResource := ResourceKubernetesServiceAccount()

	metaSchema := providermetav1.NamespacedMetadataSchema("service account", false)

	nameField := metaSchema.Elem.(*schema.Resource).Schema["name"]
	nameField.Computed = false
	nameField.Default = "default"
	nameField.ValidateFunc = validation.StringInSlice([]string{"default"}, false)

	serviceAccountResource.Schema["metadata"] = metaSchema

	serviceAccountResource.CreateContext = resourceKubernetesDefaultServiceAccountCreate

	return serviceAccountResource
}

func resourceKubernetesDefaultServiceAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := providermetav1.ExpandMetadata(d.Get("metadata").([]interface{}))
	svcAcc := api.ServiceAccount{ObjectMeta: metadata}

	log.Printf("[INFO] Checking for default service account existence: %s", metadata.Namespace)
	err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		_, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("[INFO] Default service account does not exist, will retry: %s", metadata.Namespace)
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		log.Printf("[INFO] Default service account exists: %s", metadata.Namespace)
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	secret, err := getServiceAccountDefaultSecret(ctx, "default", svcAcc, d.Timeout(schema.TimeoutCreate), conn)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("default_secret_name", secret.Name)

	ops := providermetav1.PatchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("image_pull_secret") {
		v := d.Get("image_pull_secret").(*schema.Set).List()
		ops = append(ops, &structures.ReplaceOperation{
			Path:  "/imagePullSecrets",
			Value: expandLocalObjectReferenceArray(v),
		})
	}
	if d.HasChange("secret") {
		v := d.Get("secret").(*schema.Set).List()
		ops = append(ops, &structures.ReplaceOperation{
			Path:  "/secrets",
			Value: expandServiceAccountSecrets(v, secret.Name),
		})
	}

	automountServiceAccountToken := d.Get("automount_service_account_token").(bool)
	ops = append(ops, &structures.ReplaceOperation{
		Path:  "/automountServiceAccountToken",
		Value: automountServiceAccountToken,
	})

	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating default service account %q: %v", metadata.Name, string(data))
	out, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).Patch(ctx, metadata.Name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update default service account: %s", err)
	}
	log.Printf("[INFO] Submitted updated default service account: %#v", out)

	d.SetId(providermetav1.BuildId(metadata))

	return resourceKubernetesServiceAccountRead(ctx, d, meta)
}
