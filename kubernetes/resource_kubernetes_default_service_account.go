package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesDefaultServiceAccount() *schema.Resource {
	resource := resourceKubernetesServiceAccount()

	metaSchema := namespacedMetadataSchema("service account", false)

	nameField := metaSchema.Elem.(*schema.Resource).Schema["name"]
	nameField.Computed = false
	nameField.Default = "default"
	nameField.ValidateFunc = validation.StringInSlice([]string{"default"}, false)

	resource.Schema["metadata"] = metaSchema

	resource.Create = resourceKubernetesDefaultServiceAccountCreate

	return resource
}

func resourceKubernetesDefaultServiceAccountCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	svcAcc := api.ServiceAccount{
		AutomountServiceAccountToken: ptrToBool(d.Get("automount_service_account_token").(bool)),
		ObjectMeta:                   metadata,
		ImagePullSecrets:             expandLocalObjectReferenceArray(d.Get("image_pull_secret").(*schema.Set).List()),
		Secrets:                      expandServiceAccountSecrets(d.Get("secret").(*schema.Set).List(), ""),
	}

	// Here we get the only chance to identify and store default secret name
	// so we can avoid showing it in diff as it's not managed by Terraform
	var svcAccTokens []api.Secret
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		resp, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).Get("default", metav1.GetOptions{})
		if err != nil {
			return resource.RetryableError(err)
		}

		d.SetId(buildId(resp.ObjectMeta))

		diff := diffObjectReferences(svcAcc.Secrets, resp.Secrets)
		secretList, err := conn.CoreV1().Secrets(resp.Namespace).List(metav1.ListOptions{
			FieldSelector: fmt.Sprintf("type=%s", api.SecretTypeServiceAccountToken),
		})
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("failed to list secrets: %v", err))
		}

		for _, secret := range secretList.Items {
			for _, svcSecret := range diff {
				if secret.Name != svcSecret.Name {
					continue
				}

				svcAccTokens = append(svcAccTokens, secret)
			}
		}

		if len(svcAccTokens) == 0 {
			return resource.RetryableError(fmt.Errorf("Expected 1 generated service account token, %d found", len(svcAccTokens)))
		}

		if len(svcAccTokens) > 1 {
			return resource.NonRetryableError(fmt.Errorf("Expected 1 generated service account token, %d found: %s", len(svcAccTokens), err))
		}

		return nil
	})
	if err != nil {
		return err
	}

	d.Set("default_secret_name", svcAccTokens[0].Name)

	return resourceKubernetesServiceAccountUpdate(d, meta)
}
