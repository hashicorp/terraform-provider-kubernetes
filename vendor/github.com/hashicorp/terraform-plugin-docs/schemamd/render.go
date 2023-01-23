package schemamd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

// Render writes a Markdown formatted Schema definition to the specified writer.
// A Schema contains a Version and the root Block, for example:
// "aws_accessanalyzer_analyzer": {
//   "block": {
//   },
// 	 "version": 0
// },
func Render(schema *tfjson.Schema, w io.Writer) error {
	_, err := io.WriteString(w, "## Schema\n\n")
	if err != nil {
		return err
	}

	err = writeRootBlock(w, schema.Block)
	if err != nil {
		return fmt.Errorf("unable to render schema: %w", err)
	}

	return nil
}

// Group by Attribute/Block characteristics.
type groupFilter struct {
	topLevelTitle string
	nestedTitle   string

	filterAttribute func(att *tfjson.SchemaAttribute) bool
	filterBlock     func(block *tfjson.SchemaBlockType) bool
}

var (
	// Attributes and Blocks are in one of 3 characteristic groups:
	// * Required
	// * Optional
	// * Read-Only
	groupFilters = []groupFilter{
		{"### Required", "Required:", childAttributeIsRequired, childBlockIsRequired},
		{"### Optional", "Optional:", childAttributeIsOptional, childBlockIsOptional},
		{"### Read-Only", "Read-Only:", childAttributeIsReadOnly, childBlockIsReadOnly},
	}
)

type nestedType struct {
	anchorID string
	path     []string
	block    *tfjson.SchemaBlock
	object   *cty.Type
	attrs    *tfjson.SchemaNestedAttributeType

	group groupFilter
}

func writeAttribute(w io.Writer, path []string, att *tfjson.SchemaAttribute, group groupFilter) ([]nestedType, error) {
	name := path[len(path)-1]

	_, err := io.WriteString(w, "- `"+name+"` ")
	if err != nil {
		return nil, err
	}

	if att.AttributeNestedType == nil {
		err = WriteAttributeDescription(w, att, false)
	} else {
		err = WriteNestedAttributeTypeDescription(w, att, false)
	}
	if err != nil {
		return nil, err
	}
	if att.AttributeType.IsTupleType() {
		return nil, fmt.Errorf("TODO: tuples are not yet supported")
	}

	anchorID := "nestedatt--" + strings.Join(path, "--")
	nestedTypes := []nestedType{}
	switch {
	case att.AttributeNestedType != nil:
		_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
		if err != nil {
			return nil, err
		}

		nestedTypes = append(nestedTypes, nestedType{
			anchorID: anchorID,
			path:     path,
			attrs:    att.AttributeNestedType,

			group: group,
		})
	case att.AttributeType.IsObjectType():
		_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
		if err != nil {
			return nil, err
		}

		nestedTypes = append(nestedTypes, nestedType{
			anchorID: anchorID,
			path:     path,
			object:   &att.AttributeType,

			group: group,
		})
	case att.AttributeType.IsCollectionType() && att.AttributeType.ElementType().IsObjectType():
		_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
		if err != nil {
			return nil, err
		}

		nt := att.AttributeType.ElementType()
		nestedTypes = append(nestedTypes, nestedType{
			anchorID: anchorID,
			path:     path,
			object:   &nt,

			group: group,
		})
	}

	_, err = io.WriteString(w, "\n")
	if err != nil {
		return nil, err
	}

	return nestedTypes, nil
}

func writeBlockType(w io.Writer, path []string, block *tfjson.SchemaBlockType) ([]nestedType, error) {
	name := path[len(path)-1]

	_, err := io.WriteString(w, "- `"+name+"` ")
	if err != nil {
		return nil, err
	}

	err = WriteBlockTypeDescription(w, block)
	if err != nil {
		return nil, fmt.Errorf("unable to write block description for %q: %w", name, err)
	}

	anchorID := "nestedblock--" + strings.Join(path, "--")
	nt := nestedType{
		anchorID: anchorID,
		path:     path,
		block:    block.Block,
	}

	_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
	if err != nil {
		return nil, err
	}

	_, err = io.WriteString(w, "\n")
	if err != nil {
		return nil, err
	}

	return []nestedType{nt}, nil
}

func writeRootBlock(w io.Writer, block *tfjson.SchemaBlock) error {
	return writeBlockChildren(w, nil, block, true)
}

// A Block contains:
// * Attributes (arbitrarily nested)
// * Nested Blocks (with nesting mode, max and min items)
// * Description(Kind)
// * Deprecated flag
// For example:
// "block": {
//   "attributes": {
//     "certificate_arn": {
// 	     "description_kind": "plain",
// 	     "required": true,
// 	     "type": "string"
//     }
// 	 },
// 	 "block_types": {
//     "timeouts": {
// 	     "block": {
// 		   "attributes": {
// 		   },
// 		   "description_kind": "plain"
// 	     },
// 	     "nesting_mode": "single"
//     }
// 	 },
// 	 "description_kind": "plain"
// },
func writeBlockChildren(w io.Writer, parents []string, block *tfjson.SchemaBlock, root bool) error {
	names := []string{}
	for n := range block.Attributes {
		names = append(names, n)
	}
	for n := range block.NestedBlocks {
		names = append(names, n)
	}

	groups := map[int][]string{}

	// Group Attributes/Blocks by characteristics.
nameLoop:
	for _, n := range names {
		if childBlock, ok := block.NestedBlocks[n]; ok {
			for i, gf := range groupFilters {
				if gf.filterBlock(childBlock) {
					groups[i] = append(groups[i], n)
					continue nameLoop
				}
			}
		} else if childAtt, ok := block.Attributes[n]; ok {
			for i, gf := range groupFilters {
				// By default, the attribute `id` is place in the "Read-Only" group
				// if the provider schema contained no `.Description` for it.
				//
				// If a `.Description` is provided instead, the behaviour will be the
				// same as for every other attribute.
				if strings.ToLower(n) == "id" && childAtt.Description == "" {
					if strings.Contains(gf.topLevelTitle, "Read-Only") {
						childAtt.Description = "The ID of this resource."
						groups[i] = append(groups[i], n)
						continue nameLoop
					}
				} else if gf.filterAttribute(childAtt) {
					groups[i] = append(groups[i], n)
					continue nameLoop
				}
			}
		}

		return fmt.Errorf("no match for %q, this can happen if you have incompatible schema defined, for example an "+
			"optional block where all the child attributes are computed, in which case the block itself should also "+
			"be marked computed", n)
	}

	nestedTypes := []nestedType{}

	// For each characteristic group
	//   If Attribute
	//     Write out summary including characteristic and type (if primitive type or collection of primitives)
	//     If NestedAttribute type, Object type or collection of Objects, add to list of nested types
	//   ElseIf Block
	//     Write out summary including characteristic
	//     Add block to list of nested types
	//   End
	// End
	// For each nested type:
	//   Write out heading
	//   If Block
	//     Recursively call this function (writeBlockChildren)
	//   ElseIf Object
	//     Call writeObjectChildren, which
	//       For each Object Attribute
	//         Write out summary including characteristic and type (if primitive type or collection of primitives)
	//         If Object type or collection of Objects, add to list of nested types
	//       End
	//       Recursively do nested type functionality
	//   ElseIf NestedAttribute
	//     Call writeNestedAttributeChildren, which
	//       For each nested Attribute
	//         Write out summary including characteristic and type (if primitive type or collection of primitives)
	//         If NestedAttribute type, Object type or collection of Objects, add to list of nested types
	//       End
	//       Recursively do nested type functionality
	//   End
	// End
	for i, gf := range groupFilters {
		sortedNames := groups[i]
		if len(sortedNames) == 0 {
			continue
		}
		sort.Strings(sortedNames)

		groupTitle := gf.topLevelTitle
		if !root {
			groupTitle = gf.nestedTitle
		}

		_, err := io.WriteString(w, groupTitle+"\n\n")
		if err != nil {
			return err
		}

		for _, name := range sortedNames {
			path := make([]string, len(parents), len(parents)+1)
			copy(path, parents)
			path = append(path, name)

			if childBlock, ok := block.NestedBlocks[name]; ok {
				nt, err := writeBlockType(w, path, childBlock)
				if err != nil {
					return fmt.Errorf("unable to render block %q: %w", name, err)
				}

				nestedTypes = append(nestedTypes, nt...)
				continue
			} else if childAtt, ok := block.Attributes[name]; ok {
				nt, err := writeAttribute(w, path, childAtt, gf)
				if err != nil {
					return fmt.Errorf("unable to render attribute %q: %w", name, err)
				}

				nestedTypes = append(nestedTypes, nt...)
				continue
			}

			return fmt.Errorf("unexpected name in schema render %q", name)
		}

		_, err = io.WriteString(w, "\n")
		if err != nil {
			return err
		}
	}

	err := writeNestedTypes(w, nestedTypes)
	if err != nil {
		return err
	}

	return nil
}

func writeNestedTypes(w io.Writer, nestedTypes []nestedType) error {
	for _, nt := range nestedTypes {
		_, err := io.WriteString(w, "<a id=\""+nt.anchorID+"\"></a>\n")
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, "### Nested Schema for `"+strings.Join(nt.path, ".")+"`\n\n")
		if err != nil {
			return err
		}

		switch {
		case nt.block != nil:
			err = writeBlockChildren(w, nt.path, nt.block, false)
			if err != nil {
				return err
			}
		case nt.object != nil:
			err = writeObjectChildren(w, nt.path, *nt.object, nt.group)
			if err != nil {
				return err
			}
		case nt.attrs != nil:
			err = writeNestedAttributeChildren(w, nt.path, nt.attrs, nt.group)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("missing information on nested block: %s", strings.Join(nt.path, "."))
		}

		_, err = io.WriteString(w, "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func writeObjectAttribute(w io.Writer, path []string, att cty.Type, group groupFilter) ([]nestedType, error) {
	name := path[len(path)-1]

	_, err := io.WriteString(w, "- `"+name+"` (")
	if err != nil {
		return nil, err
	}

	err = WriteType(w, att)
	if err != nil {
		return nil, err
	}

	_, err = io.WriteString(w, ")")
	if err != nil {
		return nil, err
	}

	if att.IsTupleType() {
		return nil, fmt.Errorf("TODO: tuples are not yet supported")
	}

	anchorID := "nestedobjatt--" + strings.Join(path, "--")
	nestedTypes := []nestedType{}
	switch {
	case att.IsObjectType():
		_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
		if err != nil {
			return nil, err
		}

		nestedTypes = append(nestedTypes, nestedType{
			anchorID: anchorID,
			path:     path,
			object:   &att,

			group: group,
		})
	case att.IsCollectionType() && att.ElementType().IsObjectType():
		_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
		if err != nil {
			return nil, err
		}

		nt := att.ElementType()
		nestedTypes = append(nestedTypes, nestedType{
			anchorID: anchorID,
			path:     path,
			object:   &nt,

			group: group,
		})
	}

	_, err = io.WriteString(w, "\n")
	if err != nil {
		return nil, err
	}

	return nestedTypes, nil
}

func writeObjectChildren(w io.Writer, parents []string, ty cty.Type, group groupFilter) error {
	_, err := io.WriteString(w, group.nestedTitle+"\n\n")
	if err != nil {
		return err
	}

	atts := ty.AttributeTypes()
	sortedNames := []string{}
	for n := range atts {
		sortedNames = append(sortedNames, n)
	}
	sort.Strings(sortedNames)
	nestedTypes := []nestedType{}

	for _, name := range sortedNames {
		att := atts[name]
		path := append(parents, name)

		nt, err := writeObjectAttribute(w, path, att, group)
		if err != nil {
			return fmt.Errorf("unable to render attribute %q: %w", name, err)
		}

		nestedTypes = append(nestedTypes, nt...)
	}

	_, err = io.WriteString(w, "\n")
	if err != nil {
		return err
	}

	err = writeNestedTypes(w, nestedTypes)
	if err != nil {
		return err
	}

	return nil
}

func writeNestedAttributeChildren(w io.Writer, parents []string, nestedAttributes *tfjson.SchemaNestedAttributeType, group groupFilter) error {
	sortedNames := []string{}
	for n := range nestedAttributes.Attributes {
		sortedNames = append(sortedNames, n)
	}
	sort.Strings(sortedNames)

	groups := map[int][]string{}
	for _, name := range sortedNames {
		att := nestedAttributes.Attributes[name]

		for i, gf := range groupFilters {
			if gf.filterAttribute(att) {
				groups[i] = append(groups[i], name)
			}
		}
	}

	nestedTypes := []nestedType{}

	for i, gf := range groupFilters {
		names, ok := groups[i]
		if !ok || len(names) == 0 {
			continue
		}

		_, err := io.WriteString(w, gf.nestedTitle+"\n\n")
		if err != nil {
			return err
		}

		for _, name := range names {
			att := nestedAttributes.Attributes[name]
			path := append(parents, name)

			nt, err := writeAttribute(w, path, att, group)
			if err != nil {
				return fmt.Errorf("unable to render attribute %q: %w", name, err)
			}

			nestedTypes = append(nestedTypes, nt...)
		}

		_, err = io.WriteString(w, "\n")
		if err != nil {
			return err
		}
	}

	err := writeNestedTypes(w, nestedTypes)
	if err != nil {
		return err
	}

	return nil
}
