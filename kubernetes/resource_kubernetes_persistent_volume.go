package kubernetes

import (
	"fmt"
	"log"
	"time"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
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
			if diff.Id() == "" {
				// We only care about updates, not creation
				return nil
			}

			// Mutation of PersistentVolumeSource after creation is no longer allowed in 1.9+
			// See https://github.com/kubernetes/kubernetes/blob/v1.9.3/CHANGELOG-1.9.md#storage-3
			conn, err := meta.(KubeClientsets).MainClientset()
			if err != nil {
				return err
			}
			serverVersion, err := conn.ServerVersion()
			if err != nil {
				return err
			}

			k8sVersion, err := gversion.NewVersion(serverVersion.String())
			if err != nil {
				return err
			}

			v1_9_0, _ := gversion.NewVersion("1.9.0")
			if k8sVersion.Equal(v1_9_0) || k8sVersion.GreaterThan(v1_9_0) {
				if diff.HasChange("spec.0.persistent_volume_source") {
					keys := diff.GetChangedKeysPrefix("spec.0.persistent_volume_source")
					for _, key := range keys {
						if diff.HasChange(key) {
							err := diff.ForceNew(key)
							if err != nil {
								return err
							}
						}
					}
					return nil
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
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
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
	out, err := conn.CoreV1().PersistentVolumes().Create(&volume)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new persistent volume: %#v", out)

	stateConf := &resource.StateChangeConf{
		Target:  []string{"Available", "Bound"},
		Pending: []string{"Pending"},
		Timeout: d.Timeout(schema.TimeoutCreate),
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().PersistentVolumes().Get(metadata.Name, meta_v1.GetOptions{})
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

	name := d.Id()
	log.Printf("[INFO] Reading persistent volume %s", name)
	volume, err := conn.CoreV1().PersistentVolumes().Get(name, meta_v1.GetOptions{})
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
	out, err := conn.CoreV1().PersistentVolumes().Patch(d.Id(), pkgApi.JSONPatchType, data)
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

	name := d.Id()
	log.Printf("[INFO] Deleting persistent volume: %#v", name)
	err = conn.CoreV1().PersistentVolumes().Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		out, err := conn.CoreV1().PersistentVolumes().Get(name, meta_v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
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

	name := d.Id()
	log.Printf("[INFO] Checking persistent volume %s", name)
	_, err = conn.CoreV1().PersistentVolumes().Get(name, meta_v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
