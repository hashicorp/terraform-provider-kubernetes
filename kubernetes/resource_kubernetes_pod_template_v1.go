package kubernetes

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesPodTemplateV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesPodTemplateV1Create,
		ReadContext:   resourceKubernetesPodTemplateV1Read,
		UpdateContext: resourceKubernetesPodTemplateV1Update,
		DeleteContext: resourceKubernetesPodTemplateV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Schema: resourceKubernetesPodTemplateSchemaV1(),
	}
}

func resourceKubernetesPodTemplateSchemaV1() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("podTemplate", true),
		"template": {
			Type:        schema.TypeList,
			Description: "Specification of the desired behavior of the pod template.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"metadata": metadataSchema("podTemplate", true),
					"spec": {
						Type:        schema.TypeList,
						Description: "Specification of the desired behavior of the pod.",
						Required:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: podSpecFields(false, false),
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesPodTemplateV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	template, err := expandPodTemplate(d.Get("template").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	pod := corev1.PodTemplate{
		ObjectMeta: metadata,
		Template:   *template,
	}

	log.Printf("[INFO] Creating new pod template: %#v", pod)
	out, err := conn.CoreV1().PodTemplates(metadata.Namespace).Create(ctx, &pod, metav1.CreateOptions{})

	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new pod template: %#v", out)

	d.SetId(buildId(out.ObjectMeta))

	_, err = conn.CoreV1().PodTemplates(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[ERROR] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Pod template %s created", out.Name)

	return resourceKubernetesPodTemplateV1Read(ctx, d, meta)
}

func resourceKubernetesPodTemplateV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("template") {
		specOps, err := patchPodSpec("/template", "template.0.", d)
		if err != nil {
			return diag.FromErr(err)
		}
		ops = append(ops, specOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating pod template %s: %s", d.Id(), ops)

	out, err := conn.CoreV1().PodTemplates(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted updated pod template: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	return resourceKubernetesPodTemplateV1Read(ctx, d, meta)
}

func resourceKubernetesPodTemplateV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesPodTemplateV1Exists(ctx, d, meta)
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

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading pod template %s", name)
	podTemplate, err := conn.CoreV1().PodTemplates(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received pod template: %#v", podTemplate)

	err = d.Set("metadata", flattenMetadata(podTemplate.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	podSpec, err := flattenPodTemplateSpec(podTemplate.Template, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("template", podSpec)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil

}

func resourceKubernetesPodTemplateV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting pod template: %#v", name)
	err = conn.CoreV1().PodTemplates(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = conn.CoreV1().PodTemplates(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Pod template %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPodTemplateV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking pod template %s", name)
	_, err = conn.CoreV1().PodTemplates(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
