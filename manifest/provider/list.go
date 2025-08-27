// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"math/big"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/payload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (s *RawProviderServer) ValidateListResourceConfig(ctx context.Context, req *tfprotov5.ValidateListResourceConfigRequest) (*tfprotov5.ValidateListResourceConfigResponse, error) {
	return &tfprotov5.ValidateListResourceConfigResponse{}, nil
}

func (s *RawProviderServer) ListResource(ctx context.Context, req *tfprotov5.ListResourceRequest) (*tfprotov5.ListResourceServerStream, error) {
	rt, err := GetListResourceType(req.TypeName)
	if err != nil {
		return nil, err
	}

	config, err := req.Config.Unmarshal(rt)
	if err != nil {
		return nil, err
	}

	var listConfig map[string]tftypes.Value
	err = config.As(&listConfig)
	if err != nil {
		return nil, err
	}

	rm, err := s.getRestMapper()
	if err != nil {
		return nil, err
	}

	client, err := s.getDynamicClient()
	if err != nil {
		return nil, err
	}

	var apiVersion, kind string
	listConfig["api_version"].As(&apiVersion)
	listConfig["kind"].As(&kind)

	gvr, err := getGVR(apiVersion, kind, rm)
	if err != nil {
		return nil, err
	}

	gvk := gvr.GroupVersion().WithKind(kind)
	ns, err := IsResourceNamespaced(gvk, rm)
	if err != nil {
		return nil, err
	}
	rcl := client.Resource(gvr)

	objectType, th, err := s.TFTypeFromOpenAPI(ctx, gvk, true)
	if err != nil {
		return nil, err
	}

	var labelSelector, fieldSelector string
	listConfig["label_selector"].As(&labelSelector)
	listConfig["field_selector"].As(&fieldSelector)
	var limit big.Float
	listConfig["limit"].As(&limit)
	lim, _ := limit.Int64()
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
		Limit:         lim,
	}

	var res *unstructured.UnstructuredList

	if ns {
		var namespace string
		listConfig["namespace"].As(&namespace)
		if namespace == "" {
			namespace = "default"
		}
		res, err = rcl.Namespace(namespace).List(ctx, listOptions)
	} else {
		res, err = rcl.List(ctx, listOptions)
	}
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	resourceType, err := GetResourceType(req.TypeName)
	if err != nil {
		return nil, err
	}

	results := func(push func(tfprotov5.ListResourceResult) bool) {
		for _, item := range res.Items {
			nobj, err := payload.ToTFValue(item.Object, objectType, th, tftypes.NewAttributePath())
			if err != nil {
				pushDiagFromError(push, err, "Error converting Kubernetes API resource to Terraform object")
				return
			}

			dv, err := createListResource(resourceType, nobj)
			if err != nil {
				pushDiagFromError(push, err, "Error converting Kubernetes API resource to Terraform object")
				return
			}

			id, err := createIdentityData(&item)
			if err != nil {
				pushDiagFromError(push, err, "Error creating resource identity for list result")
				return
			}

			if !push(tfprotov5.ListResourceResult{
				DisplayName: kind,
				Identity:    &tfprotov5.ResourceIdentityData{IdentityData: &id},
				Resource:    &dv,
			}) {
				return
			}
		}
	}

	return &tfprotov5.ListResourceServerStream{Results: results}, nil
}

func createListResource(resourceType tftypes.Type, obj tftypes.Value) (tfprotov5.DynamicValue, error) {
	result := tftypes.NewValue(resourceType, map[string]tftypes.Value{
		"object":   obj,
		"manifest": tftypes.NewValue(tftypes.String, ""),
		"wait_for": tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"fields": tftypes.Map{
					ElementType: tftypes.String,
				},
			},
		}, nil),
		"wait": tftypes.NewValue(tftypes.List{
			ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"rollout": tftypes.Bool,
					"fields":  tftypes.Map{ElementType: tftypes.String},
					"condition": tftypes.List{
						ElementType: tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"status": tftypes.String,
								"type":   tftypes.String,
							},
						},
					},
				},
			},
		}, nil),
		"timeouts": tftypes.NewValue(tftypes.List{
			ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"create": tftypes.String,
					"delete": tftypes.String,
					"update": tftypes.String,
				},
			},
		}, nil),
		"computed_fields": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
		"field_manager": tftypes.NewValue(tftypes.List{
			ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"force_conflicts": tftypes.Bool,
					"name":            tftypes.String,
				},
			},
		}, nil),
	})
	dv, err := tfprotov5.NewDynamicValue(result.Type(), result)
	if err != nil {
		return tfprotov5.DynamicValue{}, err
	}
	return dv, nil
}

func pushDiagFromError(push func(tfprotov5.ListResourceResult) bool, err error, summary string) {
	d := tfprotov5.Diagnostic{
		Detail:   err.Error(),
		Summary:  summary,
		Severity: tfprotov5.DiagnosticSeverityError,
	}
	push(tfprotov5.ListResourceResult{
		Diagnostics: []*tfprotov5.Diagnostic{&d},
	})
}
