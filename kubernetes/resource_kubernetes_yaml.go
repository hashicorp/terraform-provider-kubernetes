package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	k8meta "k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_v1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func resourceKubernetesYAML() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesYAMLCreate,
		Read:   resourceKubernetesYAMLRead,
		Exists: resourceKubernetesYAMLExists,
		Delete: resourceKubernetesYAMLDelete,
		Update: resourceKubernetesYAMLUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		CustomizeDiff: func(d *schema.ResourceDiff, meta interface{}) error {
			// Enable force new on yaml_body field.
			// This can't be done in the schema as it will fail internal validation
			// as all fields would be 'ForceNew' so no 'Update' func is needed.
			// but as we manually trigger an update in this compare function
			// we need the update function specified.
			d.ForceNew("yaml_body")

			// Get the UID of the K8s resource as it was when the `resourceKubernetesYAMLCreate` func completed.
			createdAtUID := d.Get("uid").(string)
			// Get the UID of the K8s resource as it currently is in the cluster.
			UID, exists := d.Get("live_uid").(string)
			if !exists {
				return nil
			}

			// Get the ResourceVersion of the K8s resource as it was when the `resourceKubernetesYAMLCreate` func completed.
			createdAtResourceVersion := d.Get("resource_version").(string)
			// Get it as it currently is in the cluster
			resourceVersion, exists := d.Get("live_resource_version").(string)
			if !exists {
				return nil
			}

			// If either UID or ResourceVersion differ between the current state and the cluster
			// trigger an update on the resource to get back in sync
			if UID != createdAtUID {
				log.Printf("[CUSTOMDIFF] DETECTED %s vs %s - %s vs %s", UID, createdAtUID)
				d.SetNewComputed("uid")
				return nil
			}

			if resourceVersion != createdAtResourceVersion {
				log.Printf("[CUSTOMDIFF] DETECTED %s vs %s", resourceVersion, createdAtResourceVersion)
				d.SetNewComputed("resource_version")
				return nil
			}

			return nil
		},
		Schema: map[string]*schema.Schema{
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"live_uid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"live_resource_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"yaml_body": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceKubernetesYAMLCreate(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)

	// Create a client to talk to the resource API based on the APIVersion and Kind
	// defined in the YAML
	client, absPath, rawObj, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		return fmt.Errorf("failed to create kuberentes rest client for resource: %+v", err)
	}
	metaObj := &meta_v1beta1.PartialObjectMetadata{}

	// Create the resource in Kubernetes
	err = client.Post().AbsPath(absPath["POST"]).Body(rawObj).Do().Into(metaObj)
	if err != nil {
		return fmt.Errorf("failed to create resource in kuberentes: %+v", err)
	}

	d.SetId(metaObj.GetSelfLink())
	// Capture the UID and Resource_version at time of creation
	// this allows us to diff these against the actual values
	// read in by the 'resourceKubernetesYAMLRead'
	d.Set("uid", metaObj.UID)
	d.Set("resource_version", metaObj.ResourceVersion)

	return resourceKubernetesYAMLRead(d, meta)
}

func resourceKubernetesYAMLRead(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)

	// Create a client to talk to the resource API based on the APIVersion and Kind
	// defined in the YAML
	client, absPaths, _, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		return fmt.Errorf("failed to create kuberentes rest client for resource: %+v", err)
	}

	// Get the resource from Kubernetes
	metaObjLive, exists, err := getResourceFromK8s(client, absPaths)
	if err != nil {
		return fmt.Errorf("failed to get resource '%s' from kubernetes: %+v", metaObjLive.SelfLink, err)
	}
	if !exists {
		return fmt.Errorf("resource '%s' reading didn't exist", metaObjLive.SelfLink)
	}

	if metaObjLive.UID == "" {
		return fmt.Errorf("Failed to parse item and get UUID: %+v", metaObjLive)
	}

	// Capture the UID and Resource_version from the cluster at the current time
	d.Set("live_uid", metaObjLive.UID)
	d.Set("live_resource_version", metaObjLive.ResourceVersion)

	return nil
}

func resourceKubernetesYAMLDelete(d *schema.ResourceData, meta interface{}) error {
	yaml := d.Get("yaml_body").(string)

	client, absPaths, _, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		return fmt.Errorf("failed to create kuberentes rest client for resource: %+v", err)
	}

	metaObj := &meta_v1beta1.PartialObjectMetadata{}
	err = client.Delete().AbsPath(absPaths["DELETE"]).Do().Into(metaObj)
	if err != nil {
		return fmt.Errorf("failed to delete kuberentes resource '%s': %+v", metaObj.SelfLink, err)
	}

	// Success remove it from state
	d.SetId("")

	return nil
}

func resourceKubernetesYAMLUpdate(d *schema.ResourceData, meta interface{}) error {
	err := resourceKubernetesYAMLDelete(d, meta)
	if err != nil {
		return err
	}
	return resourceKubernetesYAMLCreate(d, meta)
}

func resourceKubernetesYAMLExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	yaml := d.Get("yaml_body").(string)

	client, absPaths, _, err := getRestClientFromYaml(yaml, meta.(KubeProvider))
	if err != nil {
		return false, fmt.Errorf("failed to create kuberentes rest client for resource: %+v", err)
	}

	metaObj, exists, err := getResourceFromK8s(client, absPaths)
	if err != nil {
		return false, fmt.Errorf("failed to get resource '%s' from kubernetes: %+v", metaObj.SelfLink, err)
	}
	if exists {
		return true, nil
	}
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

	// Use the k8s Discovery service to find all valid APIs for this cluster
	clientSet, config := provider()
	discovery := clientSet.Discovery()
	resources, err := discovery.ServerResources()
	if err != nil {
		return nil, absPaths, nil, err
	}

	// Validate that the APIVersion provided in the YAML is valid for this cluster
	apiResource, exists := checkAPIResourceIsPresent(resources, metaObj)
	if !exists {
		return nil, absPaths, nil, fmt.Errorf("resource provided in yaml isn't valid for cluster, check the APIVersion and Kind fields are valid")
	}

	// Create rest config for the correct API based
	// on the YAML input
	gv := metaObj.TypeMeta.GroupVersionKind().GroupVersion()
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config.APIPath = "/apis"
	config.GroupVersion = &gv

	restClient, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, absPaths, nil, err
	}

	// To simplify usage of the client in each of the users of this func
	// we build up the correct AbsPaths to use with the rest client
	// this centralises logic around namespaced vs non-namespaced resources
	// leaving CRUD operations to simply select the right AbsPath to use
	// Note: This uses the APIResource information from the server to correctly
	// 			construct the URL for any supported resource on the server and isn't limited
	// 			to those present in the typed client.
	if apiResource.Namespaced {
		absPaths["GET"] = fmt.Sprintf("/apis/%s/namespaces/%s/%s/%s", gv.String(), metaObj.Namespace, apiResource.Name, metaObj.Name)
		absPaths["DELETE"] = fmt.Sprintf("/apis/%s/namespaces/%s/%s/%s", gv.String(), metaObj.Namespace, apiResource.Name, metaObj.Name)
		absPaths["POST"] = fmt.Sprintf("/apis/%s/namespaces/%s/%s/", gv.String(), metaObj.Namespace, apiResource.Name)
	} else {
		absPaths["GET"] = fmt.Sprintf("/apis/%s/%s/%s", gv.String(), apiResource.Name, metaObj.Name)
		absPaths["DELETE"] = fmt.Sprintf("/apis/%s/%s/%s", gv.String(), apiResource.Name, metaObj.Name)
		absPaths["POST"] = fmt.Sprintf("/apis/%s/%s/", gv.String(), apiResource.Name)
	}

	return restClient, absPaths, rawObj, nil
}

// getResourceMetaObjFromYaml Uses the UniversalDeserializer to deserialize
// the yaml provided into a k8s runtime.Object
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

// checkAPIResourceIsPresent Loops through a list of available APIResources and
// checks there is a resource for the APIVersion and Kind defined in the 'resource'
// if found it returns true and the APIResource which matched
func checkAPIResourceIsPresent(available []*meta_v1.APIResourceList, resource *meta_v1beta1.PartialObjectMetadata) (*meta_v1.APIResource, bool) {
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

// runtimeObjToMetaObj Gets a subset of the full object information
// just enough to construct the API Calls needed and detect any changes
// made to the object in the cluster (UID & ResourceVersion)
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
