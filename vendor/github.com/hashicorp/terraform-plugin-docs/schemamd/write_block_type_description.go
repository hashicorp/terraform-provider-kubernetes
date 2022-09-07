package schemamd

import (
	"fmt"
	"io"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

func WriteBlockTypeDescription(w io.Writer, block *tfjson.SchemaBlockType) error {
	_, err := io.WriteString(w, "(Block")
	if err != nil {
		return err
	}

	switch block.NestingMode {
	default:
		return fmt.Errorf("unexpected nesting mode for block: %s", block.NestingMode)
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

	if block.NestingMode == tfjson.SchemaNestingModeSingle {
		switch {
		case childBlockIsRequired(block):
			_, err = io.WriteString(w, ", Required")
			if err != nil {
				return err
			}
		case childBlockIsOptional(block):
			_, err = io.WriteString(w, ", Optional")
			if err != nil {
				return err
			}
		case childBlockIsReadOnly(block):
			_, err = io.WriteString(w, ", Read-only")
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("block does not match any filter states")
		}
	} else {
		if block.MinItems > 0 {
			_, err = io.WriteString(w, fmt.Sprintf(", Min: %d", block.MinItems))
			if err != nil {
				return err
			}
		}
	}

	if block.MaxItems > 0 {
		_, err = io.WriteString(w, fmt.Sprintf(", Max: %d", block.MaxItems))
		if err != nil {
			return err
		}
	}

	if block.Block.Deprecated {
		_, err = io.WriteString(w, ", Deprecated")
		if err != nil {
			return err
		}
	}

	_, err = io.WriteString(w, ")")
	if err != nil {
		return err
	}

	desc := strings.TrimSpace(block.Block.Description)
	if desc != "" {
		_, err = io.WriteString(w, " "+desc)
		if err != nil {
			return err
		}
	}

	return nil
}
