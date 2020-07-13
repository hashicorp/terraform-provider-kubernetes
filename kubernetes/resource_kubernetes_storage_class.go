package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
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
				ValidateFunc: validation.StringInSlice([]string{
					"Recycle",
					"Delete",
					"Retain",
				}, false),
			},
			"volume_binding_mode": {
				Type:        schema.TypeString,
				Description: "Indicates when volume binding and dynamic provisioning should occur",
				Optional:    true,
				ForceNew:    true,
				Default:     "Immediate",
				ValidateFunc: validation.StringInSlice([]string{
					"Immediate",
					"WaitForFirstConsumer",
				}, false),
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
			"allowed_topologies": {
				Type:        schema.TypeList,
				Description: "Restrict the node topologies where volumes can be dynamically provisioned.",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"match_label_expressions": {
							Type:        schema.TypeList,
							Description: "A list of topology selector requirements by labels.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Description: "The label key that the selector applies to.",
										Optional:    true,
									},
									"values": {
										Type:        schema.TypeSet,
										Description: "An array of string values. One value must match the label to be selected.",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
					},
				},
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

	if v, ok := d.GetOk("allowed_topologies"); ok && len(v.([]interface{})) > 0 {
		storageClass.AllowedTopologies = expandStorageClassAllowedTopologies(v.([]interface{}))
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

	if storageClass.AllowedTopologies != nil {
		d.Set("allowed_topologies", flattenStorageClassAllowedTopologies(storageClass.AllowedTopologies))
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

	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := conn.StorageV1().StorageClasses().Get(d.Id(), metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		e := fmt.Errorf("storage class (%s) still exists", d.Id())
		return resource.RetryableError(e)
	})
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

func expandStorageClassAllowedTopologies(l []interface{}) []v1.TopologySelectorTerm {
	if len(l) == 0 || l[0] == nil {
		return []v1.TopologySelectorTerm{}
	}

	in := l[0].(map[string]interface{})
	topologies := make([]v1.TopologySelectorTerm, 0)
	obj := v1.TopologySelectorTerm{}

	if v, ok := in["match_label_expressions"].([]interface{}); ok && len(v) > 0 {
		obj.MatchLabelExpressions = expandStorageClassMatchLabelExpressions(v)
	}

	topologies = append(topologies, obj)

	return topologies
}

func expandStorageClassMatchLabelExpressions(l []interface{}) []v1.TopologySelectorLabelRequirement {
	if len(l) == 0 || l[0] == nil {
		return []v1.TopologySelectorLabelRequirement{}
	}
	obj := make([]v1.TopologySelectorLabelRequirement, len(l), len(l))
	for i, n := range l {
		in := n.(map[string]interface{})
		obj[i] = v1.TopologySelectorLabelRequirement{
			Key:    in["key"].(string),
			Values: sliceOfString(in["values"].(*schema.Set).List()),
		}
	}
	return obj
}

func flattenStorageClassAllowedTopologies(in []v1.TopologySelectorTerm) []interface{} {
	att := make(map[string]interface{})
	for _, n := range in {
		if len(n.MatchLabelExpressions) > 0 {
			att["match_label_expressions"] = flattenStorageClassMatchLabelExpressions(n.MatchLabelExpressions)
		}
	}
	return []interface{}{att}
}

func flattenStorageClassMatchLabelExpressions(in []v1.TopologySelectorLabelRequirement) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["key"] = n.Key
		m["values"] = newStringSet(schema.HashString, n.Values)
		att[i] = m
	}
	return att
}
