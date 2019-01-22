package kubernetes

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/ghodss/yaml"
	flat "github.com/hashicorp/terraform/flatmap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	rest "k8s.io/client-go/rest"
)

type GenericKubernetesObject struct {
	*runtime.Unknown
	RawJSON    map[string]interface{}
	ObjectMeta *metav1.ObjectMeta
	GenericID  *GenericObjectID
}

func (obj *GenericKubernetesObject) FlattenedSpec() map[string]string {
	processedSpec := obj.RawJSON["spec"].(map[string]interface{})
	return flat.Flatten(processedSpec)
}

func (obj *GenericKubernetesObject) YamlSpec() string {
	processedSpec := obj.RawJSON["spec"].(map[string]interface{})
	bytes, _ := yaml.Marshal(processedSpec)
	return string(bytes)
}

func (obj *GenericKubernetesObject) ID() string {
	return obj.GenericID.ID()
}

type GenericObjectID struct {
	Group     string
	Version   string
	Kind      string
	Namespace string
	Name      string
}

func genericObjectID(id, kind string) *GenericObjectID {
	if strings.TrimSpace(id) == "" {
		return nil
	}

	parts := strings.SplitN(strings.TrimSpace(id), "/", 4)

	genericID := &GenericObjectID{
		Group:   parts[0],
		Version: parts[1],
		Kind:    kind,
	}

	if len(parts) == 3 {
		genericID.Name = parts[2]
	} else if len(parts) == 4 {
		genericID.Namespace = parts[2]
		genericID.Name = parts[3]
	}

	return genericID
}

func (id *GenericObjectID) ApiVersion() string {
	return fmt.Sprintf("%s/%s", id.Group, id.Version)
}

func (id *GenericObjectID) Resource() string {
	return fmt.Sprintf("%ss", strings.ToLower(id.Kind))
}

func (id *GenericObjectID) ID() string {
	idString := make([]string, 1)
	idString[0] = id.ApiVersion()
	if id.Namespace != "" {
		idString = append(idString, id.Namespace)
	}

	return strings.Join(append(idString, id.Name), "/")
}

type CustomResourceClient struct {
	config *rest.Config
}

// NewForConfig creates a new AdmissionregistrationV1alpha1Client for the given config.
func NewCustomResourceClient(c *rest.Config) (*CustomResourceClient, error) {
	configShallowCopy := *c
	return &CustomResourceClient{&configShallowCopy}, nil
}

func (c *CustomResourceClient) setConfigDefaults(target *GenericObjectID) rest.Config {
	configShallowCopy := *c.config
	configShallowCopy.GroupVersion = &schema.GroupVersion{Group: target.Group, Version: target.Version}
	configShallowCopy.APIPath = "/apis"
	configShallowCopy.ContentType = runtime.ContentTypeJSON
	configShallowCopy.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	return configShallowCopy
}

func (c *CustomResourceClient) client(target *GenericObjectID) (rest.Interface, error) {
	config := c.setConfigDefaults(target)
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (result *GenericKubernetesObject) process() error {
	rawJSON := map[string]interface{}{}
	err := json.Unmarshal(result.Raw, &rawJSON)
	if err != nil {
		return err
	}
	result.RawJSON = rawJSON

	metadata, err := json.Marshal(result.RawJSON["metadata"])
	if err != nil {
		return err
	}

	objectMeta := metav1.ObjectMeta{}
	err = json.Unmarshal(metadata, &objectMeta)
	if err != nil {
		return err
	}

	result.ObjectMeta = &objectMeta
	result.GenericID = genericObjectID(result.RawJSON["apiVersion"].(string), result.RawJSON["kind"].(string))
	result.GenericID.Name = objectMeta.Name
	result.GenericID.Namespace = objectMeta.Namespace

	return nil
}

func (c *CustomResourceClient) request(target *GenericObjectID, request *rest.Request) *rest.Request {
	if target.Namespace != "" {
		return request.Resource(target.Resource()).Namespace(target.Namespace)
	}
	return request.Resource(target.Resource())
}

func (c *CustomResourceClient) Get(target *GenericObjectID) (result *GenericKubernetesObject, err error) {
	client, err := c.client(target)
	if err != nil {
		return nil, err
	}

	rawResult := runtime.Unknown{}
	err = c.request(target, client.Get()).
		Name(target.Name).
		Do().
		Into(&rawResult)

	if err != nil {
		return
	}

	result = &GenericKubernetesObject{Unknown: &rawResult}
	result.process()
	return
}

func (c *CustomResourceClient) Delete(target *GenericObjectID) (err error) {
	client, err := c.client(target)
	if err != nil {
		return err
	}

	return c.request(target, client.Delete()).
		Name(target.Name).
		Do().
		Error()
}

func (c *CustomResourceClient) Create(target *GenericObjectID, obj map[string]interface{}) (result *GenericKubernetesObject, err error) {
	client, err := c.client(target)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	rawResult := runtime.Unknown{}
	err = c.request(target, client.Post()).
		Body(raw).
		Do().
		Into(&rawResult)

	if err != nil {
		return
	}

	result = &GenericKubernetesObject{Unknown: &rawResult}
	result.process()
	return
}

func (c *CustomResourceClient) Update(target *GenericObjectID, ops PatchOperations) (result *GenericKubernetesObject, err error) {
	client, err := c.client(target)
	if err != nil {
		return nil, err
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating custom %s: %s", target.ID(), string(data))

	rawResult := runtime.Unknown{}
	err = c.request(target, client.Patch(pkgApi.JSONPatchType)).
		Name(target.Name).
		Body(data).
		Do().
		Into(&rawResult)

	if err != nil {
		return
	}

	result = &GenericKubernetesObject{Unknown: &rawResult}
	result.process()
	return
}
