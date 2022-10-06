package schemamd

import (
	"fmt"
	"io"

	"github.com/zclconf/go-cty/cty"
)

func WriteType(w io.Writer, ty cty.Type) error {
	switch {
	case ty == cty.DynamicPseudoType:
		_, err := io.WriteString(w, "Dynamic")
		return err
	case ty.IsPrimitiveType():
		switch ty {
		case cty.String:
			_, err := io.WriteString(w, "String")
			return err
		case cty.Bool:
			_, err := io.WriteString(w, "Boolean")
			return err
		case cty.Number:
			_, err := io.WriteString(w, "Number")
			return err
		}
		return fmt.Errorf("unexpected primitive type %q", ty.FriendlyName())
	case ty.IsCollectionType():
		switch {
		default:
			return fmt.Errorf("unexpected collection type %q", ty.FriendlyName())
		case ty.IsListType():
			_, err := io.WriteString(w, "List of ")
			if err != nil {
				return err
			}
		case ty.IsSetType():
			_, err := io.WriteString(w, "Set of ")
			if err != nil {
				return err
			}
		case ty.IsMapType():
			_, err := io.WriteString(w, "Map of ")
			if err != nil {
				return err
			}
		}
		err := WriteType(w, ty.ElementType())
		if err != nil {
			return fmt.Errorf("unable to write element type for %q: %w", ty.FriendlyName(), err)
		}
		return nil
	case ty.IsTupleType():
		// TODO: write additional type info?
		_, err := io.WriteString(w, "Tuple")
		return err
	case ty.IsObjectType():
		_, err := io.WriteString(w, "Object")
		return err
	}
	return fmt.Errorf("unexpected type %q", ty.FriendlyName())
}
