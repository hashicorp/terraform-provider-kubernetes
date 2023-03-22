// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/core/helper"
	"k8s.io/kubernetes/pkg/util/taints"
)

var taintMap = map[string]v1.TaintEffect{
	"NoExecute":        v1.TaintEffectNoExecute,
	"NoSchedule":       v1.TaintEffectNoSchedule,
	"PreferNoSchedule": v1.TaintEffectPreferNoSchedule,
}

func resourceKubernetesNodeTaint() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesNodeTaintCreate,
		ReadContext:   resourceKubernetesNodeTaintRead,
		UpdateContext: resourceKubernetesNodeTaintUpdate,
		DeleteContext: resourceKubernetesNodeTaintDelete,
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
	d.SetId(fmt.Sprintf("%s-taints", &metadata.Name))
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
	nodeName, idTaint, err := idToNodeTaint(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	conn, err := m.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}
	nodeApi := conn.CoreV1().Nodes()

	node, err := nodeApi.Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	nodeTaints := node.Spec.Taints
	if len(nodeTaints) == 0 {
		d.SetId("")
		return nil
	}
	if !hasTaint(nodeTaints, idTaint) {
		d.SetId("")
		return nil
	}
	d.Set("taint", flattenNodeTaints(*idTaint))
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

	node, err := nodeApi.Get(ctx, nodeName, metav1.GetOptions{})
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
	newTaint, err := expandNodeTaint(taints[0].(map[string]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	var newNode *v1.Node
	if d.Id() == "" {
		var removed bool
		newNode, removed = removeTaint(node, newTaint)
		if !removed {
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "Resource deleted",
				Detail:   fmt.Sprintf("Node %s does not have taint %+v. You should re-create it, or remove this resource from your configuration", nodeName, newTaint),
			}}
		}
	} else {
		log.Printf("[INFO] adding taint %+v to node %s", newTaint, nodeName)
		var updated bool
		newNode, updated = addOrUpdateTaint(node, newTaint)
		if !updated {
			return diag.Errorf("Node %s already has taint %+v", nodeName, newTaint)
		}
	}
	patchObj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Node",
		"metadata": map[string]interface{}{
			"name": nodeName,
		},
		"spec": map[string]interface{}{
			"taints": flattenNodeTaints(newNode.Spec.Taints...),
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
		Force:        ptrToBool(d.Get("force").(bool)),
	}
	if _, err := nodeApi.Patch(ctx, nodeName, types.ApplyPatchType, patchBytes, patchOpts); err != nil {
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
	return resourceKubernetesNodeTaintRead(ctx, d, m)
}

func addOrUpdateTaint(node *v1.Node, taint *v1.Taint) (*v1.Node, bool) {
	nodeTaints := node.Spec.Taints
	newTaints := []v1.Taint{}
	updated := false
	for i := range nodeTaints {
		log.Printf("[INFO] Checking taint: %+v", nodeTaints[i])
		if taint.MatchTaint(&nodeTaints[i]) {
			if helper.Semantic.DeepEqual(*taint, nodeTaints[i]) {
				return node, false
			}
			newTaints = append(newTaints, *taint)
			updated = true
			continue
		}
		newTaints = append(newTaints, nodeTaints[i])
	}
	if !updated {
		newTaints = append(newTaints, *taint)
		log.Printf("[INFO] appended taint: %+v", taint)
		updated = true
	}
	newNode := node.DeepCopy()
	newNode.Spec.Taints = newTaints
	return newNode, updated
}

func removeTaint(node *v1.Node, delTaint *v1.Taint) (*v1.Node, bool) {
	taints := node.Spec.Taints
	newTaints := []v1.Taint{}
	deleted := false
	for i := range taints {
		if delTaint.MatchTaint(&taints[i]) {
			deleted = true
			continue
		}
		newTaints = append(newTaints, taints[i])
	}
	if !deleted {
		return node, false
	}
	newNode := node.DeepCopy()
	newNode.Spec.Taints = newTaints
	return newNode, deleted
}

func hasTaint(taints []v1.Taint, taint *v1.Taint) bool {
	for i := range taints {
		if taint.MatchTaint(&taints[i]) {
			return true
		}
	}
	return false
}

func expandNodeTaint(t map[string]interface{}) (*v1.Taint, error) {
	tt := expandStringMap(t)
	taintEffect, ok := taintMap[tt["effect"]]
	if !ok {
		return nil, fmt.Errorf("Invalid taint effect '%s'", tt["effect"])
	}
	taint := &v1.Taint{
		Key:    tt["key"],
		Value:  tt["value"],
		Effect: taintEffect,
	}
	return taint, nil
}

func idToNodeTaint(id string) (string, *v1.Taint, error) {
	idVals := strings.Split(id, ",")
	nodeName := idVals[0]
	taintStr := idVals[1]
	taints, _, err := taints.ParseTaints([]string{taintStr})
	if err != nil {
		return "", nil, err
	}
	if len(taints) == 0 {
		return "", nil, fmt.Errorf("failed to parse taint %s", taintStr)
	}
	return nodeName, &taints[0], nil
}
