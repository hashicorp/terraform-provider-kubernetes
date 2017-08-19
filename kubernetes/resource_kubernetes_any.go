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
				// TODO: Add a validator (kube lib?)
			},
		},
	}
}

func resourceAnyCreate(d *schema.ResourceData, meta interface{}) error {
	objs, err := getKubeObjects(d)
	if err != nil {
		return err
	}

	rc, err := getResourceClient(meta.(*Meta).Config, objs)
	if err != nil {
		return err
	}

	_, err = rc.Create(&unstructured.Unstructured{
		Object: objs.Unstructured,
	})
	if err != nil {
		return fmt.Errorf("unable to create kubernetes resource: %s", err)
	}

	d.SetId(buildId(objs.Structured.Metadata))
	return nil
}

func resourceAnyRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAnyUpdate(d *schema.ResourceData, meta interface{}) error {
	objs, err := getKubeObjects(d)
	if err != nil {
		return err
	}

	rc, err := getResourceClient(meta.(*Meta).Config, objs)
	if err != nil {
		return err
	}

	_, err = rc.Update(&unstructured.Unstructured{
		Object: objs.Unstructured,
	})
	if err != nil {
		return fmt.Errorf("unable to create kubernetes resource: %s", err)
	}

	d.SetId(buildId(objs.Structured.Metadata))
	return nil
}

func resourceAnyDelete(d *schema.ResourceData, meta interface{}) error {
	objs, err := getKubeObjects(d)
	if err != nil {
		return err
	}

	rc, err := getResourceClient(meta.(*Meta).Config, objs)
	if err != nil {
		return err
	}

	fg := metav1.DeletePropagationForeground
	if err := rc.Delete(objs.Structured.Metadata.Name, &metav1.DeleteOptions{
		PropagationPolicy: &fg,
	}); err != nil {
		return err
	}
	// TODO: Check err to see it if it did not exist in first place

	d.SetId("")
	return nil
}

type kubeObjects struct {
	Unstructured map[string]interface{}
	Structured   struct {
		APIVersion string            `json:"apiVersion"`
		Metadata   metav1.ObjectMeta `json:"metadata"`
	}
}

func (o *kubeObjects) process() {
	if len(o.Structured.Metadata.Namespace) == 0 {
		o.Structured.Metadata.Namespace = "default"
	}
}

func getKubeObjects(d *schema.ResourceData) (*kubeObjects, error) {
	objJSON := []byte(d.Get("object_json").(string))

	var objs kubeObjects

	// Unmarshal json twice into a map and a struct for pulling needed variables
	if err := json.Unmarshal(objJSON, &objs.Structured); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(objJSON, &objs.Unstructured); err != nil {
		return nil, err
	}

	objs.process()

	return &objs, nil
}

func getResourceClient(cfg *restclient.Config, objs *kubeObjects) (*dynamic.ResourceClient, error) {
	// TODO: More error handling (type assertion, etc.)
	gv := strings.Split(objs.Structured.APIVersion, "/")
	cfg.ContentConfig = restclient.ContentConfig{GroupVersion: &runtimeschema.GroupVersion{Group: gv[0], Version: gv[1]}}
	// TODO: Look into using API Path resolver out of kube lib
	cfg.APIPath = "/apis"

	c, err := dynamic.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create dynamic client: %s", err)
	}

	resource := &metav1.APIResource{Name: "deployments", Namespaced: true}
	return c.Resource(resource, objs.Structured.Metadata.Namespace), nil
}
