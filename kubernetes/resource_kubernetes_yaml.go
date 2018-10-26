package kubernetes

import (
	// "bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	k8meta "k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_v1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func resourceKubernetesYAML() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesYAMLCreate,
		Read:   resourceKubernetesYAMLRead,
		Exists: resourceKubernetesYAMLExists,
		Delete: resourceKubernetesYAMLDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"uuid": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"resource_version": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"yaml_body": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceKubernetesYAMLCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Resource kubernetes_yaml")

	yaml := d.Get("yaml_body").(string)
	client, absPath, rawObj, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		log.Printf("[INFO] !!!! Error creating client: '%+v'", err)
		return err
	}
	metaObj := &meta_v1beta1.PartialObjectMetadata{}

	res := client.Post().AbsPath(absPath["POST"]).Body(rawObj).Do().Into(metaObj)
	if res != nil {
		log.Printf("[INFO] !!!! Error creating resource: '%+v'", res.Error())
		return res
	}

	d.SetId(metaObj.GetSelfLink())

	return resourceKubernetesYAMLRead(d, meta)
}

func resourceKubernetesYAMLRead(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)

	client, absPaths, _, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		log.Printf("[INFO] !!!! Error creating client: '%+v'", err)
		return err
	}

	metaObjLive, exists, err := getResourceFromK8s(client, absPaths)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Errorf("resource reading didn't exist, unexpected")
	}

	log.Printf("[INFO] !!!! META: '%+v'", metaObjLive)

	if metaObjLive.UID == "" {
		return fmt.Errorf("Failed to parse item and get UUID: %+v", metaObjLive)
	}

	d.Set("uuid", metaObjLive.UID)
	d.Set("resource_version", metaObjLive.ResourceVersion)

	return nil
}

func resourceKubernetesYAMLDelete(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)

	client, absPaths, _, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		log.Printf("[INFO] !!!! Error creating client: '%+v'", err)
		return err
	}

	res := client.Delete().AbsPath(absPaths["GET"]).Do()
	if res.Error() != nil {
		log.Printf("[INFO] !!!! Error creating resource: '%+v'", res.Error())
		return res.Error()
	}

	// Success remove it from state
	d.SetId("")

	return nil
}

func resourceKubernetesYAMLExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	log.Printf("[INFO] !!!! EXISTS called")

	yaml := d.Get("yaml_body").(string)

	client, absPaths, _, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		log.Printf("[INFO] !!!! Error creating client: '%+v'", err)
		return false, err
	}
	log.Printf("[INFO] !!!! EXISTS - GOT client")

	_, exists, err := getResourceFromK8s(client, absPaths)
	if err != nil {
		log.Printf("[INFO] !!!! EXISTS - ERROR GETTING RESOURCE")

		return false, err
	}
	if exists {
		log.Printf("[INFO] !!!! WE THINK IT EXISTS")
		return true, nil
	}
	log.Printf("[INFO] !!!! EXISTS - RETURNED FALSE")
	return false, nil
}

func getResourceFromK8s(client rest.Interface, absPaths map[string]string) (*meta_v1beta1.PartialObjectMetadata, bool, error) {
	result := client.Get().AbsPath(absPaths["GET"]).Do()

	var statusCode int
	result.StatusCode(&statusCode)
	// Resource doesn't exist
	if statusCode != 200 {
		return nil, false, nil
	}

	// Another error occured
	response, err := result.Get()
	if err != nil {
		return nil, false, err
	}

	// Get the metadata we need
	metaObj, err := runtimeObjToMetaObj(response)
	if err != nil {
		return nil, true, err
	}

	return metaObj, true, err
}

func getRestClientFromYaml(yaml string, provider KubeProvider) (*rest.RESTClient, map[string]string, runtime.Object, error) {
	absPaths := map[string]string{}
	metaObj, rawObj, err := getResourceMetaObjFromYaml(yaml)
	if err != nil {
		return nil, absPaths, nil, err
	}
	clientSet, config := provider()
	discovery := clientSet.Discovery()
	resources, err := discovery.ServerResources()
	if err != nil {
		return nil, absPaths, nil, err
	}

	apiResource, exists := getAPIResourceFromServer(resources, metaObj)
	log.Printf("[INFO] Is Resource Valid: '%+v'", exists)
	if !exists {
		return nil, absPaths, nil, fmt.Errorf("resource provided in yaml isn't valid for cluster, check the APIVersion and Kind fields are valid")
	}

	log.Printf("[INFO] Resource: '%+v'", metaObj.TypeMeta.GroupVersionKind().GroupVersion())

	gv := metaObj.TypeMeta.GroupVersionKind().GroupVersion()
	log.Printf("[INFO] !!!! GroupVersion Kind: %#v", gv)

	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config.APIPath = "/apis"
	config.GroupVersion = &gv
	log.Printf("[INFO] !!!! Build config")

	restClient, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, absPaths, nil, err
	}

	if apiResource.Namespaced {
		absPaths["GET"] = fmt.Sprintf("/apis/%s/namespaces/%s/%s/%s", gv.String(), metaObj.Namespace, apiResource.Name, metaObj.Name)
		absPaths["POST"] = fmt.Sprintf("/apis/%s/namespaces/%s/%s/", gv.String(), metaObj.Namespace, apiResource.Name)
	} else {
		absPaths["GET"] = fmt.Sprintf("/apis/%s/%s/%s", gv.String(), apiResource.Name, metaObj.Name)
		absPaths["POST"] = fmt.Sprintf("/apis/%s/%s/", gv.String(), apiResource.Name)
	}
	log.Printf("[INFO] !!!! PATH: %#v", absPaths)

	return restClient, absPaths, rawObj, nil
}

func getResourceMetaObjFromYaml(yaml string) (*meta_v1beta1.PartialObjectMetadata, runtime.Object, error) {
	decoder := scheme.Codecs.UniversalDeserializer()
	obj, _, err := decoder.Decode([]byte(yaml), nil, nil)
	if err != nil {
		log.Printf("[INFO] Error parsing type: %#v", err)
		return nil, nil, err
	}
	metaObj, err := runtimeObjToMetaObj(obj)
	if err != nil {
		return nil, nil, err
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

func runtimeObjToMetaObj(obj runtime.Object) (*meta_v1beta1.PartialObjectMetadata, error) {
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
