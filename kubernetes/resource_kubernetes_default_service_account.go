package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	// wait for ServiceAccountsController to create default ServiceAccount in namespace
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		_, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).Get(metadata.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.SetId(buildId(metadata))

	secret, err := getServiceAccountDefaultSecret("default", svcAcc, d.Timeout(schema.TimeoutCreate), conn)
	if err != nil {
		return err
	}
	d.Set("default_secret_name", secret.Name)

	return resourceKubernetesServiceAccountUpdate(d, meta)
}
