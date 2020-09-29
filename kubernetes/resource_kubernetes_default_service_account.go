package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesDefaultServiceAccount() *schema.Resource {
	serviceAccountResource := resourceKubernetesServiceAccount()

	metaSchema := namespacedMetadataSchema("service account", false)

	nameField := metaSchema.Elem.(*schema.Resource).Schema["name"]
	nameField.Computed = false
	nameField.Default = "default"
	nameField.ValidateFunc = validation.StringInSlice([]string{"default"}, false)

	serviceAccountResource.Schema["metadata"] = metaSchema

	serviceAccountResource.Create = resourceKubernetesDefaultServiceAccountCreate

	return serviceAccountResource
}

func resourceKubernetesDefaultServiceAccountCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	svcAcc := api.ServiceAccount{ObjectMeta: metadata}

	log.Printf("[INFO] Checking for default service account existence: %s", metadata.Namespace)
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
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
		return err
	}

	d.SetId(buildId(metadata))

	secret, err := getServiceAccountDefaultSecret(ctx, "default", svcAcc, d.Timeout(schema.TimeoutCreate), conn)
	if err != nil {
		return err
	}
	d.Set("default_secret_name", secret.Name)

	return resourceKubernetesServiceAccountUpdate(d, meta)
}
