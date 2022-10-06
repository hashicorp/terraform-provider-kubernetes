package schemamd

import (
	tfjson "github.com/hashicorp/terraform-json"
)

func childAttributeIsRequired(att *tfjson.SchemaAttribute) bool {
	return att.Required
}

func childBlockIsRequired(block *tfjson.SchemaBlockType) bool {
	return block.MinItems > 0
}

func childAttributeIsOptional(att *tfjson.SchemaAttribute) bool {
	return att.Optional
}

// childBlockIsOptional returns true for blocks with with min items 0
// which are either empty or have any required or optional children.
func childBlockIsOptional(block *tfjson.SchemaBlockType) bool {
	if block.MinItems > 0 {
		return false
	}

	if len(block.Block.NestedBlocks) == 0 && len(block.Block.Attributes) == 0 {
		return true
	}

	for _, childBlock := range block.Block.NestedBlocks {
		if childBlockIsRequired(childBlock) {
			return true
		}
		if childBlockIsOptional(childBlock) {
			return true
		}
	}

	for _, childAtt := range block.Block.Attributes {
		if childAttributeIsRequired(childAtt) {
			return true
		}
		if childAttributeIsOptional(childAtt) {
			return true
		}
	}

	return false
}

// Read-only is computed but not optional.
func childAttributeIsReadOnly(att *tfjson.SchemaAttribute) bool {
	// these shouldn't be able to be required, but just in case
	return att.Computed && !att.Optional && !att.Required
}

// childBlockIsReadOnly returns true for blocks where all leaves are read-only.
func childBlockIsReadOnly(block *tfjson.SchemaBlockType) bool {
	if block.MinItems != 0 || block.MaxItems != 0 {
		return false
	}

	for _, childBlock := range block.Block.NestedBlocks {
		if !childBlockIsReadOnly(childBlock) {
			return false
		}
	}

	for _, childAtt := range block.Block.Attributes {
		if !childAttributeIsReadOnly(childAtt) {
			return false
		}
	}

	return true
}
