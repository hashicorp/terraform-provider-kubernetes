package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (s *RawProviderServer) GetResourceIdentitySchemas(ctx context.Context, req *tfprotov5.GetResourceIdentitySchemasRequest) (*tfprotov5.GetResourceIdentitySchemasResponse, error) {
	s.logger.Trace("[GetResourceIdentitySchemas][Request]\n%s\n", dump(*req))
	resp := &tfprotov5.GetResourceIdentitySchemasResponse{
		IdentitySchemas: map[string]*tfprotov5.ResourceIdentitySchema{
			"kubernetes_manifest": {
				Version: 1,
				IdentityAttributes: []*tfprotov5.ResourceIdentitySchemaAttribute{
					{Name: "api_version", RequiredForImport: true, Type: tftypes.String},
					{Name: "kind", RequiredForImport: true, Type: tftypes.String},
					{Name: "name", RequiredForImport: true, Type: tftypes.String},
					{Name: "namespace", OptionalForImport: true, Type: tftypes.String},
				},
			},
		},
	}
	return resp, nil
}

func (s *RawProviderServer) UpgradeResourceIdentity(ctx context.Context, req *tfprotov5.UpgradeResourceIdentityRequest) (*tfprotov5.UpgradeResourceIdentityResponse, error) {
	s.logger.Trace("[UpgradeResourceIdentity][Request]\n%s\n", dump(*req))
	resp := &tfprotov5.UpgradeResourceIdentityResponse{}
	return resp, nil
}

func parseResourceIdentityData(rid *tfprotov5.ResourceIdentityData) (schema.GroupVersionKind, string, string, error) {
	namespace := "default"
	var apiVersion, kind, name string

	iddata, err := rid.IdentityData.Unmarshal(getIdentityType())
	if err != nil {
		return schema.GroupVersionKind{}, "", "",
			fmt.Errorf("could not unmarshal identity data: %v", err.Error())
	}

	var idvals map[string]tftypes.Value
	iddata.As(&idvals)

	idvals["api_version"].As(&apiVersion)
	idvals["kind"].As(&kind)
	idvals["namespace"].As(&namespace)
	idvals["name"].As(&name)

	gvk := schema.FromAPIVersionAndKind(apiVersion, kind)
	return gvk, name, namespace, nil
}

func getIdentityType() tftypes.Type {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"namespace":   tftypes.String,
			"name":        tftypes.String,
			"api_version": tftypes.String,
			"kind":        tftypes.String,
		},
	}
}

func createIdentityData(obj *unstructured.Unstructured) (tfprotov5.DynamicValue, error) {
	idVal := tftypes.NewValue(getIdentityType(), map[string]tftypes.Value{
		"namespace":   tftypes.NewValue(tftypes.String, obj.GetNamespace()),
		"name":        tftypes.NewValue(tftypes.String, obj.GetName()),
		"api_version": tftypes.NewValue(tftypes.String, obj.GetAPIVersion()),
		"kind":        tftypes.NewValue(tftypes.String, obj.GetKind()),
	})
	return tfprotov5.NewDynamicValue(idVal.Type(), idVal)
}
