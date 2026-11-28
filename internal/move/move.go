// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package move

import (
	"context"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// moveMap defines supported source -> target resource type moves
// for identical-schema resource pairs where the deprecated name and
// the _v1 name share the same constructor function.
var moveMap = map[string]string{
	// core
	"kubernetes_namespace":               "kubernetes_namespace_v1",
	"kubernetes_service":                 "kubernetes_service_v1",
	"kubernetes_service_account":         "kubernetes_service_account_v1",
	"kubernetes_default_service_account": "kubernetes_default_service_account_v1",
	"kubernetes_config_map":              "kubernetes_config_map_v1",
	"kubernetes_secret":                  "kubernetes_secret_v1",
	"kubernetes_pod":                     "kubernetes_pod_v1",
	"kubernetes_endpoints":               "kubernetes_endpoints_v1",
	"kubernetes_limit_range":             "kubernetes_limit_range_v1",
	"kubernetes_persistent_volume":       "kubernetes_persistent_volume_v1",
	"kubernetes_persistent_volume_claim": "kubernetes_persistent_volume_claim_v1",
	"kubernetes_replication_controller":  "kubernetes_replication_controller_v1",
	"kubernetes_resource_quota":          "kubernetes_resource_quota_v1",
	// api registration
	"kubernetes_api_service": "kubernetes_api_service_v1",
	// apps
	"kubernetes_deployment":   "kubernetes_deployment_v1",
	"kubernetes_daemonset":    "kubernetes_daemon_set_v1",
	"kubernetes_stateful_set": "kubernetes_stateful_set_v1",
	// batch
	"kubernetes_job": "kubernetes_job_v1",
	// rbac
	"kubernetes_role":                 "kubernetes_role_v1",
	"kubernetes_role_binding":         "kubernetes_role_binding_v1",
	"kubernetes_cluster_role":         "kubernetes_cluster_role_v1",
	"kubernetes_cluster_role_binding": "kubernetes_cluster_role_binding_v1",
	// networking
	"kubernetes_ingress_class":  "kubernetes_ingress_class_v1",
	"kubernetes_network_policy": "kubernetes_network_policy_v1",
	// scheduling
	"kubernetes_priority_class": "kubernetes_priority_class_v1",
	// storage
	"kubernetes_storage_class": "kubernetes_storage_class_v1",
}

// reverseMoveMap supports moves in the opposite direction (v1 -> deprecated).
var reverseMoveMap map[string]string

func init() {
	reverseMoveMap = make(map[string]string, len(moveMap))
	for source, target := range moveMap {
		reverseMoveMap[target] = source
	}
}

// isSupported returns true if the given source -> target move is supported.
func isSupported(source, target string) bool {
	if t, ok := moveMap[source]; ok && t == target {
		return true
	}
	if t, ok := reverseMoveMap[source]; ok && t == target {
		return true
	}
	return false
}

// ServerWithMoveState wraps a tfprotov6.ProviderServer to add
// MoveResourceState support for identical-schema resource pairs.
type ServerWithMoveState struct {
	tfprotov6.ProviderServer
}

// NewServerWithMoveState creates a new wrapper around the given upstream server.
func NewServerWithMoveState(upstream tfprotov6.ProviderServer) *ServerWithMoveState {
	return &ServerWithMoveState{ProviderServer: upstream}
}

// MoveResourceState handles moving state between deprecated and versioned
// resource types that share identical schemas.
func (s *ServerWithMoveState) MoveResourceState(ctx context.Context, req *tfprotov6.MoveResourceStateRequest) (*tfprotov6.MoveResourceStateResponse, error) {
	if !isSupported(req.SourceTypeName, req.TargetTypeName) {
		return s.ProviderServer.MoveResourceState(ctx, req)
	}

	if req.SourceState == nil {
		return &tfprotov6.MoveResourceStateResponse{}, nil
	}

	return &tfprotov6.MoveResourceStateResponse{
		TargetState:   &tfprotov6.DynamicValue{JSON: req.SourceState.JSON},
		TargetPrivate: req.SourcePrivate,
	}, nil
}

// GetMetadata overrides the upstream response to advertise MoveResourceState support.
func (s *ServerWithMoveState) GetMetadata(ctx context.Context, req *tfprotov6.GetMetadataRequest) (*tfprotov6.GetMetadataResponse, error) {
	resp, err := s.ProviderServer.GetMetadata(ctx, req)
	if err != nil {
		return resp, err
	}
	if resp.ServerCapabilities == nil {
		resp.ServerCapabilities = &tfprotov6.ServerCapabilities{}
	}
	resp.ServerCapabilities.MoveResourceState = true
	return resp, err
}

// GetProviderSchema overrides the upstream response to advertise MoveResourceState support.
func (s *ServerWithMoveState) GetProviderSchema(ctx context.Context, req *tfprotov6.GetProviderSchemaRequest) (*tfprotov6.GetProviderSchemaResponse, error) {
	resp, err := s.ProviderServer.GetProviderSchema(ctx, req)
	if err != nil {
		return resp, err
	}
	if resp.ServerCapabilities == nil {
		resp.ServerCapabilities = &tfprotov6.ServerCapabilities{}
	}
	resp.ServerCapabilities.MoveResourceState = true
	return resp, err
}
