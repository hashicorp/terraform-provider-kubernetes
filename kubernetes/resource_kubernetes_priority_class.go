package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesPriorityClass() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesPriorityClassCreate,
		ReadContext:   resourceKubernetesPriorityClassRead,
		UpdateContext: resourceKubernetesPriorityClassUpdate,
		DeleteContext: resourceKubernetesPriorityClassDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("priority class", true),
			"description": {
				Type:        schema.TypeString,
				Description: "An arbitrary string that usually provides guidelines on when this priority class should be used.",
				Optional:    true,
				Default:     "",
			},
			"global_default": {
				Type:        schema.TypeBool,
				Description: "Specifies whether this PriorityClass should be considered as the default priority for pods that do not have any priority class. Only one PriorityClass can be marked as `globalDefault`. However, if more than one PriorityClasses exists with their `globalDefault` field set to true, the smallest value of such global default PriorityClasses will be used as the default priority.",
				Optional:    true,
				Default:     false,
			},
			"value": {
				Type:        schema.TypeInt,
				Description: "The value of this priority class. This is the actual priority that pods receive when they have the name of this class in their pod spec.",
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceKubernetesPriorityClassCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	value := d.Get("value").(int)
	description := d.Get("description").(string)
	globalDefault := d.Get("global_default").(bool)

	priorityClass := api.PriorityClass{
		ObjectMeta:    metadata,
		Description:   description,
		GlobalDefault: globalDefault,
		Value:         int32(value),
	}

	log.Printf("[INFO] Creating new priority class: %#v", priorityClass)
	out, err := conn.SchedulingV1().PriorityClasses().Create(ctx, &priorityClass, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create priority class: %s", err)
	}
	log.Printf("[INFO] Submitted new priority class: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesPriorityClassRead(ctx, d, meta)
}

func resourceKubernetesPriorityClassRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesPriorityClassExists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	log.Printf("[INFO] Reading priority class %s", name)
	priorityClass, err := conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received priority class: %#v", priorityClass)

	err = d.Set("metadata", flattenMetadata(priorityClass.ObjectMeta, d))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("value", priorityClass.Value)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("description", priorityClass.Description)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("global_default", priorityClass.GlobalDefault)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesPriorityClassUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("description") {
		description := d.Get("description").(string)
		ops = append(ops, &ReplaceOperation{
			Path:  "/description",
			Value: description,
		})
	}

	if d.HasChange("global_default") {
		globalDefault := d.Get("global_default").(bool)
		ops = append(ops, &ReplaceOperation{
			Path:  "/globalDefault",
			Value: globalDefault,
		})
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating priority class %q: %v", name, string(data))
	out, err := conn.SchedulingV1().PriorityClasses().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update priority class: %s", err)
	}
	log.Printf("[INFO] Submitted updated priority class: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesPriorityClassRead(ctx, d, meta)
}

func resourceKubernetesPriorityClassDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	log.Printf("[INFO] Deleting priority class: %#v", name)
	err = conn.SchedulingV1().PriorityClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] priority class %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPriorityClassExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()

	log.Printf("[INFO] Checking priority class %s", name)
	_, err = conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
