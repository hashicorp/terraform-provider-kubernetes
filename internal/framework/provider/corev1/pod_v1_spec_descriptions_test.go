// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"sort"
	"strings"
	"testing"

	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func TestPodV1SpecDescriptionDataParity(t *testing.T) {
	legacy := collectLegacyPodSpecDescriptionMetadata(t)

	if len(podSpecDescriptionByPath) != len(legacy) {
		t.Fatalf("static path count = %d, want %d", len(podSpecDescriptionByPath), len(legacy))
	}

	for path, want := range legacy {
		got, ok := podSpecDescriptionByPath[path]
		if !ok {
			t.Fatalf("missing static description entry for %q", path)
		}
		if got.description != want.description {
			t.Fatalf("%s: static description mismatch\nstatic: %q\nlegacy: %q", path, got.description, want.description)
		}
		if got.minItems != want.minItems || got.maxItems != want.maxItems {
			t.Fatalf("%s: static cardinality mismatch: static (%d,%d), legacy (%d,%d)", path, got.minItems, got.maxItems, want.minItems, want.maxItems)
		}
	}

	for path := range podSpecDescriptionByPath {
		if _, ok := legacy[path]; !ok {
			t.Fatalf("unexpected static description entry for %q", path)
		}
	}
}

func TestPodV1SpecWithDescriptionsCoverage(t *testing.T) {
	legacy := collectLegacyPodSpecDescriptionMetadata(t)

	paths := map[string]string{}
	collectFrameworkBlockDescriptions("spec", podSpecWithDescriptions(podSpecObject()), paths)

	for path, source := range legacy {
		if path == "spec" {
			continue
		}
		got, ok := paths[path]
		if !ok {
			parentPath, hasParent := podSpecParentPath(path)
			if !hasParent {
				t.Fatalf("framework schema path %q is missing", path)
			}
			parentDescription, parentFound := paths[parentPath]
			if !parentFound || !strings.Contains(parentDescription, source.description) {
				t.Fatalf("framework schema path %q is missing", path)
			}
			continue
		}
		if !strings.HasPrefix(got, source.description) {
			t.Fatalf("%s: framework description mismatch\nframework: %q\nlegacy: %q", path, got, source.description)
		}
		if hint := podSpecCardinalityHint(source.minItems, source.maxItems); hint != "" && !strings.Contains(got, hint) {
			t.Fatalf("%s: framework description missing cardinality hint %q", path, hint)
		}
	}
}

func TestPodV1SpecWithDescriptionsPreservesNativeSupplementalText(t *testing.T) {
	paths := map[string]string{}
	collectFrameworkBlockDescriptions("spec", podSpecWithDescriptions(podSpecObject()), paths)

	resourcesDescription := mustDescription(t, paths, "spec.container.resources")
	if !strings.HasPrefix(resourcesDescription, podSpecDescriptionByPath["spec.container.resources"].description) {
		t.Fatalf("spec.container.resources: expected legacy description prefix, got %q", resourcesDescription)
	}
	for _, snippet := range []string{
		"Omit or use null to allow API defaults",
		"Use [{}] to leave limits and requests unset for API defaults.",
		"An empty list is not omission.",
		podSpecCardinalityHint(podSpecDescriptionByPath["spec.container.resources"].minItems, podSpecDescriptionByPath["spec.container.resources"].maxItems),
	} {
		if !strings.Contains(resourcesDescription, snippet) {
			t.Fatalf("spec.container.resources: expected description to contain %q\nactual: %q", snippet, resourcesDescription)
		}
	}

	imagePullSecretsDescription := mustDescription(t, paths, "spec.image_pull_secrets")
	if !strings.HasPrefix(imagePullSecretsDescription, podSpecDescriptionByPath["spec.image_pull_secrets"].description) {
		t.Fatalf("spec.image_pull_secrets: expected legacy description prefix, got %q", imagePullSecretsDescription)
	}
	for _, snippet := range []string{
		"Omit or use null to retain API-populated references.",
		"an empty list is not omission.",
		"Nested object fields: `name`: Name of the referent.",
		"Each object must set `name`; `{}` is not valid.",
	} {
		if !strings.Contains(imagePullSecretsDescription, snippet) {
			t.Fatalf("spec.image_pull_secrets: expected description to contain %q\nactual: %q", snippet, imagePullSecretsDescription)
		}
	}

	readinessGateDescription := mustDescription(t, paths, "spec.readiness_gate")
	for _, snippet := range []string{
		"Nested object fields: `condition_type`:",
		"matching type.",
		"Each object must set `condition_type`; `{}` is not valid.",
	} {
		if !strings.Contains(readinessGateDescription, snippet) {
			t.Fatalf("spec.readiness_gate: expected description to contain %q\nactual: %q", snippet, readinessGateDescription)
		}
	}
}

func podSpecParentPath(path string) (string, bool) {
	index := strings.LastIndex(path, ".")
	if index < 0 {
		return "", false
	}
	return path[:index], true
}

func mustDescription(t *testing.T, paths map[string]string, path string) string {
	t.Helper()
	description, ok := paths[path]
	if !ok {
		t.Fatalf("missing description for path %q", path)
	}
	return description
}

func collectLegacyPodSpecDescriptionMetadata(t *testing.T) map[string]podSpecDescriptionMetadata {
	t.Helper()
	resource := kubernetes.Provider().ResourcesMap["kubernetes_pod"]
	spec, ok := resource.Schema["spec"]
	if !ok {
		t.Fatal("legacy pod schema has no spec")
	}
	metadata := map[string]podSpecDescriptionMetadata{}
	collectLegacySchemaDescriptionMetadata("spec", spec, metadata)
	return metadata
}

func collectLegacySchemaDescriptionMetadata(path string, value *sdkschema.Schema, metadata map[string]podSpecDescriptionMetadata) {
	if strings.TrimSpace(value.Description) != "" {
		metadata[path] = podSpecDescriptionMetadata{
			description: value.Description,
			minItems:    value.MinItems,
			maxItems:    value.MaxItems,
		}
	}

	switch nested := value.Elem.(type) {
	case *sdkschema.Resource:
		names := make([]string, 0, len(nested.Schema))
		for name := range nested.Schema {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			collectLegacySchemaDescriptionMetadata(path+"."+name, nested.Schema[name], metadata)
		}
	case *sdkschema.Schema:
		collectLegacySchemaDescriptionMetadata(path+".*", nested, metadata)
	}
}

func collectFrameworkBlockDescriptions(path string, object fwschema.NestedBlockObject, paths map[string]string) {
	for name, attribute := range object.Attributes {
		collectFrameworkAttributeDescriptions(path+"."+name, attribute, paths)
	}
	for name, block := range object.Blocks {
		collectFrameworkNestedBlockDescriptions(path+"."+name, block, paths)
	}
}

func collectFrameworkAttributeDescriptions(path string, attribute fwschema.Attribute, paths map[string]string) {
	switch value := attribute.(type) {
	case fwschema.BoolAttribute:
		paths[path] = value.Description
	case fwschema.Int64Attribute:
		paths[path] = value.Description
	case fwschema.ListAttribute:
		paths[path] = value.Description
	case fwschema.ListNestedAttribute:
		paths[path] = value.Description
		collectFrameworkAttributeObjectDescriptions(path, value.NestedObject, paths)
	case fwschema.MapAttribute:
		paths[path] = value.Description
	case fwschema.MapNestedAttribute:
		paths[path] = value.Description
		collectFrameworkAttributeObjectDescriptions(path, value.NestedObject, paths)
	case fwschema.ObjectAttribute:
		paths[path] = value.Description
	case fwschema.SetAttribute:
		paths[path] = value.Description
	case fwschema.SetNestedAttribute:
		paths[path] = value.Description
		collectFrameworkAttributeObjectDescriptions(path, value.NestedObject, paths)
	case fwschema.SingleNestedAttribute:
		paths[path] = value.Description
		collectFrameworkAttributeObjectDescriptions(path, fwschema.NestedAttributeObject{Attributes: value.Attributes}, paths)
	case fwschema.StringAttribute:
		paths[path] = value.Description
	}
}

func collectFrameworkAttributeObjectDescriptions(path string, object fwschema.NestedAttributeObject, paths map[string]string) {
	for name, attribute := range object.Attributes {
		collectFrameworkAttributeDescriptions(path+"."+name, attribute, paths)
	}
}

func collectFrameworkNestedBlockDescriptions(path string, block fwschema.Block, paths map[string]string) {
	switch value := block.(type) {
	case fwschema.ListNestedBlock:
		paths[path] = value.Description
		collectFrameworkBlockDescriptions(path, value.NestedObject, paths)
	case fwschema.SetNestedBlock:
		paths[path] = value.Description
		collectFrameworkBlockDescriptions(path, value.NestedObject, paths)
	case fwschema.SingleNestedBlock:
		paths[path] = value.Description
		collectFrameworkBlockDescriptions(path, fwschema.NestedBlockObject{Attributes: value.Attributes, Blocks: value.Blocks}, paths)
	}
}
