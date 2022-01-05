package kubernetes

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

type kubernetesLabel struct {
	path       string
	id         string
	labelKey   string
	labelValue string
	patchBytes []byte
}

func kubernetesLabelFromResourceData(d *schema.ResourceData, operation string) (kubernetesLabel, error) {
	apiVersion, ok := d.Get("api_version").(string)
	if !ok || apiVersion == "" {
		return kubernetesLabel{}, fmt.Errorf("unable to extract api_version")
	}

	kind, ok := d.Get("kind").(string)
	if !ok || kind == "" {
		return kubernetesLabel{}, fmt.Errorf("unable to extract kind")
	}

	namespaceScoped, ok := d.Get("namespace_scoped").(bool)
	if !ok {
		return kubernetesLabel{}, fmt.Errorf("unable to extract namespace_scoped")
	}

	namespace, ok := d.Get("namespace").(string)
	if !ok {
		return kubernetesLabel{}, fmt.Errorf("unable to extract namespace")
	}

	name, ok := d.Get("name").(string)
	if !ok || name == "" {
		return kubernetesLabel{}, fmt.Errorf("unable to extract name")
	}

	labelKey, ok := d.Get("label_key").(string)
	if !ok || labelKey == "" {
		return kubernetesLabel{}, fmt.Errorf("unable to extract label_key")
	}

	labelValue, ok := d.Get("label_value").(string)
	if !ok || labelValue == "" {
		return kubernetesLabel{}, fmt.Errorf("unable to extract label_value")
	}

	apiPrefix := "apis"
	if apiVersion == "v1" {
		apiPrefix = "api"
	}

	path := fmt.Sprintf("/%s/%s/%s/%s", apiPrefix, apiVersion, kind, name)
	if namespaceScoped {
		path = fmt.Sprintf("/%s/%s/namespaces/%s/%s/%s", apiPrefix, apiVersion, namespace, kind, name)
	}
	path = strings.ToLower(path)

	patchPath := fmt.Sprintf("/metadata/labels/%s", labelKey)

	var op PatchOperation
	switch operation {
	case "add":
		op = &AddOperation{
			Path:  patchPath,
			Value: labelValue,
		}
	case "replace":
		op = &ReplaceOperation{
			Path:  patchPath,
			Value: labelValue,
		}
	case "remove":
		op = &RemoveOperation{
			Path: patchPath,
		}
	case "":
		op = nil
	default:
		return kubernetesLabel{}, fmt.Errorf("invalid operation")
	}

	ops := PatchOperations{op}
	patchBytes, err := ops.MarshalJSON()
	if err != nil {
		return kubernetesLabel{}, fmt.Errorf("failed to marshal operations: %w", err)
	}

	return kubernetesLabel{
		path:       path,
		id:         path,
		labelKey:   labelKey,
		labelValue: labelValue,
		patchBytes: patchBytes,
	}, nil
}

func resourceKubernetesLabel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesLabelCreate,
		ReadContext:   resourceKubernetesLabelRead,
		UpdateContext: resourceKubernetesLabelUpdate,
		DeleteContext: resourceKubernetesLabelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"api_version": {
				Type:        schema.TypeString,
				Description: "API Version defines what the Kubernetes API version of the resources.",
				Optional:    false,
				ForceNew:    true,
				Required:    true,
			},
			"kind": {
				Type:        schema.TypeString,
				Description: "Kind defines what the Kubernetes Kind of the resources.",
				Optional:    false,
				ForceNew:    true,
				Required:    true,
			},
			"namespace_scoped": {
				Type:        schema.TypeBool,
				Description: "Namespace scoped defines what the Kubernetes scope of the resource is. Defaults to false.",
				Optional:    true,
				ForceNew:    true,
				Default:     false,
			},
			"namespace": {
				Type:        schema.TypeString,
				Description: "Namespace defines the space within which name of the labeled resource must be unique.",
				Optional:    true,
				ForceNew:    true,
				Required:    false,
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "Name of the resource, must be unique. Cannot be updated.",
				Optional:     false,
				ForceNew:     true,
				Computed:     false,
				Required:     true,
				ValidateFunc: validateName,
			},
			"label_key": {
				Type:        schema.TypeString,
				Description: "Label key for the Kubernetes resource. (key=value)",
				Optional:    false,
				Required:    true,
				Computed:    false,
				ForceNew:    true,
			},
			"label_value": {
				Type:        schema.TypeString,
				Description: "Label value for the Kubernetes resource. (key=value)",
				Optional:    false,
				Required:    true,
				Computed:    false,
			},
		},
	}
}

func resourceKubernetesLabelCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	kl, err := kubernetesLabelFromResourceData(d, "add")
	if err != nil {
		return diag.Errorf("Failed to extract label data from resource data: %s", err)
	}

	log.Printf("[INFO] Creating new label: %#v", kl)

	res := conn.RESTClient().Patch(pkgApi.JSONPatchType).Body(kl.patchBytes).AbsPath(kl.path).Do(ctx)
	if res.Error() != nil {
		return diag.FromErr(res.Error())
	}

	log.Printf("[INFO] Submitted new label: %#v", kl)
	d.SetId(kl.id)

	return resourceKubernetesLabelRead(ctx, d, meta)
}

func resourceKubernetesLabelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesLabelExists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}

	kl, err := kubernetesLabelFromResourceData(d, "")
	if err != nil {
		return diag.FromErr(err)
	}

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading label %s", kl.id)

	res := conn.RESTClient().Get().AbsPath(kl.path).Do(ctx)
	if res.Error() != nil {
		log.Printf("[DEBUG] Received error: %#v", res.Error())
		return diag.FromErr(res.Error())
	}

	obj := metav1.PartialObjectMetadata{}
	err = res.Into(&obj)
	if err != nil {
		return diag.Errorf("unable to convert response into PartialObjectMetadata")
	}

	label, ok := obj.Labels[kl.labelKey]
	if !ok {
		return diag.Errorf("label not found with key: %s", kl.labelKey)
	}

	log.Printf("[INFO] Received label %s=%s", kl.labelKey, label)
	return nil
}

func resourceKubernetesLabelUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	kl, err := kubernetesLabelFromResourceData(d, "replace")
	if err != nil {
		return diag.FromErr(err)
	}

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating label %q: %v", kl.id, string(kl.patchBytes))

	res := conn.RESTClient().Patch(pkgApi.JSONPatchType).Body(kl.patchBytes).AbsPath(kl.path).Do(ctx)
	if res.Error() != nil {
		return diag.FromErr(res.Error())
	}

	log.Printf("[INFO] Submitted updated label: %s=%s", kl.labelKey, kl.labelValue)
	d.SetId(kl.id)

	return resourceKubernetesLabelRead(ctx, d, meta)
}

func resourceKubernetesLabelDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	kl, err := kubernetesLabelFromResourceData(d, "remove")
	if err != nil {
		return diag.FromErr(err)
	}

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting label: %#v", kl.id)
	res := conn.RESTClient().Patch(pkgApi.JSONPatchType).Body(kl.patchBytes).AbsPath(kl.path).Do(ctx)
	if res.Error() != nil {
		return diag.FromErr(res.Error())
	}

	log.Printf("[INFO] Label %s deleted", kl.id)

	d.SetId("")
	return nil
}

func resourceKubernetesLabelExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	kl, err := kubernetesLabelFromResourceData(d, "")
	if err != nil {
		return false, err
	}

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking label %s", kl.id)

	res := conn.RESTClient().Get().AbsPath(kl.path).Do(ctx)
	if res.Error() != nil {
		err := res.Error()
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}

		log.Printf("[DEBUG] Received error: %#v", err)
		return false, err
	}

	obj := metav1.PartialObjectMetadata{}
	err = res.Into(&obj)
	if err != nil {
		return false, err
	}

	_, ok := obj.Labels[kl.labelKey]
	return ok, nil
}
