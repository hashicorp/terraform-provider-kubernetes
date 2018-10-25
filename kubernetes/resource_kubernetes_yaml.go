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
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
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
			"link": {
				Type:     schema.TypeString,
				Computed: true,
			},
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
	// conn := meta.(*kubernetes.Clientset)
	// restClient := conn.Core().RESTClient()

	// Check we can decode the yaml
	yaml := d.Get("yaml_body").(string)
	decoder := scheme.Codecs.UniversalDeserializer()
	obj, _, err := decoder.Decode([]byte(yaml), nil, nil)
	if err != nil {
		log.Printf("[INFO] Error parsing type: %#v", err)

		return err
	}
	metaObj := k8meta.AsPartialObjectMetadata(obj.(meta_v1.Object))
	log.Printf("[INFO] Creating YAML of type: %#v", metaObj)

	// kubectlcmd.Thing()
	obj.GetObjectKind()
	//todo: apply

	return resourceKubernetesYAMLRead(d, meta)
}

func resourceKubernetesYAMLRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)
	restClient := conn.Core().RESTClient()

	resourceLink := d.Get("link").(string)
	result := restClient.Get().RequestURI(resourceLink).Do()
	if result.Error() != nil {
		return result.Error()
	}

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
	conn := meta.(*kubernetes.Clientset)
	restClient := conn.Core().RESTClient()

	resourceLink := d.Get("link").(string)
	result := restClient.Get().RequestURI(resourceLink).Do()
	err := result.Error()
	if err != nil {
		return false, err
	}
	var statusCode int
	result.StatusCode(&statusCode)
	if statusCode != 200 {
		return false, nil
	}

	resource, err := result.Get()
	if err != nil {
		return false, err
	}

	log.Print(resource)

	return true, nil
}
