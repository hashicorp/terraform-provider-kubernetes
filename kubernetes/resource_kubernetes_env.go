package kubernetes

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

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
				Type:        schema.TypeString,
				Description: "",
				Required:    true,
			},
			"api_version": {
				Type:        schema.TypeString,
				Description: "API Version of Field Manager",
				Required:    true,
			},
			"kind": {
				Type:        schema.TypeString,
				Description: "Type of resource being used",
				Required:    true,
			},
			"env": {
				Type:        schema.TypeList,
				Description: "Rule defining a set of permissions for the role",
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
					},
				},
			},
			// TODO: Add 'force' to schema
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
	return nil
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

	// Apply the Patch
	/*
		---
		apiVersion: apps/v1
		kind: Deployment
		metadata:
		   name: nginx-deployment
		spec:
		  template:
			spec:
			  containers:
			  - name: nginx
				env:
				- name: NGINX_PORT
				  value: "9999"
	*/
	patchmeta := map[string]interface{}{
		"name": name,
	}
	if namespacedResource {
		patchmeta["namespace"] = namespace
	}

	patchobj := map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   patchmeta,
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name": d.Get("container").(string),
							"env":  d.Get("env"),
						},
					},
				},
			},
		},
	}

	patch := unstructured.Unstructured{}
	patch.Object = patchobj
	patchbytes, err := patch.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = r.Patch(ctx,
		name,
		types.ApplyPatchType,
		patchbytes,
		v1.PatchOptions{
			FieldManager: defaultFieldManagerName,
			//Force:        ptrToBool(d.Get("force").(bool)),
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

	return nil
}
func resourceKubernetesEnvDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}
