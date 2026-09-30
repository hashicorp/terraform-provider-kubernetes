// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	batch "k8s.io/api/batch/v1"
	"k8s.io/utils/ptr"
)

func TestCronJobSpecIndexedOptionalPresence(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name  string
		value types.Int64
		want  *int32
	}{
		{name: "omitted", value: types.Int64Null()},
		{name: "explicit zero", value: types.Int64Value(0), want: ptr.To(int32(0))},
		{name: "explicit positive", value: types.Int64Value(2), want: ptr.To(int32(2))},
	} {
		t.Run(test.name, func(t *testing.T) {
			object := cronJobTestObject()
			object.Spec.JobTemplate.Spec.CompletionMode = ptr.To(batch.IndexedCompletion)
			_, state := cronJobTestState(t, object)
			jobSpec := path.Root("spec").AtListIndex(0).AtName("job_template").AtListIndex(0).AtName("spec").AtListIndex(0)
			for _, field := range []string{"backoff_limit_per_index", "max_failed_indexes"} {
				if d := state.SetAttribute(ctx, jobSpec.AtName(field), test.value); d.HasError() {
					t.Fatal(d)
				}
			}
			var spec types.List
			if d := state.GetAttribute(ctx, path.Root("spec"), &spec); d.HasError() {
				t.Fatal(d)
			}
			actual, d := expandCronJobSpec(ctx, spec)
			if d.HasError() {
				t.Fatal(d)
			}
			for field, got := range map[string]*int32{
				"backoff_limit_per_index": actual.JobTemplate.Spec.BackoffLimitPerIndex,
				"max_failed_indexes":      actual.JobTemplate.Spec.MaxFailedIndexes,
			} {
				if !reflect.DeepEqual(got, test.want) {
					t.Errorf("%s expanded to %v, want %v", field, got, test.want)
				}
			}
		})
	}
}
