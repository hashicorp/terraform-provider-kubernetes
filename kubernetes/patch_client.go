package kubernetes

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type patchClient struct {
	kubeClient *kubernetes.Clientset
	apiPath    string
	patchPath  string
	id         string
	key        string
	value      string
	patchBytes []byte
}

func newPatchClient(getFn func(key string) interface{}, kubeClient *kubernetes.Clientset, patchPath, key, value string) (*patchClient, error) {
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

	apiPrefix := "apis"
	if apiVersion == "v1" {
		apiPrefix = "api"
	}

	apiPath := fmt.Sprintf("/%s/%s/%s/%s", apiPrefix, apiVersion, kind, name)
	if namespaceScoped {
		apiPath = fmt.Sprintf("/%s/%s/namespaces/%s/%s/%s", apiPrefix, apiVersion, namespace, kind, name)
	}
	apiPath = strings.ToLower(apiPath)

	return &patchClient{
		kubeClient: kubeClient,
		apiPath:    apiPath,
		patchPath:  patchPath,
		id:         apiPath,
		key:        key,
		value:      value,
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

func (p *patchClient) Create(ctx context.Context) (rest.Result, error) {
	patch := &AddOperation{
		Path:  p.patchPath,
		Value: p.value,
	}
	body, err := p.newPatch(patch)
	if err != nil {
		return rest.Result{}, err
	}
	req := p.kubeClient.RESTClient().Patch(pkgApi.JSONPatchType).Body(body).AbsPath(p.apiPath)
	res := req.Do(ctx)
	if res.Error() != nil {
		return rest.Result{}, res.Error()
	}

	return res, nil
}

func (p *patchClient) Update(ctx context.Context) (rest.Result, error) {
	patch := &ReplaceOperation{
		Path:  p.patchPath,
		Value: p.value,
	}
	body, err := p.newPatch(patch)
	if err != nil {
		return rest.Result{}, err
	}
	req := p.kubeClient.RESTClient().Patch(pkgApi.JSONPatchType).Body(body).AbsPath(p.apiPath)
	res := req.Do(ctx)
	if res.Error() != nil {
		return rest.Result{}, res.Error()
	}

	return res, nil
}

func (p *patchClient) Delete(ctx context.Context) (rest.Result, error) {
	patch := &RemoveOperation{
		Path: p.patchPath,
	}
	body, err := p.newPatch(patch)
	if err != nil {
		return rest.Result{}, err
	}
	req := p.kubeClient.RESTClient().Patch(pkgApi.JSONPatchType).Body(body).AbsPath(p.apiPath)
	res := req.Do(ctx)
	if res.Error() != nil {
		return rest.Result{}, res.Error()
	}

	return res, nil
}

func (p *patchClient) ReadResource(ctx context.Context) (rest.Result, error) {
	req := p.kubeClient.RESTClient().Get().AbsPath(p.apiPath)
	res := req.Do(ctx)
	if res.Error() != nil {
		return rest.Result{}, res.Error()
	}

	return res, nil
}

type labelClient struct {
	patchClient *patchClient
}

func newLabelClient(getFn func(key string) interface{}, kubeClient *kubernetes.Clientset) (*labelClient, error) {
	key, ok := getFn("label_key").(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("unable to extract label_key")
	}

	value, ok := getFn("label_value").(string)
	if !ok || value == "" {
		return nil, fmt.Errorf("unable to extract label_value")
	}

	patchPath := fmt.Sprintf("/metadata/labels/%s", key)

	patchClient, err := newPatchClient(getFn, kubeClient, patchPath, key, value)
	if err != nil {
		return nil, err
	}

	return &labelClient{
		patchClient: patchClient,
	}, nil
}

func (l *labelClient) getLabelsFromResponse(res rest.Result) (map[string]string, error) {
	obj := metav1.PartialObjectMetadata{}
	err := res.Into(&obj)
	if err != nil {
		return nil, err
	}

	return obj.Labels, nil
}

func (l *labelClient) getLabelValueFromResponse(res rest.Result) (string, error) {
	labels, err := l.getLabelsFromResponse(res)
	if err != nil {
		return "", err
	}

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

func (l *labelClient) ReadResource(ctx context.Context) (rest.Result, error) {
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
