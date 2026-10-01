// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDaemonSetMetadataPatchOps_NullToEmptyNoWrite(t *testing.T) {
	t.Parallel()

	state := DaemonSetV1Model{
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
				Annotations: daemonSetTFMap(nil),
				Labels:      daemonSetTFMap(nil),
			}},
		}},
	}
	plan := DaemonSetV1Model{
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
				Annotations: daemonSetTFMap(map[string]string{}),
				Labels:      daemonSetTFMap(nil),
			}},
		}},
	}

	ops := daemonSetMetadataPatchOps(state, plan, metav1.ObjectMeta{
		Annotations: map[string]string{"external": "keep"},
	})
	if len(ops) != 0 {
		t.Fatalf("expected no patch operations, got %v", ops)
	}
}

func TestDaemonSetMetadataPatchOps_FirstManagedKeyPreservesExternal(t *testing.T) {
	t.Parallel()

	state := DaemonSetV1Model{
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
				Annotations: daemonSetTFMap(nil),
				Labels:      daemonSetTFMap(nil),
			}},
		}},
	}
	plan := DaemonSetV1Model{
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
				Annotations: daemonSetTFMap(map[string]string{"managed": "new"}),
				Labels:      daemonSetTFMap(nil),
			}},
		}},
	}

	live := metav1.ObjectMeta{Annotations: map[string]string{"external": "keep"}}
	before, err := json.Marshal(map[string]metav1.ObjectMeta{"metadata": live})
	if err != nil {
		t.Fatal(err)
	}

	ops := daemonSetMetadataPatchOps(state, plan, live)
	if len(ops) == 0 {
		t.Fatal("expected metadata patch operations")
	}
	encoded, err := ops.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	patch, err := jsonpatch.DecodePatch(encoded)
	if err != nil {
		t.Fatal(err)
	}
	after, err := patch.Apply(before)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]metav1.ObjectMeta
	if err := json.Unmarshal(after, &got); err != nil {
		t.Fatal(err)
	}
	want := metav1.ObjectMeta{
		Annotations: map[string]string{"external": "keep", "managed": "new"},
	}
	if !reflect.DeepEqual(got["metadata"], want) {
		t.Fatalf("metadata=%#v want=%#v", got["metadata"], want)
	}
}

func daemonSetTFMap(kv map[string]string) types.Map {
	if kv == nil {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(kv))
	for k, v := range kv {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}
