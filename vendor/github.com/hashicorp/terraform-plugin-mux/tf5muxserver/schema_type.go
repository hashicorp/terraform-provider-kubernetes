package tf5muxserver

import (
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// schemaType returns the Type for a Schema.
//
// This function should be migrated to a (*tfprotov5.Schema).Type() method
// in terraform-plugin-go.
func schemaType(schema *tfprotov5.Schema) tftypes.Type {
	if schema == nil {
		return tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{},
		}
	}

	return schemaBlockType(schema.Block)
}

// schemaAttributeType returns the Type for a SchemaAttribute.
//
// This function should be migrated to a (*tfprotov5.SchemaAttribute).Type()
// method in terraform-plugin-go.
func schemaAttributeType(attribute *tfprotov5.SchemaAttribute) tftypes.Type {
	if attribute == nil {
		return nil
	}

	return attribute.Type
}

// schemaBlockType returns the Type for a SchemaBlock.
//
// This function should be migrated to a (*tfprotov5.SchemaBlock).Type()
// method in terraform-plugin-go.
func schemaBlockType(block *tfprotov5.SchemaBlock) tftypes.Type {
	if block == nil {
		return tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{},
		}
	}

	attributeTypes := map[string]tftypes.Type{}

	for _, attribute := range block.Attributes {
		if attribute == nil {
			continue
		}

		attributeType := schemaAttributeType(attribute)

		if attributeType == nil {
			continue
		}

		attributeTypes[attribute.Name] = attributeType
	}

	for _, block := range block.BlockTypes {
		if block == nil {
			continue
		}

		blockType := schemaNestedBlockType(block)

		if blockType == nil {
			continue
		}

		attributeTypes[block.TypeName] = blockType
	}

	return tftypes.Object{
		AttributeTypes: attributeTypes,
	}
}

// schemaNestedBlockType returns the Type for a SchemaNestedBlock.
//
// This function should be migrated to a (*tfprotov5.SchemaNestedBlock).Type()
// method in terraform-plugin-go.
func schemaNestedBlockType(nestedBlock *tfprotov5.SchemaNestedBlock) tftypes.Type {
	if nestedBlock == nil {
		return nil
	}

	switch nestedBlock.Nesting {
	case tfprotov5.SchemaNestedBlockNestingModeGroup:
		return schemaBlockType(nestedBlock.Block)
	case tfprotov5.SchemaNestedBlockNestingModeList:
		return tftypes.List{
			ElementType: schemaBlockType(nestedBlock.Block),
		}
	case tfprotov5.SchemaNestedBlockNestingModeMap:
		return tftypes.Map{
			ElementType: schemaBlockType(nestedBlock.Block),
		}
	case tfprotov5.SchemaNestedBlockNestingModeSet:
		return tftypes.Set{
			ElementType: schemaBlockType(nestedBlock.Block),
		}
	case tfprotov5.SchemaNestedBlockNestingModeSingle:
		return schemaBlockType(nestedBlock.Block)
	default:
		return nil
	}
}
