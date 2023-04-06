// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	// "fmt"
	"log"
	"regexp"
	// "time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	//"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	nodev1 "k8s.io/api/node/v1"
	// api "k8s.io/api/core/v1"
	// "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesRuntimeClassV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesRuntimeClassV1Create,
		ReadContext:   resourceKubernetesRuntimeClassV1Read,
		UpdateContext: resourceKubernetesRuntimeClassV1Update,
		DeleteContext: resourceKubernetesRuntimeClassV1Delete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("runtimeclass", true),
			// "overhead": {
			// 	Type:        schema.TypeMap,
			// 	Description: "Represents the esource overhead associated with running a pod for a given RuntimeClass",
			// },
			// "scheduling": {
			// 	Type:        schema.TypeMap,
			// 	Description: "Holds the scheduling constraints to ensure that pods running with this RuntimeClass are scheduled to nodes that support it",
			// },
			"handler": {
				Type:         schema.TypeString,
				Description:  "Specifies the underlying runtime and configuration that the CRI implementation will use to handle pods of this class",
				Required:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?`), ""),
			},
		},
	}
}

func resourceKubernetesRuntimeClassV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	//converting metadata for resource -> HCL for TF
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	runtimeClass := nodev1.RuntimeClass{
		ObjectMeta: metadata,
		Handler:    d.Get("handler").(string), //returns the string from handler
	}

	out, err := conn.NodeV1().RuntimeClasses().Create(ctx, &runtimeClass, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] New runtime class created: %#v", out)
	//d.SetID(out.Name)

	return nil
}

func resourceKubernetesRuntimeClassV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// if err != nil {
	// 	return diag.FromErr(err)
	// }
	// if !exists {
	// 	d.SetId("")
	// 	return diags
	// }
	// conn, err := meta.(KubeClientsets).MainClientset()
	// if err != nil {
	// 	return diag.FromErr(err)
	// }
	return nil
}

func resourceKubernetesRuntimeClassV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// conn, err := meta.(KubeClientsets).MainClientset()
	// if err != nil {
	// 	return diag.FromErr(err)
	// }
	// diags := diag.Diagnostics{}

	// exists, err := resourceKubernetesStorageClassExists(ctx, d, meta)

	return nil
}

func resourceKubernetesRuntimeClassV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}
