package kubernetes

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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

type labelClientAction int

const (
	createLabelClientAction labelClientAction = iota + 1
	updateLabelClientAction
	deleteLabelClientAction
	readLabelClientAction
)

type labelClient struct {
	action     labelClientAction
	path       string
	id         string
	key        string
	value      string
	patchBytes []byte
}

func newLabelClient(d *schema.ResourceData, action labelClientAction) (*labelClient, error) {
	apiVersion, ok := d.Get("api_version").(string)
	if !ok || apiVersion == "" {
		return nil, fmt.Errorf("unable to extract api_version")
	}

	kind, ok := d.Get("kind").(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("unable to extract kind")
	}

	namespaceScoped, ok := d.Get("namespace_scoped").(bool)
	if !ok {
		return nil, fmt.Errorf("unable to extract namespace_scoped")
	}

	namespace, ok := d.Get("namespace").(string)
	if !ok {
		return nil, fmt.Errorf("unable to extract namespace")
	}

	name, ok := d.Get("name").(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("unable to extract name")
	}

	key, ok := d.Get("label_key").(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("unable to extract label_key")
	}

	value, ok := d.Get("label_value").(string)
	if !ok || value == "" {
		return nil, fmt.Errorf("unable to extract label_value")
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

	patchPath := fmt.Sprintf("/metadata/labels/%s", key)

	var op PatchOperation
	switch action {
	case createLabelClientAction:
		op = &AddOperation{
			Path:  patchPath,
			Value: value,
		}
	case updateLabelClientAction:
		op = &ReplaceOperation{
			Path:  patchPath,
			Value: value,
		}
	case deleteLabelClientAction:
		op = &RemoveOperation{
			Path: patchPath,
		}
	case readLabelClientAction:
		op = nil
	default:
		return nil, fmt.Errorf("invalid operation")
	}

	ops := PatchOperations{op}
	patchBytes, err := ops.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal operations: %w", err)
	}

	return &labelClient{
		action:     action,
		path:       path,
		id:         path,
		key:        key,
		value:      value,
		patchBytes: patchBytes,
	}, nil
}

func (lc *labelClient) getRequest(conn *kubernetes.Clientset) *rest.Request {
	if lc.action == readLabelClientAction {
		return conn.RESTClient().Get().AbsPath(lc.path)
	}

	return conn.RESTClient().Patch(pkgApi.JSONPatchType).Body(lc.patchBytes).AbsPath(lc.path)
}

func (lc *labelClient) doRequest(ctx context.Context, conn *kubernetes.Clientset) (rest.Result, error) {
	req := lc.getRequest(conn)
	res := req.Do(ctx)
	if res.Error() != nil {
		return rest.Result{}, res.Error()
	}

	return res, nil
}

func (lc *labelClient) getLabelsFromResponse(res rest.Result) (map[string]string, error) {
	obj := metav1.PartialObjectMetadata{}
	err := res.Into(&obj)
	if err != nil {
		return nil, err
	}

	return obj.Labels, nil
}

func (lc *labelClient) getvalueFromResponse(res rest.Result) (string, error) {
	labels, err := lc.getLabelsFromResponse(res)
	if err != nil {
		return "", err
	}

	value, ok := labels[lc.key]
	if !ok {
		return "", fmt.Errorf("label not found with key: %s", lc.key)
	}

	return value, nil
}

func resourceKubernetesLabelCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	lc, err := newLabelClient(d, createLabelClientAction)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Print("[INFO] Creating new label %s", lc.key)

	_, err = lc.doRequest(ctx, conn)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new label: %#v", lc)
	d.SetId(lc.id)

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

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	lc, err := newLabelClient(d, readLabelClientAction)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading label %s", lc.key)

	res, err := lc.doRequest(ctx, conn)
	if err != nil {
		return diag.FromErr(err)
	}

	resValue, err := lc.getvalueFromResponse(res)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received label %s=%s", lc.key, resValue)
	return nil
}

func resourceKubernetesLabelUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	lc, err := newLabelClient(d, updateLabelClientAction)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating label %s", lc.key)

	_, err = lc.doRequest(ctx, conn)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted updated label: %s=%s", lc.key, lc.value)
	d.SetId(lc.id)

	return resourceKubernetesLabelRead(ctx, d, meta)
}

func resourceKubernetesLabelDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	lc, err := newLabelClient(d, deleteLabelClientAction)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting label %s", lc.key)

	_, err = lc.doRequest(ctx, conn)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Label %s deleted", lc.id)

	d.SetId("")
	return nil
}

func resourceKubernetesLabelExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	lc, err := newLabelClient(d, readLabelClientAction)
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking label %s", lc.key)

	res, err := lc.doRequest(ctx, conn)
	if err != nil {
		// requested resource does not exist
		return false, err
	}

	labels, err := lc.getLabelsFromResponse(res)
	if err != nil {
		return false, err
	}

	_, ok := labels[lc.key]
	return ok, nil
}
