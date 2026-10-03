// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Command frameworkdocs prepares a provider schema for tfplugindocs. It reads
// the output of "terraform providers schema -json" on stdin and writes it back
// with the details Plugin Framework resources cannot express in the protocol:
// block item counts, which their list validators enforce, and the field
// descriptions of reference lists such as image_pull_secrets.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	fwprovider "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
)

const providerAddress = "registry.terraform.io/hashicorp/kubernetes"

var (
	atLeast = regexp.MustCompile(`at least (\d+)`)
	atMost  = regexp.MustCompile(`at most (\d+)`)
	// Reference lists describe their single field inline, for example
	// "Nested object fields: `name`: Name of the referent. Each object must set `name`; `{}` is not valid."
	inlineField = regexp.MustCompile("^(.*?) ?Nested object fields: `(\\w+)`: (.*?)(?: Each object must set `\\w+`; `\\{\\}` is not valid\\.)?$")
)

type jsonObject = map[string]any

func main() {
	var doc jsonObject
	if err := json.NewDecoder(os.Stdin).Decode(&doc); err != nil {
		fail(err)
	}
	resources, ok := lookup(doc, "provider_schemas", providerAddress, "resource_schemas")
	if !ok {
		fail(fmt.Errorf("no resource schemas for %s", providerAddress))
	}

	ctx := context.Background()
	for _, newResource := range fwprovider.New("", nil).Resources(ctx) {
		r := newResource()
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "kubernetes"}, &meta)
		var s resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &s)
		block, ok := lookup(resources, meta.TypeName, "block")
		if !ok {
			fail(fmt.Errorf("resource %s missing from the provider schema", meta.TypeName))
		}
		annotate(ctx, block, s.Schema.Attributes, s.Schema.Blocks)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		fail(err)
	}
}

func annotate(ctx context.Context, block jsonObject, attributes map[string]schema.Attribute, blocks map[string]schema.Block) {
	for name, a := range attributes {
		if list, ok := a.(schema.ListAttribute); ok {
			if attr, ok := lookup(block, "attributes", name); ok {
				describeReferenceFields(attr, list)
			}
		}
	}
	for name, b := range blocks {
		nested, ok := lookup(block, "block_types", name)
		if !ok {
			continue
		}
		inner, _ := lookup(nested, "block")
		switch b := b.(type) {
		case schema.ListNestedBlock:
			var descriptions []string
			for _, v := range b.Validators {
				descriptions = append(descriptions, v.Description(ctx))
			}
			if minItems, maxItems := itemCounts(descriptions); minItems > 0 || maxItems > 0 {
				nested["min_items"], nested["max_items"] = minItems, maxItems
			}
			annotate(ctx, inner, b.NestedObject.Attributes, b.NestedObject.Blocks)
		case schema.SetNestedBlock:
			annotate(ctx, inner, b.NestedObject.Attributes, b.NestedObject.Blocks)
		case schema.SingleNestedBlock:
			annotate(ctx, inner, b.Attributes, b.Blocks)
		}
	}
}

func itemCounts(descriptions []string) (minItems, maxItems int) {
	for _, d := range descriptions {
		if m := atLeast.FindStringSubmatch(d); m != nil {
			minItems = max(minItems, atoi(m[1]))
		}
		if m := atMost.FindStringSubmatch(d); m != nil {
			maxItems = atoi(m[1])
		}
		if strings.Contains(d, "marked it as required") {
			minItems = max(minItems, 1)
		}
	}
	return minItems, maxItems
}

// describeReferenceFields renders a list of single-field objects as nested
// attributes, so the field is listed with its description.
func describeReferenceFields(attr jsonObject, list schema.ListAttribute) {
	element, ok := list.ElementType.(types.ObjectType)
	if !ok || len(element.AttrTypes) != 1 {
		return
	}
	description, _ := attr["description"].(string)
	m := inlineField.FindStringSubmatch(description)
	if m == nil || element.AttrTypes[m[2]] != types.StringType {
		return
	}
	delete(attr, "type")
	attr["description"] = m[1]
	attr["nested_type"] = jsonObject{
		"nesting_mode": "list",
		"attributes": jsonObject{
			m[2]: jsonObject{"type": "string", "description": m[3], "description_kind": "plain", "required": true},
		},
	}
}

func lookup(obj jsonObject, keys ...string) (jsonObject, bool) {
	for _, k := range keys {
		next, ok := obj[k].(jsonObject)
		if !ok {
			return nil, false
		}
		obj = next
	}
	return obj, true
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		fail(err)
	}
	return n
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "frameworkdocs:", err)
	os.Exit(1)
}
