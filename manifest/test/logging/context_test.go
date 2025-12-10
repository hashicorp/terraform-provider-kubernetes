// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package logging_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-log/tfsdklog"
	"github.com/hashicorp/terraform-plugin-log/tfsdklogtest"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/logging"
)

func TestInitContext(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	ctx := tfsdklogtest.RootLogger(context.Background(), &output)

	// Simulate root logger fields that would have been associated by
	// terraform-plugin-go prior to the InitContext() call.
	ctx = tfsdklog.SetField(ctx, "tf_rpc", "GetProviderSchema")
	ctx = tfsdklog.SetField(ctx, "tf_req_id", "123-testing-123")

	ctx = logging.InitContext(ctx)

	logging.HelperSchemaTrace(ctx, "test message")

	entries, err := tfsdklogtest.MultilineJSONDecode(&output)

	if err != nil {
		t.Fatalf("unable to read multiple line JSON: %s", err)
	}

	expectedEntries := []map[string]interface{}{
		{
			"@level":    "trace",
			"@message":  "test message",
			"@module":   "sdk.helper_schema",
			"tf_rpc":    "GetProviderSchema",
			"tf_req_id": "123-testing-123",
		},
	}

	if diff := cmp.Diff(entries, expectedEntries); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}

func TestTestNameContext(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	ctx := tfsdklogtest.RootLogger(context.Background(), &output)

	// InitTestContext messes with the standard library log package, which
	// we want to avoid in this unit testing. Instead, just create the
	// helper_resource subsystem and avoid the other InitTestContext logic.
	ctx = tfsdklog.NewSubsystem(ctx, logging.SubsystemHelperResource)

	ctx = logging.TestNameContext(ctx, "TestTestTest")

	logging.HelperResourceTrace(ctx, "test message")

	entries, err := tfsdklogtest.MultilineJSONDecode(&output)

	if err != nil {
		t.Fatalf("unable to read multiple line JSON: %s", err)
	}

	expectedEntries := []map[string]interface{}{
		{
			"@level":    "trace",
			"@message":  "test message",
			"@module":   "sdk.helper_resource",
			"test_name": "TestTestTest",
		},
	}

	if diff := cmp.Diff(entries, expectedEntries); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}

func TestTestStepNumberContext(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	ctx := tfsdklogtest.RootLogger(context.Background(), &output)

	// InitTestContext messes with the standard library log package, which
	// we want to avoid in this unit testing. Instead, just create the
	// helper_resource subsystem and avoid the other InitTestContext logic.
	ctx = tfsdklog.NewSubsystem(ctx, logging.SubsystemHelperResource)

	ctx = logging.TestStepNumberContext(ctx, 123)

	logging.HelperResourceTrace(ctx, "test message")

	entries, err := tfsdklogtest.MultilineJSONDecode(&output)

	if err != nil {
		t.Fatalf("unable to read multiple line JSON: %s", err)
	}

	expectedEntries := []map[string]interface{}{
		{
			"@level":           "trace",
			"@message":         "test message",
			"@module":          "sdk.helper_resource",
			"test_step_number": float64(123), // float64 due to default JSON unmarshalling
		},
	}

	if diff := cmp.Diff(entries, expectedEntries); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}

func TestTestTerraformPathContext(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	ctx := tfsdklogtest.RootLogger(context.Background(), &output)

	// InitTestContext messes with the standard library log package, which
	// we want to avoid in this unit testing. Instead, just create the
	// helper_resource subsystem and avoid the other InitTestContext logic.
	ctx = tfsdklog.NewSubsystem(ctx, logging.SubsystemHelperResource)

	ctx = logging.TestTerraformPathContext(ctx, "/usr/local/bin/terraform")

	logging.HelperResourceTrace(ctx, "test message")

	entries, err := tfsdklogtest.MultilineJSONDecode(&output)

	if err != nil {
		t.Fatalf("unable to read multiple line JSON: %s", err)
	}

	expectedEntries := []map[string]interface{}{
		{
			"@level":              "trace",
			"@message":            "test message",
			"@module":             "sdk.helper_resource",
			"test_terraform_path": "/usr/local/bin/terraform",
		},
	}

	if diff := cmp.Diff(entries, expectedEntries); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}

func TestTestWorkingDirectoryContext(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	ctx := tfsdklogtest.RootLogger(context.Background(), &output)

	// InitTestContext messes with the standard library log package, which
	// we want to avoid in this unit testing. Instead, just create the
	// helper_resource subsystem and avoid the other InitTestContext logic.
	ctx = tfsdklog.NewSubsystem(ctx, logging.SubsystemHelperResource)

	ctx = logging.TestWorkingDirectoryContext(ctx, "/tmp/test")

	logging.HelperResourceTrace(ctx, "test message")

	entries, err := tfsdklogtest.MultilineJSONDecode(&output)

	if err != nil {
		t.Fatalf("unable to read multiple line JSON: %s", err)
	}

	expectedEntries := []map[string]interface{}{
		{
			"@level":                 "trace",
			"@message":               "test message",
			"@module":                "sdk.helper_resource",
			"test_working_directory": "/tmp/test",
		},
	}

	if diff := cmp.Diff(entries, expectedEntries); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}
