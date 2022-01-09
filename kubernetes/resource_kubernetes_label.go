package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

func resourceKubernetesLabelCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := newDynamicClientFromMeta(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	l, err := newLabelClient(d.Get, client)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new label %s", l.Key())

	err = l.Create(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new label: %s", l.Id())
	d.SetId(l.Id())

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

	client, err := newDynamicClientFromMeta(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	l, err := newLabelClient(d.Get, client)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading label %s", l.Key())

	value, err := l.Read(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if value != l.Value() {
		return diag.Errorf("expected value to be %s but received %s", l.Value(), value)
	}

	log.Printf("[INFO] Received label %s=%s", l.Key(), value)
	return nil
}

func resourceKubernetesLabelUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := newDynamicClientFromMeta(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	l, err := newLabelClient(d.Get, client)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updating label %s", l.Key())

	err = l.Update(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted updated label: %s=%s", l.Key(), l.Value())
	d.SetId(l.Id())

	return resourceKubernetesLabelRead(ctx, d, meta)
}

func resourceKubernetesLabelDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := newDynamicClientFromMeta(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	l, err := newLabelClient(d.Get, client)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting label %s", l.Key())

	err = l.Delete(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Label %s deleted", l.Id())

	d.SetId("")
	return nil
}

func resourceKubernetesLabelExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	client, err := newDynamicClientFromMeta(meta)
	if err != nil {
		return false, err
	}

	l, err := newLabelClient(d.Get, client)
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking label %s", l.Key())

	res, err := l.ReadResource(ctx)
	if err != nil {
		// requested resource does not exist
		return false, err
	}

	labels := res.GetLabels()
	_, ok := labels[l.Key()]
	return ok, nil
}
