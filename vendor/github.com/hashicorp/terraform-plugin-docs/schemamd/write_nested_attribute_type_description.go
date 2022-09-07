package schemamd

import (
	"fmt"
	"io"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

func WriteNestedAttributeTypeDescription(w io.Writer, att *tfjson.SchemaAttribute, includeRW bool) error {
	nestedAttributeType := att.AttributeNestedType
	if nestedAttributeType == nil {
		return fmt.Errorf("AttributeNestedType is nil")
	}

	_, err := io.WriteString(w, "(Attributes")
	if err != nil {
		return err
	}

	nestingMode := nestedAttributeType.NestingMode
	switch nestingMode {
	default:
		return fmt.Errorf("unexpected nesting mode for attributes: %s", nestingMode)
	case tfjson.SchemaNestingModeSingle:
		// nothing
	case tfjson.SchemaNestingModeList:
		_, err = io.WriteString(w, " List")
		if err != nil {
			return err
		}
	case tfjson.SchemaNestingModeSet:
		_, err = io.WriteString(w, " Set")
		if err != nil {
			return err
		}
	case tfjson.SchemaNestingModeMap:
		_, err = io.WriteString(w, " Map")
		if err != nil {
			return err
		}
	}

	if nestingMode == tfjson.SchemaNestingModeSingle {
		if includeRW {
			switch {
			case childAttributeIsRequired(att):
				_, err = io.WriteString(w, ", Required")
				if err != nil {
					return err
				}
			case childAttributeIsOptional(att):
				_, err = io.WriteString(w, ", Optional")
				if err != nil {
					return err
				}
			case childAttributeIsReadOnly(att):
				_, err = io.WriteString(w, ", Read-only")
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("attribute does not match any filter states")
			}
		}
	} else {
		if nestedAttributeType.MinItems > 0 {
			_, err = io.WriteString(w, fmt.Sprintf(", Min: %d", nestedAttributeType.MinItems))
			if err != nil {
				return err
			}
		}
	}

	if nestedAttributeType.MaxItems > 0 {
		_, err = io.WriteString(w, fmt.Sprintf(", Max: %d", nestedAttributeType.MaxItems))
		if err != nil {
			return err
		}
	}

	if att.Sensitive {
		_, err := io.WriteString(w, ", Sensitive")
		if err != nil {
			return err
		}
	}

	if att.Deprecated {
		_, err = io.WriteString(w, ", Deprecated")
		if err != nil {
			return err
		}
	}

	_, err = io.WriteString(w, ")")
	if err != nil {
		return err
	}

	desc := strings.TrimSpace(att.Description)
	if desc != "" {
		_, err = io.WriteString(w, " "+desc)
		if err != nil {
			return err
		}
	}

	return nil
}
