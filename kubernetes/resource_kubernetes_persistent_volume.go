package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

const (
	persistentVolumeAzureManagedError = `Unable to apply Azure Disk configuration. Managed disks require configuration: kind = "Managed"`
	persistentVolumeAzureBlobError    = `Unable to apply Azure Disk configuration. Blob storage disks require configuration: kind = "Shared" or kind = "Dedicated"`
)

func resourceKubernetesPersistentVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPersistentVolumeCreate,
		Read:   resourceKubernetesPersistentVolumeRead,
		Exists: resourceKubernetesPersistentVolumeExists,
		Update: resourceKubernetesPersistentVolumeUpdate,
		Delete: resourceKubernetesPersistentVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
		},

		CustomizeDiff: func(diff *schema.ResourceDiff, meta interface{}) error {
			// The field `data_disk_uri` expects a different value depending on the value of `kind`.
			// If `kind` is omitted, "Shared", or "Dedicated", then data_disk_uri expects a blob storage disk.
			// If `kind` is "Managed", then `data_disk_uri` expects a Managed Disk.
			kind := "spec.0.persistent_volume_source.0.azure_disk.0.kind"
			diskURI := "spec.0.persistent_volume_source.0.azure_disk.0.data_disk_uri"
			kindValue, _ := diff.GetOk(kind)
			diskURIValue, diskURIExists := diff.GetOk(diskURI)
			if diskURIExists && strings.Contains(diskURIValue.(string), "blob.core.windows.net") && kindValue == "Managed" {
				log.Printf("Configuration error:")
				log.Printf("Mismatch between Disk URI: %v = %v and Disk Kind: %v = %v", diskURI, diskURIValue, kind, kindValue)
				return errors.New(persistentVolumeAzureBlobError)
			}
			if diskURIExists && strings.Contains(diskURIValue.(string), "/providers/Microsoft.Compute/disks/") && kindValue != "Managed" {
				log.Printf("Configuration error:")
				log.Printf("Mismatch between Disk URI: %v = %v and disk Kind: %v = %v", diskURI, diskURIValue, kind, kindValue)
				return errors.New(persistentVolumeAzureManagedError)
			}
			// The following applies to Updates only.
			if diff.Id() == "" {
				return nil
			}
			// Any change to Persistent Volume Source requires a new resource.
			keys := diff.GetChangedKeysPrefix("spec.0.persistent_volume_source")
			for _, key := range keys {
				log.Printf("[DEBUG] CustomizeDiff GetChangedKeysPrefix key: %v", key)
				log.Printf("[DEBUG] CustomizeDiff key: %v", key)
				err := diff.ForceNew(key)
				if err != nil {
					return err
				}
			}
			return nil
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("persistent volume", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec of the persistent volume owned by the cluster",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_modes": {
							Type:        schema.TypeSet,
							Description: "Contains all ways the volume can be mounted. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#access-modes",
							Required:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"ReadWriteOnce",
									"ReadOnlyMany",
									"ReadWriteMany",
								}, false),
							},
							Set: schema.HashString,
						},
						"capacity": {
							Type:         schema.TypeMap,
							Description:  "A description of the persistent volume's resources and capacity. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#capacity",
							Required:     true,
							Elem:         schema.TypeString,
							ValidateFunc: validateResourceList,
						},
						"persistent_volume_reclaim_policy": {
							Type:        schema.TypeString,
							Description: "What happens to a persistent volume when released from its claim. Valid options are Retain (default) and Recycle. Recycling must be supported by the volume plugin underlying this persistent volume. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#recycling-policy",
							Optional:    true,
							Default:     "Retain",
							ValidateFunc: validation.StringInSlice([]string{
								"Recycle",
								"Delete",
								"Retain",
							}, false),
						},
						"persistent_volume_source": {
							Type:        schema.TypeList,
							Description: "The specification of a persistent volume.",
							Required:    true,
							MaxItems:    1,
							Elem:        persistentVolumeSourceSchema(),
						},
						"storage_class_name": {
							Type:        schema.TypeString,
							Description: "A description of the persistent volume's class. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#class",
							Optional:    true,
						},
						"node_affinity": {
							Type:        schema.TypeList,
							Description: "A description of the persistent volume's node affinity. More info: https://kubernetes.io/docs/concepts/storage/volumes/#local",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"required": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"node_selector_term": {
													Type:     schema.TypeList,
													Required: true,
													Elem: &schema.Resource{
														Schema: nodeSelectorTermFields(),
													},
												},
											},
										},
									},
								},
							},
						},
						"mount_options": {
							Type:        schema.TypeSet,
							Description: "A list of mount options, e.g. [\"ro\", \"soft\"]. Not validated - mount will simply fail if one is invalid.",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"volume_mode": {
							Type:        schema.TypeString,
							Description: "Defines if a volume is intended to be used with a formatted filesystem. or to remain in raw block state.",
							Optional:    true,
							ForceNew:    true,
							Default:     "Filesystem",
							ValidateFunc: validation.StringInSlice([]string{
								"Block",
								"Filesystem",
							}, false),
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesPersistentVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPersistentVolumeSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}
	volume := api.PersistentVolume{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new persistent volume: %#v", volume)
	out, err := conn.CoreV1().PersistentVolumes().Create(ctx, &volume, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new persistent volume: %#v", out)

	stateConf := &resource.StateChangeConf{
		Target:  []string{"Available", "Bound"},
		Pending: []string{"Pending"},
		Timeout: d.Timeout(schema.TimeoutCreate),
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().PersistentVolumes().Get(ctx, metadata.Name, metav1.GetOptions{})
			if err != nil {
				log.Printf("[ERROR] Received error: %#v", err)
				return out, "Error", err
			}

			statusPhase := fmt.Sprintf("%v", out.Status.Phase)
			log.Printf("[DEBUG] Persistent volume %s status received: %#v", out.Name, statusPhase)
			return out, statusPhase, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}
	log.Printf("[INFO] Persistent volume %s created", out.Name)

	d.SetId(out.Name)

	return resourceKubernetesPersistentVolumeRead(d, meta)
}

func resourceKubernetesPersistentVolumeRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Reading persistent volume %s", name)
	volume, err := conn.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received persistent volume: %#v", volume)
	err = d.Set("metadata", flattenMetadata(volume.ObjectMeta, d))
	if err != nil {
		return err
	}
	err = d.Set("spec", flattenPersistentVolumeSpec(volume.Spec))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesPersistentVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		specOps, err := patchPersistentVolumeSpec("/spec", "spec", d)
		if err != nil {
			return err
		}
		ops = append(ops, specOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating persistent volume %s: %s", d.Id(), ops)
	out, err := conn.CoreV1().PersistentVolumes().Patch(ctx, d.Id(), pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated persistent volume: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesPersistentVolumeRead(d, meta)
}

func resourceKubernetesPersistentVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Deleting persistent volume: %#v", name)
	err = conn.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		out, err := conn.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		log.Printf("[DEBUG] Current state of persistent volume: %#v", out.Status.Phase)
		e := fmt.Errorf("Persistent volume %s still exists (%s)", name, out.Status.Phase)
		return resource.RetryableError(e)
	})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Persistent volume %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPersistentVolumeExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Checking persistent volume %s", name)
	_, err = conn.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
