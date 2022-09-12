package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// TODO:
/*
* add read function
* add delete function
* add support for cronjobs
* add tests
 */

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
			"force": {
				Type:        schema.TypeBool,
				Description: "Force overwriting environments that were created or edited outside of Terraform.",
				Optional:    true,
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

	configuredEnvs := d.Get("env").(map[string]interface{})

	// strip out envs not managed by Terraform
	managedEnvs, err := getManagedEnvs(res.GetManagedFields(), defaultFieldManagerName, d)
	if err != nil {
		return diag.FromErr(err)
	}
	envs := res.GetEnvs(fmt.Sprintf("k:{\"name\":\"%s\"}", d.Get("container")))
	for k := range envs {
		_, managed := managedEnvs[k]
		_, configured := configuredEnvs[k]
		if !managed && !configured {
			delete(envs, k)
		}
	}

	d.Set("env", envs)
	return nil
}

/*
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:template:
          f:spec:
            f:containers:
              k:{"name":"nginx"}:
                .: {}
                f:env:
                  k:{"name":"NGINX_HOST"}:
                    .: {}
                    f:name: {}
                    f:value: {}
                  k:{"name":"NGINX_PORT"}:
                    .: {}
                    f:name: {}
                    f:value: {}
                f:name: {}
*/

func getManagedEnvs(managedFields []v1.ManagedFieldsEntry, manager string, d *schema.ResourceData) (map[string]interface{}, error) {
	var envs map[string]interface{}
	for _, m := range managedFields {
		if m.Manager != manager {
			continue
		}
		var mm map[string]interface{}
		err := json.Unmarshal(m.FieldsV1.Raw, &mm)
		if err != nil {
			return nil, err
		}
		spec1 := mm["f:spec"].(map[string]interface{})
		template := spec1["f:template"].(map[string]interface{})
		spec2 := template["f:spec"].(map[string]interface{})
		container := spec2["containers"].(map[string]interface{})
		containerVal := fmt.Sprintf("k:{\"name\":\"%s\"}", d.Get("container"))
		k := container[containerVal].(map[string]interface{})
		if e, ok := k["f:env"].(map[string]interface{}); ok {
			envs = e
		}

		/*
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

		*/
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
			Force:        ptrToBool(d.Get("force").(bool)),
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
