package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	api "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesCSIDriver() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesCSIDriverCreate,
		Read:   resourceKubernetesCSIDriverRead,
		Exists: resourceKubernetesCSIDriverExists,
		Update: resourceKubernetesCSIDriverUpdate,
		Delete: resourceKubernetesCSIDriverDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("csi driver", true),
			"spec": {
				Type:        schema.TypeList,
				Description: fmt.Sprintf("Spec of the CSIDriver"),
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attach_required": {
							Type:        schema.TypeBool,
							Description: "Indicates if the CSI volume driver requires an attach operation",
							Required:    true,
							ForceNew:    true,
						},
						"pod_info_on_mount": {
							Type:        schema.TypeBool,
							Description: "Indicates that the CSI volume driver requires additional pod information (like podName, podUID, etc.) during mount operations",
							Optional:    true,
						},
						"volume_lifecycle_modes": {
							Type:        schema.TypeList,
							Description: "Defines what kind of volumes this CSI volume driver supports",
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"Persistent",
									"Ephemeral",
								}, false),
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesCSIDriverCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	CSIDriver := api.CSIDriver{
		ObjectMeta: expandMetadata(d.Get("metadata").([]interface{})),
		Spec:       expandCSIDriverSpec(d.Get("spec").([]interface{})),
	}

	log.Printf("[INFO] Creating new CSIDriver: %#v", CSIDriver)
	out, err := conn.StorageV1beta1().CSIDrivers().Create(&CSIDriver)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new CSIDriver: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesCSIDriverRead(d, meta)
}

func resourceKubernetesCSIDriverRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	name := d.Id()
	log.Printf("[INFO] Reading CSIDriver %s", name)
	CSIDriver, err := conn.StorageV1beta1().CSIDrivers().Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received CSIDriver: %#v", CSIDriver)
	err = d.Set("metadata", flattenMetadata(CSIDriver.ObjectMeta, d))
	if err != nil {
		return err
	}

	spec, err := flattenCSIDriverSpec(CSIDriver.Spec)
	if err != nil {
		return err
	}

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesCSIDriverUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	name := d.Id()
	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		diffOps, err := patchCSIDriverSpec("spec.0.", "/spec", d)
		if err != nil {
			return err
		}
		ops = append(ops, *diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating CSIDriver %q: %v", name, string(data))
	out, err := conn.StorageV1beta1().CSIDrivers().Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update CSIDriver: %s", err)
	}
	log.Printf("[INFO] Submitted updated CSIDriver: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesCSIDriverRead(d, meta)
}

func resourceKubernetesCSIDriverDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	name := d.Id()
	log.Printf("[INFO] Deleting CSIDriver: %#v", name)
	err = conn.StorageV1beta1().CSIDrivers().Delete(name, &deleteOptions)
	if err != nil {
		return err
	}

	log.Printf("[INFO] CSIDriver %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesCSIDriverExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()
	log.Printf("[INFO] Checking CSIDriver %s", name)
	_, err = conn.StorageV1beta1().CSIDrivers().Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func expandCSIDriverSpec(l []interface{}) api.CSIDriverSpec {
	if len(l) == 0 || l[0] == nil {
		return api.CSIDriverSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := api.CSIDriverSpec{}

	if v, ok := in["attach_required"].(bool); ok {
		obj.AttachRequired = ptrToBool(v)
	}

	if v, ok := in["pod_info_on_mount"].(bool); ok {
		obj.PodInfoOnMount = ptrToBool(v)
	}

	if v, ok := in["volume_lifecycle_modes"].([]interface{}); ok && len(v) > 0 {
		obj.VolumeLifecycleModes = expandCSIDriverVolumeLifecycleModes(v)
	}

	return obj
}

func expandCSIDriverVolumeLifecycleModes(l []interface{}) []api.VolumeLifecycleMode {
	lifecycleModes := make([]api.VolumeLifecycleMode, 0, 0)
	for _, lifecycleMode := range l {
		lifecycleModes = append(lifecycleModes, api.VolumeLifecycleMode(lifecycleMode.(string)))
	}
	return lifecycleModes
}

func flattenCSIDriverSpec(in api.CSIDriverSpec) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["attach_required"] = in.AttachRequired

	if in.PodInfoOnMount != nil {
		att["pod_info_on_mount"] = in.PodInfoOnMount
	}

	if len(in.VolumeLifecycleModes) > 0 {
		att["volume_lifecycle_modes"] = in.VolumeLifecycleModes
	}

	return []interface{}{att}, nil
}

func patchCSIDriverSpec(keyPrefix, pathPrefix string, d *schema.ResourceData) (*PatchOperations, error) {
	ops := make(PatchOperations, 0, 0)
	if d.HasChange(keyPrefix + "attach_required") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/attachRequired",
			Value: d.Get(keyPrefix + "attach_required").(bool),
		})
	}

	if d.HasChange(keyPrefix + "pod_info_on_mount") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/podInfoOnMount",
			Value: d.Get(keyPrefix + "pod_info_on_mount").(bool),
		})
	}

	if d.HasChange(keyPrefix + "volume_lifecycle_modes") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/volumeLifecycleModes",
			Value: expandCSIDriverVolumeLifecycleModes(d.Get(keyPrefix + "volume_lifecycle_modes").([]interface{})),
		})
	}

	return &ops, nil
}
