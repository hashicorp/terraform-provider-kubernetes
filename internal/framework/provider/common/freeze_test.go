// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common_test

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/appsv1"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/batchv1"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
)

// An unfrozen pod-template schema makes every plan and apply about ten times
// slower. Freezing must not change the schema type or the GetProviderSchema
// response, and must not modify its input.
func TestPodTemplateSchemasFrozen(t *testing.T) {
	for _, newResource := range []func() resource.Resource{
		appsv1.NewDeploymentV1, appsv1.NewDaemonSetV1, appsv1.NewStatefulSetV1,
		corev1.NewPodV1, batchv1.NewJobV1, batchv1.NewCronJobV1,
	} {
		r := newResource()
		var md resource.MetadataResponse
		var resp resource.SchemaResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "kubernetes"}, &md)
		r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
		frozen, thawed := resp.Schema, resp.Schema
		var unfrozen []string
		thawed.Blocks, unfrozen = thaw("", frozen.Blocks)
		if len(unfrozen) > 0 {
			t.Errorf("%s: %d blocks without a precomputed type, first %s", md.TypeName, len(unfrozen), unfrozen[0])
			continue
		}
		if !frozen.Type().Equal(thawed.Type()) {
			t.Errorf("%s: precomputed types differ from the derived ones", md.TypeName)
		}
		if !bytes.Equal(protocolSchema(t, frozen), protocolSchema(t, thawed)) {
			t.Errorf("%s: freezing changed the GetProviderSchema response", md.TypeName)
		}
		common.FreezeSchema(thawed)
		if _, unfrozen = thaw("", thawed.Blocks); len(unfrozen) == 0 {
			t.Errorf("%s: FreezeSchema modified its input", md.TypeName)
		}
	}
}

// thaw drops the block types FreezeSchema precomputes (timeouts keeps its
// own) and returns the sorted paths of blocks that had none.
func thaw(prefix string, blocks map[string]schema.Block) (map[string]schema.Block, []string) {
	out, unfrozen := make(map[string]schema.Block, len(blocks)), []string(nil)
	for name, block := range blocks {
		frozen, nested := true, []string(nil)
		switch b := block.(type) {
		case schema.ListNestedBlock:
			frozen = b.CustomType != nil && b.NestedObject.CustomType != nil
			b.CustomType, b.NestedObject.CustomType = nil, nil
			b.NestedObject.Blocks, nested = thaw(prefix+name+".", b.NestedObject.Blocks)
			block = b
		case schema.SetNestedBlock:
			frozen = b.CustomType != nil && b.NestedObject.CustomType != nil
			b.CustomType, b.NestedObject.CustomType = nil, nil
			b.NestedObject.Blocks, nested = thaw(prefix+name+".", b.NestedObject.Blocks)
			block = b
		case schema.SingleNestedBlock:
			if frozen = b.CustomType != nil; frozen {
				if _, ok := b.CustomType.(types.ObjectType); ok {
					b.CustomType = nil
				}
			}
			b.Blocks, nested = thaw(prefix+name+".", b.Blocks)
			block = b
		}
		if !frozen {
			unfrozen = append(unfrozen, prefix+name)
		}
		out[name], unfrozen = block, append(unfrozen, nested...)
	}
	slices.Sort(unfrozen)
	return out, unfrozen
}

// protocolSchema returns the resource schema a protocol 6 server sends for s.
func protocolSchema(t *testing.T, s schema.Schema) []byte {
	resp, err := providerserver.NewProtocol6(schemaProvider{s: s})().GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil || len(resp.Diagnostics) > 0 || resp.ResourceSchemas["test_resource"] == nil {
		t.Fatal(err, resp.Diagnostics)
	}
	out, err := json.Marshal(resp.ResourceSchemas["test_resource"])
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// schemaProvider serves one resource with a fixed schema; GetProviderSchema
// needs nothing else.
type schemaProvider struct {
	provider.Provider
	s schema.Schema
}

func (schemaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "test"
}
func (schemaProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {}

func (schemaProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }
func (p schemaProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return schemaResource{s: p.s} }}
}

type schemaResource struct {
	resource.Resource
	s schema.Schema
}

func (schemaResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "test_resource"
}
func (r schemaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.s
}
