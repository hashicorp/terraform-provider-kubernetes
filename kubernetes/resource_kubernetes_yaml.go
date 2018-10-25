package kubernetes

import (
	// "bytes"
	// "fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	// k8serrors "k8s.io/apimachinery/pkg/api/errors"
	// meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8meta "k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_v1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	kubectlcmd "k8s.io/kubernetes/pkg/kubectl/cmd"
	// "k8s.io/client-go/pkg/api"
	// api "k8s.io/api/core/v1"
)

func resourceKubernetesYAML() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesYAMLCreate,
		Read:   resourceKubernetesYAMLRead,
		Exists: resourceKubernetesYAMLExists,
		Update: resourceKubernetesYAMLUpdate,
		Delete: resourceKubernetesYAMLDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"yaml_body": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"outputs": {
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func resourceKubernetesYAMLCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Resource kubernetes_yaml")

	conn := meta.(*kubernetes.Clientset)
	restClient := conn.Core().RESTClient()

	// Check we can decode the yaml
	yaml := d.Get("yaml_body").(string)
	metaObj, err := getResourceMetaObjFromYaml(yaml)

	log.Printf("[INFO] Resource: '%+v'", metaObj)

	// Does that resource already exist in Kubernetes
	_, exists, err := getResourceFromMetaObj(restClient, metaObj)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("[INFO] Resource already present in cluster: %#v", metaObj)
	}

	return resourceKubernetesYAMLRead(d, meta)
}

func resourceKubernetesYAMLRead(d *schema.ResourceData, meta interface{}) error {
	// conn := meta.(*kubernetes.Clientset)
	// restClient := conn.Core().RESTClient()

	//todo: return resource

	return nil
}

func resourceKubernetesYAMLUpdate(d *schema.ResourceData, meta interface{}) error {
	// conn := meta.(*kubernetes.Clientset)

	return resourceKubernetesSecretRead(d, meta)
}

func resourceKubernetesYAMLDelete(d *schema.ResourceData, meta interface{}) error {
	// conn := meta.(*kubernetes.Clientset)

	return nil
}

func resourceKubernetesYAMLExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	log.Printf("[INFO] Checking Resource exists kubernetes_yaml")
	conn := meta.(*kubernetes.Clientset)
	restClient := conn.Core().RESTClient()

	yaml := d.Get("yaml").(string)
	metaObj, err := getResourceMetaObjFromYaml(yaml)
	if err != nil {
		return false, err
	}
	_, exists, err := getResourceFromMetaObj(restClient, metaObj)

	return exists, err
}

func getResourceFromMetaObj(client rest.Interface, metaObj *meta_v1beta1.PartialObjectMetadata) (meta_v1.Object, bool, error) {
	result := client.
		Get().
		Namespace(metaObj.GetNamespace()).
		Resource(metaObj.TypeMeta.Kind).
		Name(metaObj.GetName()).
		Do()

	response, err := result.Get()
	if err != nil && err.Error() == "the server could not find the requested resource" {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	res, err := k8meta.Accessor(response.(meta_v1.Object))
	return res, true, err
}

func getResourceMetaObjFromYaml(yaml string) (*meta_v1beta1.PartialObjectMetadata, error) {
	decoder := scheme.Codecs.UniversalDeserializer()
	obj, _, err := decoder.Decode([]byte(yaml), nil, nil)
	if err != nil {
		log.Printf("[INFO] Error parsing type: %#v", err)
		return nil, err
	}
	metaObj := k8meta.AsPartialObjectMetadata(obj.(meta_v1.Object))
	typeMeta, err := k8meta.TypeAccessor(obj)
	if err != nil {
		return nil, err
	}
	metaObj.TypeMeta = meta_v1.TypeMeta{
		APIVersion: typeMeta.GetAPIVersion(),
		Kind:       typeMeta.GetKind(),
	}
	if metaObj.Namespace == "" {
		metaObj.Namespace = "default"
	}
	return metaObj, nil

}
