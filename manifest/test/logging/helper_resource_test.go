// Copyright IBM Corp. 2017, 2026
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

func TestHelperResourceDebug(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	ctx := tfsdklogtest.RootLogger(context.Background(), &output)

	// InitTestContext messes with the standard library log package, which
	// we want to avoid in this unit testing. Instead, just create the
	// helper_resource subsystem and avoid the other InitTestContext logic.
	ctx = tfsdklog.NewSubsystem(ctx, logging.SubsystemHelperResource)

	logging.HelperResourceDebug(ctx, "test message")

	entries, err := tfsdklogtest.MultilineJSONDecode(&output)

	if err != nil {
		t.Fatalf("unable to read multiple line JSON: %s", err)
	}

	expectedEntries := []map[string]interface{}{
		{
			"@level":   "debug",
			"@message": "test message",
			"@module":  "sdk.helper_resource",
		},
	}

	if diff := cmp.Diff(entries, expectedEntries); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}

func TestHelperResourceError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	ctx := tfsdklogtest.RootLogger(context.Background(), &output)

	// InitTestContext messes with the standard library log package, which
	// we want to avoid in this unit testing. Instead, just create the
	// helper_resource subsystem and avoid the other InitTestContext logic.
	ctx = tfsdklog.NewSubsystem(ctx, logging.SubsystemHelperResource)

	logging.HelperResourceError(ctx, "test message")

	entries, err := tfsdklogtest.MultilineJSONDecode(&output)

	if err != nil {
		t.Fatalf("unable to read multiple line JSON: %s", err)
	}

	expectedEntries := []map[string]interface{}{
		{
			"@level":   "error",
			"@message": "test message",
			"@module":  "sdk.helper_resource",
		},
	}

	if diff := cmp.Diff(entries, expectedEntries); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}

func TestHelperResourceTrace(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	ctx := tfsdklogtest.RootLogger(context.Background(), &output)

	// InitTestContext messes with the standard library log package, which
	// we want to avoid in this unit testing. Instead, just create the
	// helper_resource subsystem and avoid the other InitTestContext logic.
	ctx = tfsdklog.NewSubsystem(ctx, logging.SubsystemHelperResource)

	logging.HelperResourceTrace(ctx, "test message")

	entries, err := tfsdklogtest.MultilineJSONDecode(&output)

	if err != nil {
		t.Fatalf("unable to read multiple line JSON: %s", err)
	}

	expectedEntries := []map[string]interface{}{
		{
			"@level":   "trace",
			"@message": "test message",
			"@module":  "sdk.helper_resource",
		},
	}

	if diff := cmp.Diff(entries, expectedEntries); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}

func TestHelperResourceWarn(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	ctx := tfsdklogtest.RootLogger(context.Background(), &output)

	// InitTestContext messes with the standard library log package, which
	// we want to avoid in this unit testing. Instead, just create the
	// helper_resource subsystem and avoid the other InitTestContext logic.
	ctx = tfsdklog.NewSubsystem(ctx, logging.SubsystemHelperResource)

	logging.HelperResourceWarn(ctx, "test message")

	entries, err := tfsdklogtest.MultilineJSONDecode(&output)

	if err != nil {
		t.Fatalf("unable to read multiple line JSON: %s", err)
	}

	expectedEntries := []map[string]interface{}{
		{
			"@level":   "warn",
			"@message": "test message",
			"@module":  "sdk.helper_resource",
		},
	}

	if diff := cmp.Diff(entries, expectedEntries); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}
}
