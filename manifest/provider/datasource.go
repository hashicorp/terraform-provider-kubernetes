// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/morph"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/payload"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (s *RawProviderServer) ReadDataSource(ctx context.Context, req *tfprotov5.ReadDataSourceRequest) (*tfprotov5.ReadDataSourceResponse, error) {
	switch req.TypeName {
	case "kubernetes_resource":
		return s.ReadSingularDataSource(ctx, req)
	case "kubernetes_resources":
		return s.ReadPluralDataSource(ctx, req)
	}

	resp := &tfprotov5.ReadDataSourceResponse{}
	resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
		Severity: tfprotov5.DiagnosticSeverityError,
		Summary:  fmt.Sprintf("Unknown Data Source: %s", req.TypeName),
	})
	return resp, nil
}

func (s *RawProviderServer) ReadPluralDataSource(ctx context.Context, req *tfprotov5.ReadDataSourceRequest) (*tfprotov5.ReadDataSourceResponse, error) {

	s.logger.Trace("[ReadDataSource][Request]\n%s\n", dump(*req))

	resp := &tfprotov5.ReadDataSourceResponse{}

	execDiag := s.canExecute()
	if len(execDiag) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, execDiag...)
		return resp, nil
	}

	rt, err := GetDataSourceType(req.TypeName)

	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine data source type",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	config, err := req.Config.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal data source configuration",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	var dsConfig map[string]tftypes.Value
	err = config.As(&dsConfig)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract attributes from data source configuration",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	rm, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to get RESTMapper client",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	client, err := s.getDynamicClient()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "failed to get Dynamic client",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	var apiVersion, kind string
	dsConfig["api_version"].As(&apiVersion)
	dsConfig["kind"].As(&kind)

	gvr, err := getGVR(apiVersion, kind, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine resource GroupVersion",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	gvk := gvr.GroupVersion().WithKind(kind)
	ns, err := IsResourceNamespaced(gvk, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed determine if resource is namespaced",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	rcl := client.Resource(gvr)

	objectType, th, err := s.TFTypeFromOpenAPI(ctx, gvk, true)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to save resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	var labelSelector, fieldSelector string
	dsConfig["label_selector"].As(&labelSelector)
	dsConfig["field_selector"].As(&fieldSelector)
	var limit big.Float
	dsConfig["limit"].As(&limit)
	lim, _ := limit.Int64()
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
		Limit:         lim,
	}

	var res *unstructured.UnstructuredList

	if ns {
		var namespace string
		dsConfig["namespace"].As(&namespace)
		if namespace == "" {
			namespace = "default"
		}
		res, err = rcl.Namespace(namespace).List(ctx, listOptions)
	} else {
		res, err = rcl.List(ctx, listOptions)
	}
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resp, nil
		}
		d := tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to get data source",
			Detail:   err.Error(),
		}
		resp.Diagnostics = append(resp.Diagnostics, &d)
		return resp, nil
	}

	listObjects := []tftypes.Value{}
	for _, item := range res.Items {
		nobj, err := payload.ToTFValue(item.Object, objectType, th, tftypes.NewAttributePath())
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to convert API response to Terraform value type",
				Detail:   err.Error(),
			})
			return resp, nil
		}
		nobj, err = morph.DeepUnknown(objectType, nobj, tftypes.NewAttributePath())
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to save resource state",
				Detail:   err.Error(),
			})
			return resp, nil
		}
		listObjects = append(listObjects, nobj)
	}

	elementTypes := make([]tftypes.Type, len(listObjects))

	for i, t := range listObjects {
		elementTypes[i] = t.Type()
	}

	tupleType := tftypes.Tuple{ElementTypes: elementTypes}
	tuple := tftypes.NewValue(tupleType, listObjects)

	rawState := make(map[string]tftypes.Value)
	err = config.As(&rawState)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to save resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	rawState["objects"] = morph.UnknownToNull(tuple)

	v := tftypes.NewValue(rt, rawState)
	state, err := tfprotov5.NewDynamicValue(v.Type(), v)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to save resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	resp.State = &state
	return resp, nil
}

// ReadDataSource function
func (s *RawProviderServer) ReadSingularDataSource(ctx context.Context, req *tfprotov5.ReadDataSourceRequest) (*tfprotov5.ReadDataSourceResponse, error) {
	s.logger.Trace("[ReadDataSource][Request]\n%s\n", dump(*req))

	resp := &tfprotov5.ReadDataSourceResponse{}

	execDiag := s.canExecute()
	if len(execDiag) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, execDiag...)
		return resp, nil
	}

	rt, err := GetDataSourceType(req.TypeName)

	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine data source type",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	config, err := req.Config.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal data source configuration",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	var dsConfig map[string]tftypes.Value
	err = config.As(&dsConfig)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract attributes from data source configuration",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	rm, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to get RESTMapper client",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	client, err := s.getDynamicClient()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "failed to get Dynamic client",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	var apiVersion, kind string
	dsConfig["api_version"].As(&apiVersion)
	dsConfig["kind"].As(&kind)

	gvr, err := getGVR(apiVersion, kind, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine resource GroupVersion",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	gvk := gvr.GroupVersion().WithKind(kind)
	ns, err := IsResourceNamespaced(gvk, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed determine if resource is namespaced",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	rcl := client.Resource(gvr)

	objectType, th, err := s.TFTypeFromOpenAPI(ctx, gvk, true)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to save resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	var metadataBlock []tftypes.Value
	dsConfig["metadata"].As(&metadataBlock)

	var metadata map[string]tftypes.Value
	metadataBlock[0].As(&metadata)

	var name string
	metadata["name"].As(&name)

	var res *unstructured.Unstructured
	if ns {
		var namespace string
		metadata["namespace"].As(&namespace)
		if namespace == "" {
			namespace = "default"
		}
		res, err = rcl.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		res, err = rcl.Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resp, nil
		}
		d := tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to get data source",
			Detail:   err.Error(),
		}
		resp.Diagnostics = append(resp.Diagnostics, &d)
		return resp, nil
	}

	nobj, err := payload.ToTFValue(res.Object, objectType, th, tftypes.NewAttributePath())
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to convert API response to Terraform value type",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	nobj, err = morph.DeepUnknown(objectType, nobj, tftypes.NewAttributePath())
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to save resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	rawState := make(map[string]tftypes.Value)
	err = config.As(&rawState)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to save resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	rawState["object"] = morph.UnknownToNull(nobj)

	v := tftypes.NewValue(rt, rawState)
	state, err := tfprotov5.NewDynamicValue(v.Type(), v)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to save resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	resp.State = &state
	return resp, nil
}

func getGVR(apiVersion, kind string, m meta.RESTMapper) (schema.GroupVersionResource, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	mapping, err := m.RESTMapping(gv.WithKind(kind).GroupKind(), gv.Version)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return mapping.Resource, err
}
