package validate

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

// LowerCasedString validates that the string is lower-cased
func LowerCasedString(i interface{}, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}

	if strings.TrimSpace(v) == "" {
		return nil, []error{fmt.Errorf("%q must not be empty", k)}
	}

	if strings.ToLower(v) != v {
		return nil, []error{fmt.Errorf("%q must be a lower-cased string", k)}
	}

	if strings.ContainsAny(v, " ") {
		return nil, []error{fmt.Errorf("%q cannot contain whitespace", k)}
	}

	return nil, nil
}

// deprecated use validation.StringIsNotEmpty instead
func NoEmptyStrings(i interface{}, k string) ([]string, []error) {
	return validation.StringIsNotEmpty(i, k)
}
