package main

import (
	specresource "github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	specschema "github.com/hashicorp/terraform-plugin-codegen-spec/schema"
)

type ResourceGenerator struct {
	ResourceConfig ResourceConfig
	Schema         SchemaGenerator
}

func NewResourceGenerator(cfg ResourceConfig, spec specresource.Resource) ResourceGenerator {
	return ResourceGenerator{
		ResourceConfig: cfg,
		Schema: SchemaGenerator{
			Name:        cfg.Name,
			Description: cfg.Description,
			Attributes:  generateAttributes(spec.Schema.Attributes),
		},
	}
}

func (g *ResourceGenerator) GenerateSchemaFunctionCode() string {
	return renderTemplate(schemaFunctionTemplate, g)
}

func (g *ResourceGenerator) GenerateCRUDStubCode() string {
	return renderTemplate(crudStubsTemplate, g)
}

func (g *ResourceGenerator) GenerateResourceCode() string {
	return renderTemplate(resourceTemplate, g)
}

func (g *ResourceGenerator) GenerateModelCode() string {
	return renderTemplate(modelTemplate, g)
}

func generateAttributes(attrs specresource.Attributes) AttributesGenerator {
	generatedAttrs := AttributesGenerator{}
	for _, attr := range attrs {
		generatedAttr := AttributeGenerator{
			Name: attr.Name,
		}
		switch {
		case attr.Bool != nil:
			if attr.Bool.Description != nil {
				generatedAttr.Description = *attr.Bool.Description
			}
			generatedAttr.AttributeType = BoolAttributeType
			generatedAttr.Required = isRequired(attr.Bool.ComputedOptionalRequired)
			generatedAttr.Computed = isComputed(attr.Bool.ComputedOptionalRequired)
			generatedAttr.Sensitive = isSensitive(attr.Bool.Sensitive)
		case attr.String != nil:
			if attr.String.Description != nil {
				generatedAttr.Description = *attr.String.Description
			}
			generatedAttr.AttributeType = StringAttributeType
			generatedAttr.Required = isRequired(attr.String.ComputedOptionalRequired)
			generatedAttr.Computed = isComputed(attr.String.ComputedOptionalRequired)
			generatedAttr.Sensitive = isSensitive(attr.String.Sensitive)
		case attr.Number != nil:
			if attr.Number.Description != nil {
				generatedAttr.Description = *attr.Number.Description
			}
			generatedAttr.AttributeType = NumberAttributeType
			generatedAttr.Required = isRequired(attr.Number.ComputedOptionalRequired)
			generatedAttr.Computed = isComputed(attr.Number.ComputedOptionalRequired)
			generatedAttr.Sensitive = isSensitive(attr.Number.Sensitive)
		case attr.Int64 != nil:
			if attr.Int64.Description != nil {
				generatedAttr.Description = *attr.Int64.Description
			}
			generatedAttr.AttributeType = Int64AttributeType
			generatedAttr.Required = isRequired(attr.Int64.ComputedOptionalRequired)
			generatedAttr.Computed = isComputed(attr.Int64.ComputedOptionalRequired)
			generatedAttr.Sensitive = isSensitive(attr.Int64.Sensitive)
		case attr.Map != nil:
			if attr.Map.Description != nil {
				generatedAttr.Description = *attr.Map.Description
			}
			generatedAttr.AttributeType = MapAttributeType
			generatedAttr.Required = isRequired(attr.Map.ComputedOptionalRequired)
			generatedAttr.Computed = isComputed(attr.Map.ComputedOptionalRequired)
			generatedAttr.Sensitive = isSensitive(attr.Map.Sensitive)
			generatedAttr.ElementType = getElementType(attr.Map.ElementType)
		case attr.List != nil:
			if attr.List.Description != nil {
				generatedAttr.Description = *attr.List.Description
			}
			generatedAttr.AttributeType = ListAttributeType
			generatedAttr.Required = isRequired(attr.List.ComputedOptionalRequired)
			generatedAttr.Computed = isComputed(attr.List.ComputedOptionalRequired)
			generatedAttr.Sensitive = isSensitive(attr.List.Sensitive)
			generatedAttr.ElementType = getElementType(attr.List.ElementType)
		case attr.SingleNested != nil:
			if attr.SingleNested.Description != nil {
				generatedAttr.Description = *attr.SingleNested.Description
			}
			generatedAttr.AttributeType = SingleNestedAttributeType
			generatedAttr.Required = isRequired(attr.SingleNested.ComputedOptionalRequired)
			generatedAttr.Computed = isComputed(attr.SingleNested.ComputedOptionalRequired)
			generatedAttr.Sensitive = isSensitive(attr.SingleNested.Sensitive)
			generatedAttr.NestedAttributes = generateAttributes(attr.SingleNested.Attributes)
		case attr.ListNested != nil:
			if attr.ListNested.Description != nil {
				generatedAttr.Description = *attr.ListNested.Description
			}
			generatedAttr.AttributeType = ListNestedAttributeType
			generatedAttr.Required = isRequired(attr.ListNested.ComputedOptionalRequired)
			generatedAttr.Computed = isComputed(attr.ListNested.ComputedOptionalRequired)
			generatedAttr.Sensitive = isSensitive(attr.ListNested.Sensitive)
			generatedAttr.NestedAttributes = generateAttributes(attr.ListNested.NestedObject.Attributes)
		}
		generatedAttrs = append(generatedAttrs, generatedAttr)
	}
	return generatedAttrs
}

func isComputed(c specschema.ComputedOptionalRequired) bool {
	return c == specschema.Computed || c == specschema.ComputedOptional
}

func isRequired(c specschema.ComputedOptionalRequired) bool {
	return c == specschema.Required
}

func isSensitive(s *bool) bool {
	return s != nil && *s
}

func getElementType(e specschema.ElementType) string {
	switch {
	case e.Bool != nil:
		return BoolElementType
	case e.String != nil:
		return StringElementType
	case e.Number != nil:
		return NumberElementType
	case e.Int64 != nil:
		return Int64ElementType
	}
	panic("unsupported element type")
}
