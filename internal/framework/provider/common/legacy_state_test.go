// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func TestDecodeLegacyState(t *testing.T) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}},
		Blocks: map[string]schema.Block{"metadata": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{"name": schema.StringAttribute{Optional: true}},
		}}},
	}
	renameID := func(state map[string]any) error {
		state["id"] = "ns/renamed"
		return nil
	}
	for _, tc := range []struct {
		name    string
		json    string
		rewrite func(map[string]any) error
		id      string
		wantErr bool
	}{
		{name: "removed attributes", json: `{"id":"ns/a","self_link":"x","metadata":[{"name":"a","self_link":"/apis/x"}]}`, id: "ns/a"},
		{name: "rewrite", json: `{"id":"ns/a","metadata":[{"name":"a","self_link":""}]}`, rewrite: renameID, id: "ns/renamed"},
		{name: "null", json: `null`, wantErr: true},
		{name: "null with rewrite", json: `null`, rewrite: renameID, wantErr: true},
		{name: "empty", json: ``, wantErr: true},
		{name: "wrong type", json: `{"id":["ns/a"]}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := common.DecodeLegacyState(context.Background(), &tfprotov6.RawState{JSON: []byte(tc.json)}, s, tc.rewrite)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %s", value)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var attributes map[string]tftypes.Value
			if err := value.As(&attributes); err != nil {
				t.Fatal(err)
			}
			var id string
			if err := attributes["id"].As(&id); err != nil || id != tc.id {
				t.Fatalf("id = %q (%v), want %q", id, err, tc.id)
			}
		})
	}
}

func TestLegacyStateName(t *testing.T) {
	metadata := func(namespace, name string) []any {
		return []any{map[string]any{"namespace": namespace, "name": name, "self_link": ""}}
	}
	for _, tc := range []struct {
		name    string
		values  map[string]any
		wantErr bool
	}{
		{name: "matching", values: map[string]any{"id": "ns/a", "metadata": metadata("ns", "a")}},
		{name: "mismatched name", values: map[string]any{"id": "ns/a", "metadata": metadata("ns", "b")}, wantErr: true},
		{name: "no namespace", values: map[string]any{"id": "a", "metadata": metadata("", "a")}, wantErr: true},
		{name: "extra separator", values: map[string]any{"id": "ns/a/b", "metadata": metadata("ns", "a/b")}, wantErr: true},
		{name: "no metadata", values: map[string]any{"id": "ns/a"}, wantErr: true},
		{name: "two metadata blocks", values: map[string]any{"id": "ns/a", "metadata": append(metadata("ns", "a"), metadata("ns", "a")...)}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			namespace, name, err := common.LegacyStateName(tc.values)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %q/%q", namespace, name)
				}
				return
			}
			if err != nil || namespace != "ns" || name != "a" {
				t.Fatalf("got %q/%q (%v), want ns/a", namespace, name, err)
			}
		})
	}
}
