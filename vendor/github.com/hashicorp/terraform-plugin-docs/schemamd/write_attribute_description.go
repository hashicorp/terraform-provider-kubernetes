package schemamd

import (
	"fmt"
	"io"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

func WriteAttributeDescription(w io.Writer, att *tfjson.SchemaAttribute, includeRW bool) error {
	_, err := io.WriteString(w, "(")
	if err != nil {
		return err
	}

	err = WriteType(w, att.AttributeType)
	if err != nil {
		return err
	}

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

	if att.Sensitive {
		_, err := io.WriteString(w, ", Sensitive")
		if err != nil {
			return err
		}
	}

	if att.Deprecated {
		_, err := io.WriteString(w, ", Deprecated")
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
