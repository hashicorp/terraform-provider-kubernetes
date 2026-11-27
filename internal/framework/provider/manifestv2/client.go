// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	if !customGroup(e.apiVersion) {
		return fmt.Sprintf("the cluster does not serve kind %q in API version %q. Check apiVersion and kind", e.kind, e.apiVersion)
	}
	return fmt.Sprintf("the cluster does not serve kind %q in API version %q. If it is a custom resource, its CustomResourceDefinition must be installed first", e.kind, e.apiVersion)
}

// customGroup reports whether a CustomResourceDefinition can add kinds to the
// API group of an API version. Its group must contain a dot: core, apps,
// batch and the other groups without one are the API server's own.
func customGroup(apiVersion string) bool {
	gv, err := schema.ParseGroupVersion(apiVersion)
	return err == nil && strings.Contains(gv.Group, ".")
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

// createOnly is the resource version an apply names to create an object and
// never update one: no existing object has it, so the API server answers an
// apply to an existing object with a conflict, and ignores it on creation.
const createOnly = "1"

// apply sends the object with server-side apply. A value the API rejects is
// tried again converted to the type the kind's schema gives the field, as
// kubernetes_manifest converted it: a number or bool where a string belongs.
func (r *ManifestV2) apply(ctx context.Context, client dynamic.ResourceInterface, obj *unstructured.Unstructured, manager string, force bool) (*unstructured.Unstructured, error) {
	applied, err := applyObject(ctx, client, obj, manager, force)
	// The API server answers a value of the wrong type in a built-in kind
	// with an internal error, in a custom resource with an invalid one.
	var status apierrors.APIStatus
	if !errors.As(err, &status) {
		return applied, err
	}
	if code := status.Status().Code; code != http.StatusBadRequest && code != http.StatusUnprocessableEntity && code != http.StatusInternalServerError {
		return applied, err
	}
	if converted, ok := r.coerced(obj); ok {
		return applyObject(ctx, client, converted, manager, force)
	}
	return applied, err
}

func applyObject(ctx context.Context, client dynamic.ResourceInterface, obj *unstructured.Unstructured, manager string, force bool) (*unstructured.Unstructured, error) {
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

// clientSideManagers are the field managers of clients that write whole
// objects without server-side apply: kubectl's client-side apply, and
// Terraform providers that identify as HashiCorp, which include the other
// resources of this provider.
var clientSideManagers = sets.New("kubectl-client-side-apply", "HashiCorp")

func clientSide(entry metav1.ManagedFieldsEntry) bool {
	return clientSideManagers.Has(entry.Manager) && entry.Operation == metav1.ManagedFieldsOperationUpdate && entry.Subresource == ""
}

// handover names entries whose fields move to the manager, as far as a
// manifest sets them.
type handover struct {
	from   func(metav1.ManagedFieldsEntry) bool
	within map[string]any
}

// takeOver moves fields to the manager before it applies, as kubectl does when
// it switches an object to server-side apply: fields that a client-side
// manager wrote, or that this resource owned under a previous manager name.
// Only the fields the given manifest sets move. The others stay with their
// manager, so the apply neither conflicts on the moved ones nor removes the
// rest. The patch names the resource version, so a concurrent change makes it
// fail instead of overwriting the change. It returns the object as it is
// afterwards.
func takeOver(ctx context.Context, client dynamic.ResourceInterface, live *unstructured.Unstructured, manager string, sources ...handover) (*unstructured.Unstructured, error) {
	entries := live.GetManagedFields()
	kept := make([]metav1.ManagedFieldsEntry, 0, len(entries)+1)
	taken := map[string]any{}
	target := -1
	for _, entry := range entries {
		var source *handover
		for i := range sources {
			if sources[i].from(entry) {
				source = &sources[i]
			}
		}
		var fields map[string]any
		if appliedBy(manager)(entry) {
			target = len(kept)
		} else if source != nil && entry.APIVersion == live.GetAPIVersion() && entry.FieldsV1 != nil &&
			json.Unmarshal(entry.FieldsV1.GetRawBytes(), &fields) == nil {
			in, out := split(fields, source.within)
			mergeFields(taken, in)
			if len(out) == 0 {
				continue
			}
			raw, _ := json.Marshal(out)
			entry.FieldsV1 = &metav1.FieldsV1{Raw: raw}
		}
		kept = append(kept, entry)
	}
	if len(taken) == 0 {
		return live, nil
	}
	owned := map[string]any{}
	if target < 0 {
		now := metav1.Now()
		kept = append(kept, metav1.ManagedFieldsEntry{
			Manager: manager, Operation: metav1.ManagedFieldsOperationApply, APIVersion: live.GetAPIVersion(),
			Time: &now, FieldsType: "FieldsV1",
		})
		target = len(kept) - 1
	} else if kept[target].FieldsV1 != nil {
		_ = json.Unmarshal(kept[target].FieldsV1.GetRawBytes(), &owned)
	}
	mergeFields(owned, taken)
	raw, _ := json.Marshal(owned)
	kept[target].FieldsV1 = &metav1.FieldsV1{Raw: raw}
	patch, err := json.Marshal([]map[string]any{
		{"op": "replace", "path": "/metadata/managedFields", "value": kept},
		// "replace", not "test": etcd then rejects a stale version with a conflict.
		{"op": "replace", "path": "/metadata/resourceVersion", "value": live.GetResourceVersion()},
	})
	if err != nil {
		return nil, err
	}
	return client.Patch(ctx, live.GetName(), k8stypes.JSONPatchType, patch, metav1.PatchOptions{})
}

// missingSince records when an apply first found a namespace or a kind
// missing. An object that starts after one has been missing for a minute does
// not wait for it: while objects wait, they hold Terraform's parallel
// operations, and the one that creates the namespace or kind may be queued
// behind them.
var missingSince sync.Map

// retryWhile runs op again while it fails for a namespace or kind that a
// resource applied in parallel may be about to create, for a minute only: a
// mistyped namespace must not wait for the whole create timeout.
func retryWhile(ctx context.Context, id identity, op func() error) error {
	start, delay, waited := time.Now(), time.Second, ""
	for {
		err := op()
		dependency, waits := awaitsDependency(err)
		if dependency == "resource" {
			dependency = "kind " + id.APIVersion + " " + id.Kind
		}
		if !waits {
			// What this call waited for has arrived.
			if err == nil && waited != "" {
				missingSince.Delete(waited)
			}
			return err
		}
		waited = dependency
		first, _ := missingSince.LoadOrStore(dependency, start)
		if time.Since(start) > time.Minute || start.Sub(first.(time.Time)) > time.Minute {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(delay):
		}
		delay = min(2*delay, 4*time.Second)
	}
}

// awaitsDependency reports errors of an apply that ran before its namespace
// or its CustomResourceDefinition was ready, and names what is missing.
func awaitsDependency(err error) (string, bool) {
	var kind unknownKindError
	if errors.As(err, &kind) {
		return "kind " + kind.apiVersion + " " + kind.kind, customGroup(kind.apiVersion)
	}
	var status apierrors.APIStatus
	if !errors.As(err, &status) || !apierrors.IsNotFound(err) {
		return "", false
	}
	// A missing namespace names itself. A kind that discovery already lists
	// but that is not served yet answers without details.
	switch details := status.Status().Details; {
	case details == nil || details.Name == "":
		return "resource", true
	case details.Kind == "namespaces":
		return "namespace " + details.Name, true
	}
	return "", false
}

// uncertain reports an error after which a request may have taken effect: its
// response was lost or cut off.
func uncertain(err error) bool {
	var status apierrors.APIStatus
	return err != nil && (!errors.As(err, &status) || status.Status().Code >= http.StatusInternalServerError)
}

// transient reports an error that waiting may resolve: the connection, the API
// server being busy or restarting, or a watch that started too late.
func transient(err error) bool {
	var status apierrors.APIStatus
	if !errors.As(err, &status) {
		return true
	}
	code := status.Status().Code
	return code == http.StatusGone || code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
}
