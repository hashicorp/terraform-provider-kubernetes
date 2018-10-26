package kubernetes

import (
	// "bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	// k8serrors "k8s.io/apimachinery/pkg/api/errors"
	// meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8meta "k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_v1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	// kubectlcmd "k8s.io/kubernetes/pkg/kubectl/cmd"
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

	// Check we can decode the yaml
	yaml := d.Get("yaml_body").(string)
	metaObj, obj, err := getResourceMetaObjFromYaml(yaml)
	if err != nil {
		return err
	}
	clientSet, config := meta.(KubeProvider)()
	discovery := clientSet.Discovery()
	resources, err := discovery.ServerResources()
	if err != nil {
		return err
	}

	r, exists := getAPIResourceFromServer(resources, metaObj)
	log.Printf("[INFO] Is Resource Valid: '%+v'", exists)
	if !exists {
		return fmt.Errorf("resource provided in yaml isn't valid for cluster, check the APIVersion and Kind fields are valid")
	}

	log.Printf("[INFO] Resource: '%+v'", metaObj.TypeMeta.GroupVersionKind().GroupVersion())

	gv := metaObj.TypeMeta.GroupVersionKind().GroupVersion()
	log.Printf("[INFO] !!!! GroupVersion Kind: %#v", gv)
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config.GroupVersion = &gv
	log.Printf("[INFO] !!!! Build config")

	restClient, err := rest.RESTClientFor(&config)
	if err != nil {
		return err
	}

	// // Does that resource already exist in Kubernetes
	// _, exists, err = getResourceFromMetaObj(restClient, r.Name, metaObj)
	// if err != nil {
	// 	return err
	// }
	// if exists {
	// 	log.Printf("[INFO] Resource IS present in cluster: %#v", metaObj)
	// } else {
	// 	log.Printf("[INFO] Resource NOT present in cluster: %#v", metaObj)
	// }

	res := restClient.Post().Namespace(metaObj.Namespace).Body(obj).Resource(r.Name).Do()
	if res.Error() != nil {
		log.Printf("[INFO] !!!! Error: '%+v'", metaObj)

		return res.Error()
	}

	log.Printf("[INFO] !!!!!! Resource created in cluster: %#v", res)

	return resourceKubernetesYAMLRead(d, meta)
}

func resourceKubernetesYAMLRead(d *schema.ResourceData, meta interface{}) error {
	// conn, _ := meta.(KubeProvider)()
	// restClient := conn.Core().RESTClient()

	//todo: return resource

	return nil
}

func resourceKubernetesYAMLUpdate(d *schema.ResourceData, meta interface{}) error {
	// conn, _ := meta.(KubeProvider)()

	return resourceKubernetesSecretRead(d, meta)
}

func resourceKubernetesYAMLDelete(d *schema.ResourceData, meta interface{}) error {
	// conn, _ := meta.(KubeProvider)()

	return nil
}

func resourceKubernetesYAMLExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	log.Printf("[INFO] Checking Resource exists kubernetes_yaml")
	conn, _ := meta.(KubeProvider)()
	restClient := conn.Core().RESTClient()

	yaml := d.Get("yaml").(string)
	metaObj, _, err := getResourceMetaObjFromYaml(yaml)
	if err != nil {
		return false, err
	}
	_, exists, err := getResourceFromMetaObj(restClient, "bob", metaObj)

	return exists, err
}

func getResourceFromMetaObj(client rest.Interface, resource string, metaObj *meta_v1beta1.PartialObjectMetadata) (*meta_v1beta1.PartialObjectMetadata, bool, error) {
	// name, err := k8meta.DefaultRESTMapper{}.ResourceFor(metaObj.GetObjectKind().GroupVersionKind().GroupKind().)
	// if err != nil {
	// 	return nil, false, err
	// }

	result := client.
		Get().
		Namespace(metaObj.GetNamespace()).
		Resource(resource).
		Name(metaObj.GetName()).
		Do()

	response, err := result.Get()
	if err != nil && err.Error() == "the server could not find the requested resource" {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	typeMeta, err := k8meta.TypeAccessor(response)
	if err != nil {
		return nil, false, err
	}
	metaObj.TypeMeta = meta_v1.TypeMeta{
		APIVersion: typeMeta.GetAPIVersion(),
		Kind:       typeMeta.GetKind(),
	}
	if metaObj.Namespace == "" {
		metaObj.Namespace = "default"
	}

	return metaObj, true, err
}

func getResourceMetaObjFromYaml(yaml string) (*meta_v1beta1.PartialObjectMetadata, runtime.Object, error) {
	decoder := scheme.Codecs.UniversalDeserializer()
	obj, _, err := decoder.Decode([]byte(yaml), nil, nil)
	if err != nil {
		log.Printf("[INFO] Error parsing type: %#v", err)
		return nil, nil, err
	}
	metaObj := k8meta.AsPartialObjectMetadata(obj.(meta_v1.Object))
	typeMeta, err := k8meta.TypeAccessor(obj)
	if err != nil {
		return nil, nil, err
	}
	metaObj.TypeMeta = meta_v1.TypeMeta{
		APIVersion: typeMeta.GetAPIVersion(),
		Kind:       typeMeta.GetKind(),
	}
	if metaObj.Namespace == "" {
		metaObj.Namespace = "default"
	}
	return metaObj, obj, nil

}

func getAPIResourceFromServer(available []*meta_v1.APIResourceList, resource *meta_v1beta1.PartialObjectMetadata) (*meta_v1.APIResource, bool) {
	for _, rList := range available {
		if rList == nil {
			continue
		}
		group := rList.GroupVersion
		for _, r := range rList.APIResources {
			if group == resource.TypeMeta.APIVersion && r.Kind == resource.Kind {
				return &r, true
			}
		}
	}
	return nil, false
}
