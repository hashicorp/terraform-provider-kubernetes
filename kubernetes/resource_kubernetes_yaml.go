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

	yaml := d.Get("yaml_body").(string)
	client, absPath, rawObj, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		log.Printf("[INFO] !!!! Error creating client: '%+v'", err)
		return err
	}

	pathMap := *absPath
	res := client.Post().AbsPath(pathMap["POST"]).Body(rawObj).Do()
	if res.Error() != nil {
		log.Printf("[INFO] !!!! Error creating resource: '%+v'", res.Error())

		return res.Error()
	}

	// log.Printf("[INFO] !!!!!! Resource created in cluster: %#v", res)

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
	// conn, _ := meta.(KubeProvider)()

	// yaml := d.Get("yaml").(string)
	// metaObj, _, err := getResourceMetaObjFromYaml(yaml)
	// if err != nil {
	// 	return false, err
	// }
	// _, exists, err := getResourceFromMetaObj(restClient, "bob", metaObj)

	return false, nil
}

func getResourceFromMetaObj(client rest.Interface, metaObj *meta_v1beta1.PartialObjectMetadata) (*meta_v1beta1.PartialObjectMetadata, bool, error) {
	result := client.Get().Do()

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

func getRestClientFromYaml(yaml string, provider KubeProvider) (*rest.RESTClient, *map[string]string, runtime.Object, error) {
	metaObj, rawObj, err := getResourceMetaObjFromYaml(yaml)
	if err != nil {
		return nil, nil, nil, err
	}
	clientSet, config := provider()
	discovery := clientSet.Discovery()
	resources, err := discovery.ServerResources()
	if err != nil {
		return nil, nil, nil, err
	}

	apiResource, exists := getAPIResourceFromServer(resources, metaObj)
	log.Printf("[INFO] Is Resource Valid: '%+v'", exists)
	if !exists {
		return nil, nil, nil, fmt.Errorf("resource provided in yaml isn't valid for cluster, check the APIVersion and Kind fields are valid")
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
		return nil, nil, nil, err
	}

	absPaths := map[string]string{}
	if apiResource.Namespaced {
		absPaths["GET"] = fmt.Sprintf("/apis/%s/namespaces/%s/%s/%s", gv.String(), metaObj.Namespace, apiResource.Name, metaObj.Name)
		absPaths["POST"] = fmt.Sprintf("/apis/%s/namespaces/%s/%s/", gv.String(), metaObj.Namespace, apiResource.Name)
	} else {
		absPaths["GET"] = fmt.Sprintf("/apis/%s/%s/%s", gv.String(), apiResource.Name, metaObj.Name)
		absPaths["POST"] = fmt.Sprintf("/apis/%s/%s/", gv.String(), apiResource.Name)
	}
	log.Printf("[INFO] !!!! PATH: %#v", absPaths)

	return restClient, &absPaths, rawObj, nil
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
