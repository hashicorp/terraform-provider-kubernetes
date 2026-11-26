// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/csaupgrade"
)

// identity names one object. Namespace is empty for cluster-scoped kinds.
type identity struct {
	APIVersion, Kind, Namespace, Name string
}

func (id identity) group() string {
	gv, _ := schema.ParseGroupVersion(id.APIVersion)
	return gv.Group
}

func (id identity) String() string {
	name := id.Name
	if id.Namespace != "" {
		name = id.Namespace + "/" + name
	}
	return fmt.Sprintf("%s %s %q", id.APIVersion, id.Kind, name)
}

// unknownKindError reports a kind the API server does not serve, usually a
// custom resource whose definition is not installed.
type unknownKindError struct {
	apiVersion, kind string
}

func (e unknownKindError) Error() string {
	return fmt.Sprintf("the cluster does not serve kind %q in API version %q. If it is a custom resource, its CustomResourceDefinition must be installed first", e.kind, e.apiVersion)
}

// served caches the resources of each API version. A provider process serves
// one cluster.
var served sync.Map

func lookupKind(disco discovery.DiscoveryInterface, apiVersion, kind string) (metav1.APIResource, error) {
	find := func(list *metav1.APIResourceList) (metav1.APIResource, bool) {
		for _, api := range list.APIResources {
			if api.Kind == kind && !strings.Contains(api.Name, "/") {
				return api, true
			}
		}
		return metav1.APIResource{}, false
	}
	if cached, ok := served.Load(apiVersion); ok {
		if api, found := find(cached.(*metav1.APIResourceList)); found {
			return api, nil
		}
	}
	list, err := disco.ServerResourcesForGroupVersion(apiVersion)
	if apierrors.IsNotFound(err) {
		served.Delete(apiVersion)
		return metav1.APIResource{}, unknownKindError{apiVersion, kind}
	}
	if err != nil {
		return metav1.APIResource{}, fmt.Errorf("discovering API version %q: %w", apiVersion, err)
	}
	served.Store(apiVersion, list)
	if api, found := find(list); found {
		return api, nil
	}
	return metav1.APIResource{}, unknownKindError{apiVersion, kind}
}

// servedVersion finds the kind in another version of its API group. An object
// whose API version the cluster stopped serving still exists in the versions
// that remain.
func servedVersion(disco discovery.DiscoveryInterface, id identity) (identity, metav1.APIResource, error) {
	groups, err := disco.ServerGroups()
	if err != nil {
		return id, metav1.APIResource{}, fmt.Errorf("discovering API groups: %w", err)
	}
	for _, group := range groups.Groups {
		if group.Name != id.group() {
			continue
		}
		for _, version := range append([]metav1.GroupVersionForDiscovery{group.PreferredVersion}, group.Versions...) {
			api, err := lookupKind(disco, version.GroupVersion, id.Kind)
			if _, unknown := err.(unknownKindError); err != nil && !unknown {
				// Not an answer: absence must be confirmed, not assumed.
				return id, metav1.APIResource{}, err
			}
			if err == nil {
				id.APIVersion = version.GroupVersion
				return id, api, nil
			}
		}
	}
	return id, metav1.APIResource{}, unknownKindError{id.APIVersion, id.Kind}
}

// client returns the API client for an object and its identity with the
// namespace the API will use: "default" when a namespaced object names none.
// With anyVersion, an object whose API version is no longer served is reached
// through a version that is, and the identity names that version.
func (r *ManifestV2) client(id identity, anyVersion bool) (dynamic.ResourceInterface, identity, error) {
	clients, _, diags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	if diags.HasError() {
		return nil, id, fmt.Errorf("%s", diags[0].Detail())
	}
	disco, err := clients.DiscoveryClient()
	if err != nil {
		return nil, id, err
	}
	dyn, err := clients.DynamicClient()
	if err != nil {
		return nil, id, err
	}
	if disco == nil || dyn == nil {
		return nil, id, fmt.Errorf("the provider has no cluster configuration")
	}
	gv, err := schema.ParseGroupVersion(id.APIVersion)
	if err != nil {
		return nil, id, err
	}
	api, err := lookupKind(disco, id.APIVersion, id.Kind)
	if _, unknown := err.(unknownKindError); unknown && anyVersion {
		if id, api, err = servedVersion(disco, id); err == nil {
			gv, err = schema.ParseGroupVersion(id.APIVersion)
		}
	}
	if err != nil {
		return nil, id, err
	}
	if !api.Namespaced {
		id.Namespace = ""
		return dyn.Resource(gv.WithResource(api.Name)), id, nil
	}
	if id.Namespace == "" {
		id.Namespace = metav1.NamespaceDefault
	}
	return dyn.Resource(gv.WithResource(api.Name)).Namespace(id.Namespace), id, nil
}

// readOnlyMetadata is set by the API server and never applied.
var readOnlyMetadata = []string{
	"creationTimestamp", "deletionGracePeriodSeconds", "deletionTimestamp", "generation",
	"managedFields", "resourceVersion", "selfLink", "uid",
}

// applicable returns the part of a manifest that is sent to the API.
func applicable(manifest map[string]any, id identity) *unstructured.Unstructured {
	obj := (&unstructured.Unstructured{Object: manifest}).DeepCopy()
	delete(obj.Object, "status")
	for _, field := range readOnlyMetadata {
		unstructured.RemoveNestedField(obj.Object, "metadata", field)
	}
	obj.SetNamespace(id.Namespace)
	return obj
}

func apply(ctx context.Context, client dynamic.ResourceInterface, obj *unstructured.Unstructured, manager string, force bool) (*unstructured.Unstructured, error) {
	body, err := json.Marshal(obj.Object)
	if err != nil {
		return nil, err
	}
	return client.Patch(ctx, obj.GetName(), k8stypes.ApplyPatchType, body, metav1.PatchOptions{
		FieldManager:    manager,
		Force:           &force,
		FieldValidation: metav1.FieldValidationStrict,
	})
}

// release gives up everything a field manager owns in an object without
// changing the fields other managers also own.
func release(ctx context.Context, client dynamic.ResourceInterface, id identity, manager string) error {
	obj := applicable(map[string]any{
		"apiVersion": id.APIVersion, "kind": id.Kind, "metadata": map[string]any{"name": id.Name},
	}, id)
	_, err := apply(ctx, client, obj, manager, false)
	return err
}

// clientSideManagers are the field managers of clients that write whole
// objects without server-side apply: kubectl's client-side apply, and
// Terraform providers that identify as HashiCorp, which include the other
// resources of this provider.
var clientSideManagers = sets.New("kubectl-client-side-apply", "HashiCorp")

// adopt moves the fields those clients wrote to the manager the first time it
// applies an object, as kubectl does when it switches an object to server-side
// apply. Without it, those fields would conflict with the apply and would
// stay behind when they are removed from the manifest.
func adopt(ctx context.Context, client dynamic.ResourceInterface, live *unstructured.Unstructured, manager string) error {
	if len(managedFields(live, appliedBy(manager))) > 0 {
		return nil
	}
	patch, err := csaupgrade.UpgradeManagedFieldsPatch(live, clientSideManagers, manager)
	if err != nil || patch == nil {
		return err
	}
	_, err = client.Patch(ctx, live.GetName(), k8stypes.JSONPatchType, patch, metav1.PatchOptions{})
	return err
}

// retryWhile runs op again while its error is one that a resource applied in
// parallel may be about to resolve, for a short time only: a mistyped namespace
// must not wait for the whole create timeout.
func retryWhile(ctx context.Context, transient func(error) bool, op func() error) error {
	deadline := time.Now().Add(time.Minute)
	for {
		err := op()
		if err == nil || !transient(err) || time.Now().After(deadline) {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(2 * time.Second):
		}
	}
}

// awaitsDependency reports errors of an apply that ran before its namespace
// or its CustomResourceDefinition was ready.
func awaitsDependency(err error) bool {
	if _, ok := err.(unknownKindError); ok {
		return true
	}
	status, ok := err.(apierrors.APIStatus)
	if !ok || !apierrors.IsNotFound(err) {
		return false
	}
	details := status.Status().Details
	// A missing namespace names itself. A kind that discovery already lists
	// but that is not served yet answers without details.
	return details == nil || details.Kind == "namespaces" || details.Name == ""
}
