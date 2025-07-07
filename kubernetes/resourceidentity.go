// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceIdentitySchemaNamespaced() *schema.ResourceIdentity {
	return &schema.ResourceIdentity{
		Version: 1,
		SchemaFunc: func() map[string]*schema.Schema {
			return map[string]*schema.Schema{
				"namespace": {
					Type:              schema.TypeString,
					OptionalForImport: true,
				},
				"name": {
					Type:              schema.TypeString,
					RequiredForImport: true,
				},
				"api_version": {
					Type:              schema.TypeString,
					RequiredForImport: true,
				},
				"kind": {
					Type:              schema.TypeString,
					RequiredForImport: true,
				},
			}
		},
	}
}

func resourceIdentitySchemaNonNamespaced() *schema.ResourceIdentity {
	return &schema.ResourceIdentity{
		Version: 1,
		SchemaFunc: func() map[string]*schema.Schema {
			return map[string]*schema.Schema{
				"name": {
					Type:              schema.TypeString,
					RequiredForImport: true,
				},
				"api_version": {
					Type:              schema.TypeString,
					RequiredForImport: true,
				},
				"kind": {
					Type:              schema.TypeString,
					RequiredForImport: true,
				},
			}
		},
	}
}

func resourceIdentityImportNamespaced(ctx context.Context, rd *schema.ResourceData, i interface{}) ([]*schema.ResourceData, error) {
	if rd.Id() != "" {
		return []*schema.ResourceData{rd}, nil
	}

	rid, err := rd.Identity()
	if err != nil {
		return nil, err
	}

	namespace, ok := rid.Get("namespace").(string)
	if !ok {
		return nil, fmt.Errorf("could not get namespace from resource identity")
	}
	name, ok := rid.Get("name").(string)
	if !ok {
		return nil, fmt.Errorf("could not get name from resource identity")
	}

	rd.SetId(fmt.Sprintf("%s/%s", namespace, name))

	return []*schema.ResourceData{rd}, nil
}

func resourceIdentityImportNonNamespaced(ctx context.Context, rd *schema.ResourceData, i interface{}) ([]*schema.ResourceData, error) {
	if rd.Id() != "" {
		return []*schema.ResourceData{rd}, nil
	}

	rid, err := rd.Identity()
	if err != nil {
		return nil, err
	}

	name, ok := rid.Get("name").(string)
	if !ok {
		return nil, fmt.Errorf("could not get name from resource identity")
	}

	rd.SetId(name)

	return []*schema.ResourceData{rd}, nil
}

func setResourceIdentityNamespaced(d *schema.ResourceData, apiVersion, kind, namespace, name string) error {
	rid, err := d.Identity()
	if err != nil {
		return err
	}
	rid.Set("api_version", apiVersion)
	rid.Set("kind", kind)
	rid.Set("namespace", namespace)
	rid.Set("name", name)
	return nil
}

func setResourceIdentityNonNamespaced(d *schema.ResourceData, apiVersion, kind, name string) error {
	rid, err := d.Identity()
	if err != nil {
		return err
	}
	rid.Set("api_version", apiVersion)
	rid.Set("kind", kind)
	rid.Set("name", name)
	return nil
}
