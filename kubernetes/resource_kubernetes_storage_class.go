package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	v1 "k8s.io/api/core/v1"
	api "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesStorageClass() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesStorageClassCreate,
		Read:   resourceKubernetesStorageClassRead,
		Exists: resourceKubernetesStorageClassExists,
		Update: resourceKubernetesStorageClassUpdate,
		Delete: resourceKubernetesStorageClassDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("storage class", true),
			"parameters": {
				Type:        schema.TypeMap,
				Description: "The parameters for the provisioner that should create volumes of this storage class",
				Optional:    true,
				ForceNew:    true,
			},
			"storage_provisioner": {
				Type:        schema.TypeString,
				Description: "Indicates the type of the provisioner",
				Required:    true,
				ForceNew:    true,
			},
			"reclaim_policy": {
				Type:        schema.TypeString,
				Description: "Indicates the type of the reclaim policy",
				Optional:    true,
				Default:     "Delete",
			},
			"volume_binding_mode": {
				Type:        schema.TypeString,
				Description: "Indicates when volume binding and dynamic provisioning should occur",
				Optional:    true,
				ForceNew:    true,
				Default:     "Immediate",
			},
			"allow_volume_expansion": {
				Type:        schema.TypeBool,
				Description: "Indicates whether the storage class allow volume expand",
				Optional:    true,
				Default:     true,
			},
			"mount_options": {
				Type:        schema.TypeSet,
				Description: "Persistent Volumes that are dynamically created by a storage class will have the mount options specified",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},
		},
	}
}

func resourceKubernetesStorageClassCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	reclaimPolicy := v1.PersistentVolumeReclaimPolicy(d.Get("reclaim_policy").(string))
	volumeBindingMode := api.VolumeBindingMode(d.Get("volume_binding_mode").(string))
	allowVolumeExpansion := d.Get("allow_volume_expansion").(bool)
	storageClass := api.StorageClass{
		ObjectMeta:           metadata,
		Provisioner:          d.Get("storage_provisioner").(string),
		ReclaimPolicy:        &reclaimPolicy,
		VolumeBindingMode:    &volumeBindingMode,
		AllowVolumeExpansion: &allowVolumeExpansion,
	}

	if v, ok := d.GetOk("parameters"); ok {
		storageClass.Parameters = expandStringMap(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("mount_options"); ok {
		storageClass.MountOptions = schemaSetToStringArray(v.(*schema.Set))
	}

	log.Printf("[INFO] Creating new storage class: %#v", storageClass)
	out, err := conn.StorageV1().StorageClasses().Create(ctx, &storageClass, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new storage class: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesStorageClassRead(d, meta)
}

func resourceKubernetesStorageClassRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Reading storage class %s", name)
	storageClass, err := conn.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received storage class: %#v", storageClass)
	err = d.Set("metadata", flattenMetadata(storageClass.ObjectMeta, d))
	if err != nil {
		return err
	}
	d.Set("parameters", storageClass.Parameters)
	d.Set("storage_provisioner", storageClass.Provisioner)
	d.Set("reclaim_policy", storageClass.ReclaimPolicy)
	d.Set("volume_binding_mode", storageClass.VolumeBindingMode)
	d.Set("mount_options", newStringSet(schema.HashString, storageClass.MountOptions))
	if storageClass.AllowVolumeExpansion != nil {
		d.Set("allow_volume_expansion", *storageClass.AllowVolumeExpansion)
	}

	return nil
}

func resourceKubernetesStorageClassUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	ops := patchMetadata("metadata.0.", "/metadata/", d)
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating storage class %q: %v", name, string(data))
	out, err := conn.StorageV1().StorageClasses().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("Failed to update storage class: %s", err)
	}
	log.Printf("[INFO] Submitted updated storage class: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesStorageClassRead(d, meta)
}

func resourceKubernetesStorageClassDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Deleting storage class: %#v", name)
	err = conn.StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Storage class %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesStorageClassExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Checking storage class %s", name)
	_, err = conn.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
