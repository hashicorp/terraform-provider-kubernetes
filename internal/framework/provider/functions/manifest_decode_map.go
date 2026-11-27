// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package functions

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

var _ function.Function = ManifestDecodeMapFunction{}

func NewManifestDecodeMapFunction() function.Function {
	return &ManifestDecodeMapFunction{}
}

type ManifestDecodeMapFunction struct{}

func (f ManifestDecodeMapFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "manifest_decode_map"
}

func (f ManifestDecodeMapFunction) Definition(_ context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Decode Kubernetes YAML manifests into a map keyed by object",
		MarkdownDescription: "Decodes one or more YAML texts, each holding any number of Kubernetes objects, and returns the objects keyed by `Kind[.group]/[namespace/]name`, " +
			"for example `Deployment.apps/default/web` or `Namespace/prod`. The key does not depend on the file or the position of an object, nor on its API version, " +
			"so it is a stable `for_each` key. Objects of a `List` kind are replaced by their items. An object that appears twice is an error.",
		VariadicParameter: function.StringParameter{
			Name:                "manifests",
			MarkdownDescription: "YAML texts of Kubernetes manifests",
		},
		Return: function.DynamicReturn{},
	}
}

func (f ManifestDecodeMapFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var manifests []string
	resp.Error = req.Arguments.Get(ctx, &manifests)
	if resp.Error != nil {
		return
	}
	objects := map[string]attr.Value{}
	objectTypes := map[string]attr.Type{}
	// A namespaced object without a namespace goes to "default", so the two
	// spellings name one object although their keys differ.
	effective := map[string]string{}
	for i, manifest := range manifests {
		documents, err := decodeDocuments(ctx, manifest)
		if err != nil {
			resp.Error = function.NewArgumentFuncError(int64(i), err.Error())
			return
		}
		for _, object := range expandLists(ctx, documents) {
			key, same, err := objectKey(object)
			if err == nil && objects[key] != nil {
				err = fmt.Errorf("%s is defined more than once", key)
			}
			if err == nil && effective[same] != "" {
				err = fmt.Errorf("%s and %s are the same object when the kind is namespaced: an object without a namespace is in default", effective[same], key)
			}
			if err != nil {
				resp.Error = function.NewArgumentFuncError(int64(i), err.Error())
				return
			}
			objects[key], objectTypes[key], effective[same] = object, object.Type(ctx), key
		}
	}
	result, diags := types.ObjectValue(objectTypes, objects)
	if diags.HasError() {
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		return
	}
	dynamic := types.DynamicValue(result)
	resp.Error = resp.Result.Set(ctx, &dynamic)
}

// decodeDocuments reads the documents of a YAML text the way kubectl does:
// split at document markers only, with numbers kept exact.
func decodeDocuments(ctx context.Context, manifest string) ([]attr.Value, error) {
	var out []attr.Value
	reader := k8syaml.NewYAMLReader(bufio.NewReader(strings.NewReader(manifest)))
	for {
		document, err := reader.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		// Strict: a duplicate key is an error, not the last one winning.
		converted, err := yaml.YAMLToJSONStrict(document)
		if err != nil {
			return nil, fmt.Errorf("invalid YAML document: %w", err)
		}
		var data any
		decoder := json.NewDecoder(bytes.NewReader(converted))
		decoder.UseNumber()
		if err := decoder.Decode(&data); err != nil {
			return nil, err
		}
		if data == nil {
			continue
		}
		if _, isObject := data.(map[string]any); !isObject {
			return nil, fmt.Errorf("a document is not a Kubernetes object")
		}
		value, diags := decodeScalar(ctx, data)
		if diags.HasError() {
			return nil, fmt.Errorf("%s", diags[0].Detail())
		}
		out = append(out, value)
	}
}

// expandLists replaces every list by its items, as kubectl does when it reads
// a file. A list is an object of a List kind that has items and no name: a
// named object is itself, whatever its kind is called. The items of a typed
// list, such as a PodList, take their kind and API version from the list when
// they have none.
func expandLists(ctx context.Context, objects []attr.Value) []attr.Value {
	var out []attr.Value
	for _, object := range objects {
		items, isList := field(object, "items").(types.Tuple)
		kind, _ := field(object, "kind").(types.String)
		itemKind, typed := strings.CutSuffix(kind.ValueString(), "List")
		if !isList || !typed || field(field(object, "metadata"), "name") != nil {
			out = append(out, object)
			continue
		}
		listVersion, versioned := field(object, "apiVersion").(types.String)
		for _, item := range expandLists(ctx, items.Elements()) {
			members, isObject := item.(types.Object)
			if isObject && versioned && itemKind != "" && field(item, "kind") == nil && field(item, "apiVersion") == nil {
				attrs := members.Attributes()
				attrs["kind"], attrs["apiVersion"] = types.StringValue(itemKind), listVersion
				attrTypes := map[string]attr.Type{}
				for name, value := range attrs {
					attrTypes[name] = value.Type(ctx)
				}
				item, _ = types.ObjectValue(attrTypes, attrs)
			}
			out = append(out, item)
		}
	}
	return out
}

// objectKey identifies an object by what cannot change without making it a
// different object. The API version is left out: an object stays the same
// when its manifest moves to a newer version of its kind. The second result
// is the key with the namespace a namespaced object ends up in.
func objectKey(object attr.Value) (string, string, error) {
	text := func(names ...string) string {
		value := object
		for _, name := range names {
			value = field(value, name)
		}
		s, _ := value.(types.String)
		return s.ValueString()
	}
	apiVersion, kind, name, namespace := text("apiVersion"), text("kind"), text("metadata", "name"), text("metadata", "namespace")
	if apiVersion == "" || kind == "" || name == "" {
		return "", "", fmt.Errorf("every object needs apiVersion, kind and metadata.name; found apiVersion %q, kind %q, name %q", apiVersion, kind, name)
	}
	key := kind
	if group, _, grouped := strings.Cut(apiVersion, "/"); grouped {
		key += "." + group
	}
	if namespace == "" {
		return key + "/" + name, key + "/default/" + name, nil
	}
	return key + "/" + namespace + "/" + name, key + "/" + namespace + "/" + name, nil
}

func field(value attr.Value, name string) attr.Value {
	if object, ok := value.(types.Object); ok {
		return object.Attributes()[name]
	}
	return nil
}
