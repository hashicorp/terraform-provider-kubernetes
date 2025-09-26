// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func resourceKubernetesNodeTaint() *schema.Resource {
	return &schema.Resource{
		Description:   "[Node affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) is a property of Pods that attracts them to a set of [nodes](https://kubernetes.io/docs/concepts/architecture/nodes/) (either as a preference or a hard requirement). Taints are the opposite -- they allow a node to repel a set of pods.",
		CreateContext: resourceKubernetesNodeTaintCreate,
		ReadContext:   resourceKubernetesNodeTaintRead,
		UpdateContext: resourceKubernetesNodeTaintUpdate,
		DeleteContext: resourceKubernetesNodeTaintDelete,
		CustomizeDiff: func(ctx context.Context, rd *schema.ResourceDiff, i interface{}) error {
			if !rd.HasChange("taint") {
				return nil
			}
			// check for duplicate taint keys
			taintkeys := map[string]int{}
			for _, t := range rd.Get("taint").([]interface{}) {
				taint := t.(map[string]interface{})
				key := taint["key"].(string)
				effect := taint["effect"].(string)
				taintkey := fmt.Sprintf("%s:%s", key, effect)
				taintkeys[taintkey] = taintkeys[taintkey] + 1
			}
			for k, v := range taintkeys {
				if v > 1 {
					s := strings.Split(k, ":")
					return fmt.Errorf("taint: duplicate taint for key %q and effect %q; taints must be unique over key and effect.", s[0], s[1])
				}
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"metadata": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the node",
							Required:    true,
							ForceNew:    true,
						},
					},
				},
			},
			"field_manager": {
				Type:         schema.TypeString,
				Description:  "Set the name of the field manager for the node taint",
				Optional:     true,
				Default:      defaultFieldManagerName,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"force": {
				Type:        schema.TypeBool,
				Description: "Force overwriting annotations that were created or edited outside of Terraform.",
				Optional:    true,
			},
			"taint": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: nodeTaintFields(),
				},
			},
		},
	}
}

func resourceKubernetesNodeTaintCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	d.SetId(nodeTaintToId(metadata.Name, d.Get("taint").([]interface{})))
	diag := resourceKubernetesNodeTaintUpdate(ctx, d, m)
	if diag.HasError() {
		d.SetId("")
	}
	return diag
}

func resourceKubernetesNodeTaintDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return resourceKubernetesNodeTaintUpdate(ctx, d, m)
}

func resourceKubernetesNodeTaintRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := expandMetadata(d.Get("metadata").([]interface{}))
	nodeName := meta.Name

	conn, err := m.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}
	nodeApi := conn.CoreV1().Nodes()

	node, err := nodeApi.Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			// The node is gone so the resource should be deleted.
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "Node has been deleted",
				Detail:   fmt.Sprintf("The underlying node %q has been deleted. You should remove it from your configuration.", nodeName),
			}}
		}
		return diag.FromErr(err)
	}
	nodeTaints := node.Spec.Taints
	if len(nodeTaints) == 0 {
		d.SetId("")
		return nil
	}

	d.Set("taint", flattenNodeTaints(nodeTaints...))
	return nil
}

func resourceKubernetesNodeTaintUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := expandMetadata(d.Get("metadata").([]interface{}))
	nodeName := meta.Name

	conn, err := m.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}
	nodeApi := conn.CoreV1().Nodes()

	_, err = nodeApi.Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if d.Id() == "" {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				// The node is gone. it is ok to remove the taint resource
				return nil
			}
		}
		return diag.FromErr(err)
	}

	taints := d.Get("taint").([]interface{})
	if d.Id() == "" {
		// make taints an empty list if we're deleting the resource
		taints = []interface{}{}
	}
	patchObj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Node",
		"metadata": map[string]interface{}{
			"name": nodeName,
		},
		"spec": map[string]interface{}{
			"taints": taints,
		},
	}

	patch := unstructured.Unstructured{
		Object: patchObj,
	}
	patchBytes, err := patch.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}
	patchOpts := metav1.PatchOptions{
		FieldManager: d.Get("field_manager").(string),
		Force:        ptr.To(d.Get("force").(bool)),
	}
	node, err := nodeApi.Patch(ctx, nodeName, types.ApplyPatchType, patchBytes, patchOpts)
	if err != nil {
		if errors.IsConflict(err) {
			return diag.Diagnostics{{
				Severity: diag.Error,
				Summary:  "Field manager conflict",
				Detail:   fmt.Sprintf(`Another client is managing a field Terraform tried to update. Set "force" to true to override: %v`, err),
			}}
		}
		return diag.FromErr(err)
	}
	// Don't update id or read if deleting
	if d.Id() == "" {
		return nil
	}

	d.SetId(nodeTaintToId(node.Name, d.Get("taint").([]interface{})))
	return resourceKubernetesNodeTaintRead(ctx, d, m)
}

func nodeTaintToId(id string, taints []interface{}) string {
	for _, t := range taints {
		taint := t.(map[string]interface{})
		id += fmt.Sprintf(",%s=%s:%s", taint["key"], taint["value"], taint["effect"])
	}
	return id
}
