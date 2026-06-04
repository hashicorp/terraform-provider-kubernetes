// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package move

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// mockServer is a minimal mock that records whether MoveResourceState was called.
type mockServer struct {
	tfprotov6.ProviderServer
	moveResourceStateCalled bool
}

func (m *mockServer) MoveResourceState(_ context.Context, _ *tfprotov6.MoveResourceStateRequest) (*tfprotov6.MoveResourceStateResponse, error) {
	m.moveResourceStateCalled = true
	return &tfprotov6.MoveResourceStateResponse{
		Diagnostics: []*tfprotov6.Diagnostic{
			{
				Severity: tfprotov6.DiagnosticSeverityError,
				Summary:  "Move Resource State Not Supported",
			},
		},
	}, nil
}

func TestMoveResourceState_ForwardMove(t *testing.T) {
	mock := &mockServer{}
	server := NewServerWithMoveState(mock)

	sourceJSON := []byte(`{"id":"test-ns","metadata":[{"name":"test-ns"}]}`)
	sourcePrivate := []byte(`{"private":"data"}`)

	resp, err := server.MoveResourceState(context.Background(), &tfprotov6.MoveResourceStateRequest{
		SourceTypeName: "kubernetes_namespace",
		TargetTypeName: "kubernetes_namespace_v1",
		SourceState:    &tfprotov6.RawState{JSON: sourceJSON},
		SourcePrivate:  sourcePrivate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.moveResourceStateCalled {
		t.Fatal("expected move to be handled by wrapper, not delegated to upstream")
	}
	if resp.TargetState == nil {
		t.Fatal("expected TargetState to be set")
	}
	if string(resp.TargetState.JSON) != string(sourceJSON) {
		t.Errorf("expected TargetState.JSON = %s, got %s", sourceJSON, resp.TargetState.JSON)
	}
	if string(resp.TargetPrivate) != string(sourcePrivate) {
		t.Errorf("expected TargetPrivate = %s, got %s", sourcePrivate, resp.TargetPrivate)
	}
	if len(resp.Diagnostics) > 0 {
		t.Errorf("expected no diagnostics, got %v", resp.Diagnostics)
	}
}

func TestMoveResourceState_ReverseMove(t *testing.T) {
	mock := &mockServer{}
	server := NewServerWithMoveState(mock)

	sourceJSON := []byte(`{"id":"test-ns","metadata":[{"name":"test-ns"}]}`)

	resp, err := server.MoveResourceState(context.Background(), &tfprotov6.MoveResourceStateRequest{
		SourceTypeName: "kubernetes_namespace_v1",
		TargetTypeName: "kubernetes_namespace",
		SourceState:    &tfprotov6.RawState{JSON: sourceJSON},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.moveResourceStateCalled {
		t.Fatal("expected move to be handled by wrapper, not delegated to upstream")
	}
	if resp.TargetState == nil {
		t.Fatal("expected TargetState to be set")
	}
	if string(resp.TargetState.JSON) != string(sourceJSON) {
		t.Errorf("expected TargetState.JSON = %s, got %s", sourceJSON, resp.TargetState.JSON)
	}
}

func TestMoveResourceState_UnsupportedDelegatesToUpstream(t *testing.T) {
	mock := &mockServer{}
	server := NewServerWithMoveState(mock)

	resp, err := server.MoveResourceState(context.Background(), &tfprotov6.MoveResourceStateRequest{
		SourceTypeName: "kubernetes_cron_job",
		TargetTypeName: "kubernetes_cron_job_v1",
		SourceState:    &tfprotov6.RawState{JSON: []byte(`{}`)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.moveResourceStateCalled {
		t.Fatal("expected unsupported move to be delegated to upstream")
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected diagnostics from upstream mock")
	}
}

func TestMoveResourceState_NilSourceState(t *testing.T) {
	mock := &mockServer{}
	server := NewServerWithMoveState(mock)

	resp, err := server.MoveResourceState(context.Background(), &tfprotov6.MoveResourceStateRequest{
		SourceTypeName: "kubernetes_namespace",
		TargetTypeName: "kubernetes_namespace_v1",
		SourceState:    nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TargetState != nil {
		t.Error("expected nil TargetState for nil SourceState")
	}
}

func TestMoveResourceState_InvalidPairDelegates(t *testing.T) {
	mock := &mockServer{}
	server := NewServerWithMoveState(mock)

	// Valid source but wrong target
	_, err := server.MoveResourceState(context.Background(), &tfprotov6.MoveResourceStateRequest{
		SourceTypeName: "kubernetes_namespace",
		TargetTypeName: "kubernetes_secret_v1",
		SourceState:    &tfprotov6.RawState{JSON: []byte(`{}`)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.moveResourceStateCalled {
		t.Fatal("expected mismatched pair to be delegated to upstream")
	}
}

func TestMoveResourceState_DaemonsetNamingDifference(t *testing.T) {
	// kubernetes_daemonset -> kubernetes_daemon_set_v1 (note the underscore difference)
	mock := &mockServer{}
	server := NewServerWithMoveState(mock)

	sourceJSON := []byte(`{"id":"test"}`)

	resp, err := server.MoveResourceState(context.Background(), &tfprotov6.MoveResourceStateRequest{
		SourceTypeName: "kubernetes_daemonset",
		TargetTypeName: "kubernetes_daemon_set_v1",
		SourceState:    &tfprotov6.RawState{JSON: sourceJSON},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.moveResourceStateCalled {
		t.Fatal("expected daemonset move to be handled by wrapper")
	}
	if string(resp.TargetState.JSON) != string(sourceJSON) {
		t.Errorf("expected TargetState.JSON = %s, got %s", sourceJSON, resp.TargetState.JSON)
	}
}

func TestMoveMap_AllPairsAreBidirectional(t *testing.T) {
	for source, target := range moveMap {
		if rev, ok := reverseMoveMap[target]; !ok || rev != source {
			t.Errorf("moveMap entry %s -> %s has no valid reverse mapping", source, target)
		}
	}
}

func TestMoveMap_AllSupportedPairs(t *testing.T) {
	// Verify all entries in moveMap are recognized by isSupported in both directions
	for source, target := range moveMap {
		if !isSupported(source, target) {
			t.Errorf("isSupported(%s, %s) = false, expected true", source, target)
		}
		if !isSupported(target, source) {
			t.Errorf("isSupported(%s, %s) = false, expected true (reverse)", target, source)
		}
	}
}
