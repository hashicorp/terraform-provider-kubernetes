// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	v1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

func resourceKubernetesAPIServiceV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesAPIServiceV1Create,
		ReadContext:   resourceKubernetesAPIServiceV1Read,
		UpdateContext: resourceKubernetesAPIServiceV1Update,
		DeleteContext: resourceKubernetesAPIServiceV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("api_service", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec contains information for locating and communicating with a server. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ca_bundle": {
							Type:        schema.TypeString,
							Description: "CABundle is a PEM encoded CA bundle which will be used to validate an API server's serving certificate. If unspecified, system trust roots on the apiserver are used.",
							Optional:    true,
						},
						"group": {
							Type:        schema.TypeString,
							Description: "Group is the API group name this server hosts.",
							Required:    true,
						},
						"group_priority_minimum": {
							Type:         schema.TypeInt,
							Description:  "GroupPriorityMinimum is the priority this group should have at least. Higher priority means that the group is preferred by clients over lower priority ones. Note that other versions of this group might specify even higher GroupPriorityMininum values such that the whole group gets a higher priority. The primary sort is based on GroupPriorityMinimum, ordered highest number to lowest (20 before 10). The secondary sort is based on the alphabetical comparison of the name of the object. (v1.bar before v1.foo) We'd recommend something like: *.k8s.io (except extensions) at 18000 and PaaSes (OpenShift, Deis) are recommended to be in the 2000s.",
							Required:     true,
							ValidateFunc: validation.IntBetween(0, 20000),
						},
						"insecure_skip_tls_verify": {
							Type:        schema.TypeBool,
							Description: "InsecureSkipTLSVerify disables TLS certificate verification when communicating with this server. This is strongly discouraged. You should use the CABundle instead.",
							Optional:    true,
							Default:     false,
						},
						"service": {
							Type:        schema.TypeList,
							Description: "Service is a reference to the service for this API server. It must communicate on port 443. If the Service is nil, that means the handling for the API groupversion is handled locally on this server. The call will simply delegate to the normal handler chain to be fulfilled.",
							Optional:    true,
							ForceNew:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "Name is the name of the service.",
										Required:    true,
									},
									"namespace": {
										Type:        schema.TypeString,
										Description: "Namespace is the namespace of the service.",
										Required:    true,
									},
									"port": {
										Type:         schema.TypeInt,
										Description:  "If specified, the port on the service that is hosting the service. Defaults to 443 for backward compatibility. Should be a valid port number (1-65535, inclusive).",
										Optional:     true,
										Default:      443,
										ValidateFunc: validatePortNum,
									},
								},
							},
						},
						"version": {
							Type:        schema.TypeString,
							Description: "Version is the API version this server hosts. For example, `v1`.",
							Required:    true,
						},
						"version_priority": {
							Type:         schema.TypeInt,
							Description:  "VersionPriority controls the ordering of this API version inside of its group. Must be greater than zero. The primary sort is based on VersionPriority, ordered highest to lowest (20 before 10). Since it's inside of a group, the number can be small, probably in the 10s. In case of equal version priorities, the version string will be used to compute the order inside a group. If the version string is `kube-like`, it will sort above non `kube-like` version strings, which are ordered lexicographically. `Kube-like` versions start with a `v`, then are followed by a number (the major version), then optionally the string `alpha` or `beta` and another number (the minor version). These are sorted first by GA > `beta` > `alpha` (where GA is a version with no suffix such as `beta` or `alpha`), and then by comparing major version, then minor version. An example sorted list of versions: `v10`, `v2`, `v1`, `v11beta2`, `v10beta3`, `v3beta1`, `v12alpha1`, `v11alpha2`, `foo1`, `foo10`.",
							Required:     true,
							ValidateFunc: validation.IntBetween(0, 1000),
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesAPIServiceV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).AggregatorClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	svc := v1.APIService{
		ObjectMeta: metadata,
		Spec:       expandAPIServiceV1Spec(d.Get("spec").([]interface{})),
	}

	log.Printf("[INFO] Creating new API service: %#v", svc)
	out, err := conn.ApiregistrationV1().APIServices().Create(ctx, &svc, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new API service: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesAPIServiceV1Read(ctx, d, meta)
}

func resourceKubernetesAPIServiceV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesAPIServiceV1Exists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	conn, err := meta.(KubeClientsets).AggregatorClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	log.Printf("[INFO] Reading service %s", name)
	svc, err := conn.ApiregistrationV1().APIServices().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received API service: %#v", svc)
	err = d.Set("metadata", flattenMetadata(svc.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := flattenAPIServiceV1Spec(svc.Spec)
	log.Printf("[DEBUG] Flattened API service spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesAPIServiceV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).AggregatorClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: expandAPIServiceV1Spec(d.Get("spec").([]interface{})),
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating service %q: %v", name, string(data))
	out, err := conn.ApiregistrationV1().APIServices().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update API service: %s", err)
	}
	log.Printf("[INFO] Submitted updated API service: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesAPIServiceV1Read(ctx, d, meta)
}

func resourceKubernetesAPIServiceV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).AggregatorClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()

	log.Printf("[INFO] Deleting API service: %#v", name)
	err = conn.ApiregistrationV1().APIServices().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] API service %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesAPIServiceV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).AggregatorClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()

	log.Printf("[INFO] Checking API service %s", name)
	_, err = conn.ApiregistrationV1().APIServices().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
