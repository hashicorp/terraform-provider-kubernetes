package kubernetes

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	//"k8s.io/kubernetes/pkg/kubectl/validation"
)

func resourceKubernetesAny() *schema.Resource {
	return &schema.Resource{
		Create: resourceAnyCreate,
		Read:   resourceAnyRead,
		Update: resourceAnyUpdate,
		Delete: resourceAnyDelete,

		Schema: map[string]*schema.Schema{
			"object_json": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Raw json definition of a kubernetes resource",
				Required:    true,
				//ValidateFunc: validateObjectJSON,
			},
		},
	}
}

/*
func validateObjectJSON(x interface{}, s string) (strs []string, errs []error) {
	btys := []byte(fmt.Sprint(x))
	if err := (validation.NullSchema{}).ValidateBytes(btys); err != nil {
		errs = append(errs, err)
	}
	return nil, nil
}
*/

func resourceAnyCreate(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	uns, err := getUnstructured(d)
	if err != nil {
		return err
	}

	rc, err := getResourceClient(m.Config, m.RESTMapper, uns)
	if err != nil {
		return err
	}

	_, err = rc.Create(uns)
	if err != nil {
		return fmt.Errorf("unable to create kubernetes resource: %s", err)
	}

	d.SetId(buildId(unstructuredMeta(uns)))
	return nil
}

func resourceAnyRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAnyUpdate(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	uns, err := getUnstructured(d)
	if err != nil {
		return err
	}

	rc, err := getResourceClient(m.Config, m.RESTMapper, uns)
	if err != nil {
		return err
	}

	_, err = rc.Update(uns)
	if err != nil {
		return fmt.Errorf("unable to create kubernetes resource: %s", err)
	}

	d.SetId(buildId(unstructuredMeta(uns)))
	return nil
}

func resourceAnyDelete(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	uns, err := getUnstructured(d)
	if err != nil {
		return err
	}

	rc, err := getResourceClient(m.Config, m.RESTMapper, uns)
	if err != nil {
		return err
	}

	fg := metav1.DeletePropagationForeground
	if err := rc.Delete(uns.GetName(), &metav1.DeleteOptions{
		PropagationPolicy: &fg,
	}); err != nil {
		return err
	}
	// TODO: Check err to see it if it did not exist in first place

	d.SetId("")
	return nil
}

func getUnstructured(d *schema.ResourceData) (*unstructured.Unstructured, error) {
	objJSON := []byte(d.Get("object_json").(string))

	var uns unstructured.Unstructured
	if err := json.Unmarshal(objJSON, &uns); err != nil {
		return nil, err
	}

	return &uns, nil
}

func getResourceClient(cfg restclient.Config, rm *meta.DefaultRESTMapper, uns *unstructured.Unstructured) (*dynamic.ResourceClient, error) {
	// Create the dynamic client.
	gv, err := runtimeschema.ParseGroupVersion(uns.GetAPIVersion())
	if err != nil {
		return nil, fmt.Errorf("unable to parse group/version: %s", err)
	}
	cfg.ContentConfig = restclient.ContentConfig{GroupVersion: &gv}
	// TODO: Look into using API Path resolver out of kube lib
	cfg.APIPath = "/apis"
	c, err := dynamic.NewClient(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create dynamic client: %s", err)
	}

	// Map the object to a REST resource.
	gk := runtimeschema.GroupKind{Group: gv.Group, Kind: uns.GetKind()}
	m, err := rm.RESTMapping(gk, gv.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to get rest mapping: %s", err)
	}

	// Specify the resource.
	nsd := m.Scope.Name() == meta.RESTScopeNameNamespace
	ns := uns.GetNamespace()
	if len(ns) == 0 && nsd {
		ns = "default"
	}
	resource := &metav1.APIResource{Name: m.Resource, Namespaced: nsd}

	return c.Resource(resource, ns), nil
}

func unstructuredMeta(uns *unstructured.Unstructured) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: uns.GetNamespace(),
		Name:      uns.GetName(),
	}
}
