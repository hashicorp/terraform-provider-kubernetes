package kubernetes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
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
	obj, err := getKubeObject(d)
	if err != nil {
		return err
	}

	rc, err := getResourceClient(meta.(*Meta).Config, obj)
	if err != nil {
		return err
	}

	uns := unstructured.Unstructured{
		Object: obj.Unstructured,
	}
	_, err = rc.Create(&uns)
	if err != nil {
		return fmt.Errorf("unable to create kubernetes resource: %s", err)
	}

	d.SetId(buildId(obj.Structured.Metadata))
	return nil
}

func resourceAnyRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAnyUpdate(d *schema.ResourceData, meta interface{}) error {
	obj, err := getKubeObject(d)
	if err != nil {
		return err
	}

	rc, err := getResourceClient(meta.(*Meta).Config, obj)
	if err != nil {
		return err
	}

	_, err = rc.Update(&unstructured.Unstructured{
		Object: obj.Unstructured,
	})
	if err != nil {
		return fmt.Errorf("unable to create kubernetes resource: %s", err)
	}

	d.SetId(buildId(obj.Structured.Metadata))
	return nil
}

func resourceAnyDelete(d *schema.ResourceData, meta interface{}) error {
	obj, err := getKubeObject(d)
	if err != nil {
		return err
	}

	rc, err := getResourceClient(meta.(*Meta).Config, obj)
	if err != nil {
		return err
	}

	fg := metav1.DeletePropagationForeground
	if err := rc.Delete(obj.Structured.Metadata.Name, &metav1.DeleteOptions{
		PropagationPolicy: &fg,
	}); err != nil {
		return err
	}
	// TODO: Check err to see it if it did not exist in first place

	d.SetId("")
	return nil
}

type kubeObject struct {
	Unstructured map[string]interface{}
	Structured   struct {
		APIVersion string            `json:"apiVersion"`
		Kind       string            `json:"kind"`
		Metadata   metav1.ObjectMeta `json:"metadata"`
	}
}

func (o *kubeObject) process() {
	if len(o.Structured.Metadata.Namespace) == 0 {
		o.Structured.Metadata.Namespace = "default"
	}
}

func getKubeObject(d *schema.ResourceData) (*kubeObject, error) {
	objJSON := []byte(d.Get("object_json").(string))

	var obj kubeObject

	// Unmarshal json twice into a map and a struct for pulling needed variables
	if err := json.Unmarshal(objJSON, &obj.Structured); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(objJSON, &obj.Unstructured); err != nil {
		return nil, err
	}

	obj.process()

	return &obj, nil
}

func getResourceClient(cfg restclient.Config, obj *kubeObject) (*dynamic.ResourceClient, error) {
	gv, err := runtimeschema.ParseGroupVersion(obj.Structured.APIVersion)
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

	resource := &metav1.APIResource{Name: kindToResource(obj.Structured.Kind), Namespaced: true}
	return c.Resource(resource, obj.Structured.Metadata.Namespace), nil
}

func kindToResource(k string) string {
	// TODO: Hacky, find the proper way (using discovery api?)
	return strings.ToLower(k) + "s"
}
