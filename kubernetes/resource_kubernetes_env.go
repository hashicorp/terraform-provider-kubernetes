// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-kubernetes/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

func resourceKubernetesEnv() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesEnvCreate,
		ReadContext:   resourceKubernetesEnvRead,
		UpdateContext: resourceKubernetesEnvUpdate,
		DeleteContext: resourceKubernetesEnvDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
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
							Description: "The name of the resource.",
							Required:    true,
							ForceNew:    true,
						},
						"namespace": {
							Type:        schema.TypeString,
							Description: "The namespace of the resource.",
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},
			"container": {
				Type:         schema.TypeString,
				Description:  "Name of the container for which we are updating the environment variables.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ExactlyOneOf: []string{"container", "init_container"},
			},
			"init_container": {
				Type:         schema.TypeString,
				Description:  "Name of the initContainer for which we are updating the environment variables.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				ExactlyOneOf: []string{"container", "init_container"},
			},
			"api_version": {
				Type:        schema.TypeString,
				Description: "Resource API version",
				Required:    true,
			},
			"kind": {
				Type:         schema.TypeString,
				Description:  "Resource Kind",
				ValidateFunc: validation.StringInSlice([]string{"CronJob", "Deployment", "Pod", "DaemonSet", "replicationcontroller", "StatefulSet", "ReplicaSet"}, true),
				Required:     true,
			},
			"env": {
				Type:        schema.TypeList,
				Description: "List of custom values used to represent environment variables",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the environment variable. Must be a C_IDENTIFIER",
						},
						"value": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: `Variable references $(VAR_NAME) are expanded using the previous defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".`,
						},
						"value_from": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Source for the environment variable's value",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"config_map_key_ref": {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "Selects a key of a ConfigMap.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "The key to select.",
												},
												"name": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
												},
												"optional": {
													Type:        schema.TypeBool,
													Optional:    true,
													Description: "Specify whether the ConfigMap or its key must be defined.",
												},
											},
										},
									},
									"field_ref": {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"api_version": {
													Type:        schema.TypeString,
													Optional:    true,
													Default:     "v1",
													Description: `Version of the schema the FieldPath is written in terms of, defaults to "v1".`,
												},
												"field_path": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "Path of the field to select in the specified API version",
												},
											},
										},
									},
									"resource_field_ref": {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"container_name": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"divisor": {
													Type:             schema.TypeString,
													Optional:         true,
													Default:          "1",
													ValidateFunc:     validateResourceQuantity,
													DiffSuppressFunc: suppressEquivalentResourceQuantity,
												},
												"resource": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Resource to select",
												},
											},
										},
									},
									"secret_key_ref": {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "Selects a key of a secret in the pod's namespace.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "The key of the secret to select from. Must be a valid secret key.",
												},
												"name": {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
												},
												"optional": {
													Type:        schema.TypeBool,
													Optional:    true,
													Description: "Specify whether the Secret or its key must be defined.",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"force": {
				Type:        schema.TypeBool,
				Description: "Force overwriting environments that were created or edited outside of Terraform.",
				Optional:    true,
			},
			"field_manager": {
				Type:        schema.TypeString,
				Description: "Set the name of the field manager for the specified environment variables.",
				Optional:    true,
				Default:     defaultFieldManagerName,
			},
		},
	}
}

func resourceKubernetesEnvCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	d.SetId(buildIdWithVersionKind(metadata,
		d.Get("api_version").(string),
		d.Get("kind").(string)))
	diag := resourceKubernetesEnvUpdate(ctx, d, m)
	if diag.HasError() {
		d.SetId("")
	}
	return diag
}

func resourceKubernetesEnvRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).DynamicClient()
	if err != nil {
		return diag.FromErr(err)
	}

	gvk, name, namespace, err := util.ParseResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// figure out which resource client to use
	dc, err := m.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	if err != nil {
		return diag.FromErr(err)
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return diag.FromErr(err)
	}

	// determine if the resource is namespaced or not
	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace == "" {
			namespace = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(namespace)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	// get the resource environments
	res, err := r.Get(ctx, name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "Resource deleted",
				Detail:   fmt.Sprintf("The underlying resource %q has been deleted. You should recreate the underlying resource, or remove it from your configuration.", name),
			}}
		}
		return diag.FromErr(err)
	}

	// store names of environment variables into map
	configuredEnvs := make(map[string]interface{})
	envList := d.Get("env").([]interface{})
	for _, e := range envList {
		configuredEnvs[e.(map[string]interface{})["name"].(string)] = ""
	}

	var container string
	if c := d.Get("container").(string); c != "" {
		container = c
	} else {
		container = d.Get("init_container").(string)
	}

	// strip out envs not managed by Terraform
	fieldManagerName := d.Get("field_manager").(string)
	managedEnvs, err := getManagedEnvs(res.GetManagedFields(), fieldManagerName, d, res)
	if err != nil {
		return diag.FromErr(err)
	}

	responseEnvs, err := getResponseEnvs(res, container, d.Get("kind").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	env := []interface{}{}
	for _, e := range responseEnvs {
		envName := e.(map[string]interface{})["name"].(string)
		_, managed := managedEnvs[fmt.Sprintf(`k:{"name":%q}`, envName)]
		_, configured := configuredEnvs[envName]
		if !managed && !configured {
			continue
		}
		env = append(env, e)
	}

	env = flattenEnv(env)
	d.Set("env", env)
	return nil
}

func getResponseEnvs(u *unstructured.Unstructured, containerName string, kind string) ([]interface{}, error) {
	var containers []interface{}
	var initContainers []interface{}

	initContainers, _, _ = unstructured.NestedSlice(u.Object, "spec", "template", "spec", "initContainers")
	if kind == "CronJob" {
		initContainers, _, _ = unstructured.NestedSlice(u.Object, "spec", "jobTemplate", "spec", "template", "spec", "initContainers")
	}

	containers, _, _ = unstructured.NestedSlice(u.Object, "spec", "template", "spec", "containers")
	if kind == "CronJob" {
		containers, _, _ = unstructured.NestedSlice(u.Object, "spec", "jobTemplate", "spec", "template", "spec", "containers")
	}

	containers = append(containers, initContainers...)

	for _, c := range containers {
		container := c.(map[string]interface{})
		if container["name"].(string) == containerName {
			return container["env"].([]interface{}), nil
		}
	}
	return nil, fmt.Errorf("could not find container with name %q", containerName)
}

// getManagedEnvs reads the field manager metadata to discover which environment variables we're managing
func getManagedEnvs(managedFields []v1.ManagedFieldsEntry, manager string, d *schema.ResourceData, u *unstructured.Unstructured) (map[string]interface{}, error) {
	var envs map[string]interface{}
	kind := d.Get("kind").(string)
	for _, m := range managedFields {
		if m.Manager != manager {
			continue
		}
		var mm map[string]interface{}
		err := json.Unmarshal(m.FieldsV1.Raw, &mm)
		if err != nil {
			return nil, err
		}

		spec, _, err := unstructured.NestedMap(u.Object, "f:spec", "f:template", "f:spec")
		if kind == "CronJob" {
			spec, _, err = unstructured.NestedMap(u.Object, "f:spec", "f:jobTemplate", "f:spec", "f:template", "f:spec")
		}
		if err == nil {
			return nil, err
		}

		fieldManagerKey := "f:containers"
		containerName := d.Get("container").(string)
		if v := d.Get("init_container").(string); v != "" {
			containerName = v
			fieldManagerKey = "f:initContainers"
		}
		containers := spec[fieldManagerKey].(map[string]interface{})
		containerKey := fmt.Sprintf(`k:{"name":%q}`, containerName)
		k := containers[containerKey].(map[string]interface{})
		if e, ok := k["f:env"].(map[string]interface{}); ok {
			envs = e
		}
	}
	return envs, nil
}

func resourceKubernetesEnvUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).DynamicClient()
	if err != nil {
		return diag.FromErr(err)
	}

	apiVersion := d.Get("api_version").(string)
	kind := d.Get("kind").(string)
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.GetName()
	namespace := metadata.GetNamespace()

	// figure out which resource client to use
	dc, err := m.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	if err != nil {
		return diag.FromErr(err)
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	gv, err := k8sschema.ParseGroupVersion(apiVersion)
	if err != nil {
		return diag.FromErr(err)
	}
	mapping, err := restMapper.RESTMapping(gv.WithKind(kind).GroupKind(), gv.Version)
	if err != nil {
		return diag.FromErr(err)
	}

	// determine if the resource is namespaced or not
	var r dynamic.ResourceInterface
	namespacedResource := mapping.Scope.Name() == meta.RESTScopeNameNamespace
	if namespacedResource {
		if namespace == "" {
			namespace = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(namespace)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	// check the resource exists before we try and patch it
	_, err = r.Get(ctx, name, v1.GetOptions{})
	if err != nil {
		if d.Id() == "" {
			// if we are deleting then there is nothing to do
			// if the resource is gone
			return nil
		}
		return diag.Errorf("The resource %q does not exist", name)
	}

	patchmeta := map[string]interface{}{
		"name": name,
	}
	if namespacedResource {
		patchmeta["namespace"] = namespace
	}

	env := d.Get("env")
	env = expandEnv(env.([]interface{}))
	if d.Id() == "" {
		env = []map[string]interface{}{}
	}

	containersField := "containers"
	containerName := d.Get("container")
	if v := d.Get("init_container").(string); v != "" {
		containersField = "initContainers"
		containerName = v
	}

	containerSpec := map[string]interface{}{
		"name": containerName,
		"env":  env,
	}

	spec := map[string]interface{}{
		"template": map[string]interface{}{
			"spec": map[string]interface{}{
				containersField: []interface{}{
					containerSpec,
				},
			},
		},
	}

	if kind == "CronJob" {
		// CronJob nests under an additional jobTemplate field
		spec = map[string]interface{}{
			"jobTemplate": map[string]interface{}{
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							containersField: []interface{}{
								containerSpec,
							},
						},
					},
				},
			},
		}
	}

	patchObj := map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   patchmeta,
		"spec":       spec,
	}
	patch := unstructured.Unstructured{}
	patch.Object = patchObj
	patchbytes, err := patch.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = r.Patch(ctx,
		name,
		types.ApplyPatchType,
		patchbytes,
		v1.PatchOptions{
			FieldManager: d.Get("field_manager").(string),
			Force:        ptr.To(d.Get("force").(bool)),
		},
	)
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

	if d.Id() == "" {
		// don't try to read if we're deleting
		return nil
	}

	return resourceKubernetesEnvRead(ctx, d, m)
}

func resourceKubernetesEnvDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return resourceKubernetesEnvUpdate(ctx, d, m)
}
