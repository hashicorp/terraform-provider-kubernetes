package kubernetes

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

type patchClient struct {
	client     dynamic.ResourceInterface
	name       string
	apiPath    string
	patchPath  string
	id         string
	key        string
	value      string
	patchBytes []byte
}

func newDynamicClientFromMeta(meta interface{}) (dynamic.Interface, error) {
	kc, ok := meta.(kubeClientsets)
	if !ok {
		return nil, fmt.Errorf("unable to typecast meta to kubeClientsets")
	}

	return dynamic.NewForConfig(kc.config)
}

func newPatchClient(getFn func(key string) interface{}, client dynamic.Interface, patchPath, key, value string) (*patchClient, error) {
	apiVersion, ok := getFn("api_version").(string)
	if !ok || apiVersion == "" {
		return nil, fmt.Errorf("unable to extract api_version")
	}

	kind, ok := getFn("kind").(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("unable to extract kind")
	}

	namespaceScoped, ok := getFn("namespace_scoped").(bool)
	if !ok {
		return nil, fmt.Errorf("unable to extract namespace_scoped")
	}

	namespace, ok := getFn("namespace").(string)
	if !ok {
		return nil, fmt.Errorf("unable to extract namespace")
	}

	name, ok := getFn("name").(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("unable to extract name")
	}

	id := fmt.Sprintf("%s/%s/%s", apiVersion, kind, name)
	if namespaceScoped {
		id = fmt.Sprintf("%s/%s/%s/%s", apiVersion, namespace, kind, name)
	}
	id = strings.ToLower(id)

	gvk := schema.FromAPIVersionAndKind(apiVersion, kind)
	gvr := schema.GroupVersionResource{
		Group:    strings.ToLower(gvk.Group),
		Version:  strings.ToLower(gvk.Version),
		Resource: strings.ToLower(gvk.Kind),
	}

	resourceClient := client.Resource(gvr).Namespace(namespace)
	if !namespaceScoped {
		resourceClient = client.Resource(gvr)
	}

	return &patchClient{
		client:    resourceClient,
		name:      name,
		patchPath: patchPath,
		id:        id,
		key:       key,
		value:     value,
	}, nil
}

func (p *patchClient) newPatch(op PatchOperation) ([]byte, error) {
	ops := PatchOperations{op}
	patchBytes, err := ops.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal operations: %w", err)
	}

	return patchBytes, nil
}

func (p *patchClient) Create(ctx context.Context) (*unstructured.Unstructured, error) {
	body, err := p.newPatch(&AddOperation{Path: p.patchPath, Value: p.value})
	if err != nil {
		return nil, err
	}

	res, err := p.client.Patch(ctx, p.name, pkgApi.JSONPatchType, body, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *patchClient) Update(ctx context.Context) (*unstructured.Unstructured, error) {
	body, err := p.newPatch(&ReplaceOperation{Path: p.patchPath, Value: p.value})
	if err != nil {
		return nil, err
	}

	res, err := p.client.Patch(ctx, p.name, pkgApi.JSONPatchType, body, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *patchClient) Delete(ctx context.Context) (*unstructured.Unstructured, error) {
	body, err := p.newPatch(&RemoveOperation{Path: p.patchPath})
	if err != nil {
		return nil, err
	}

	res, err := p.client.Patch(ctx, p.name, pkgApi.JSONPatchType, body, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *patchClient) ReadResource(ctx context.Context) (*unstructured.Unstructured, error) {
	res, err := p.client.Get(ctx, p.name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return res, nil
}

type labelClient struct {
	patchClient *patchClient
}

func newLabelClient(getFn func(key string) interface{}, client dynamic.Interface) (*labelClient, error) {
	key, ok := getFn("label_key").(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("unable to extract label_key")
	}

	value, ok := getFn("label_value").(string)
	if !ok || value == "" {
		return nil, fmt.Errorf("unable to extract label_value")
	}

	patchPath := fmt.Sprintf("/metadata/labels/%s", key)

	patchClient, err := newPatchClient(getFn, client, patchPath, key, value)
	if err != nil {
		return nil, err
	}

	return &labelClient{
		patchClient: patchClient,
	}, nil
}

func (l *labelClient) getLabelValueFromResponse(res *unstructured.Unstructured) (string, error) {
	labels := res.GetLabels()

	value, ok := labels[l.patchClient.key]
	if !ok {
		return "", fmt.Errorf("label not found with key: %s", l.patchClient.key)
	}

	return value, nil
}

func (l *labelClient) Create(ctx context.Context) error {
	_, err := l.patchClient.Create(ctx)
	return err
}

func (l *labelClient) Update(ctx context.Context) error {
	_, err := l.patchClient.Update(ctx)
	return err
}

func (l *labelClient) Delete(ctx context.Context) error {
	_, err := l.patchClient.Delete(ctx)
	return err
}

func (l *labelClient) ReadResource(ctx context.Context) (*unstructured.Unstructured, error) {
	return l.patchClient.ReadResource(ctx)
}

func (l *labelClient) Read(ctx context.Context) (string, error) {
	res, err := l.patchClient.ReadResource(ctx)
	if err != nil {
		return "", err
	}

	return l.getLabelValueFromResponse(res)
}

func (l *labelClient) Id() string {
	return l.patchClient.id
}

func (l *labelClient) Key() string {
	return l.patchClient.key
}

func (l *labelClient) Value() string {
	return l.patchClient.value
}
