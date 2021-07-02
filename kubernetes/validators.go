package kubernetes

import (
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	apiValidation "k8s.io/apimachinery/pkg/api/validation"
	utilValidation "k8s.io/apimachinery/pkg/util/validation"
)

func validateAnnotations(value interface{}, key string) (ws []string, es []error) {
	m := value.(map[string]interface{})
	for k := range m {
		errors := utilValidation.IsQualifiedName(strings.ToLower(k))
		if len(errors) > 0 {
			for _, e := range errors {
				es = append(es, fmt.Errorf("%s (%q) %s", key, k, e))
			}
		}
	}
	return
}

func validateBase64Encoded(v interface{}, key string) (ws []string, es []error) {
	s, ok := v.(string)
	if !ok {
		es = []error{fmt.Errorf("%s: must be a non-nil base64-encoded string", key)}
		return
	}

	_, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		es = []error{fmt.Errorf("%s: must be a base64-encoded string", key)}
		return
	}
	return
}

func validateBase64EncodedMap(value interface{}, key string) (ws []string, es []error) {
	m, ok := value.(map[string]interface{})
	if !ok {
		es = []error{fmt.Errorf("%s: must be a map of strings to base64 encoded strings", key)}
		return
	}

	for k, v := range m {
		_, errs := validateBase64Encoded(v, k)
		for _, e := range errs {
			es = append(es, fmt.Errorf("%s (%q) %s", k, v, e))
		}
	}

	return
}

func validateName(value interface{}, key string) (ws []string, es []error) {
	v := value.(string)
	errors := apiValidation.NameIsDNSSubdomain(v, false)
	if len(errors) > 0 {
		for _, err := range errors {
			es = append(es, fmt.Errorf("%s %s", key, err))
		}
	}
	return
}

func validateGenerateName(value interface{}, key string) (ws []string, es []error) {
	v := value.(string)

	errors := apiValidation.NameIsDNSLabel(v, true)
	if len(errors) > 0 {
		for _, err := range errors {
			es = append(es, fmt.Errorf("%s %s", key, err))
		}
	}
	return
}

func validateLabels(value interface{}, key string) (ws []string, es []error) {
	m := value.(map[string]interface{})
	for k, v := range m {
		for _, msg := range utilValidation.IsQualifiedName(k) {
			es = append(es, fmt.Errorf("%s (%q) %s", key, k, msg))
		}
		val, isString := v.(string)
		if !isString {
			es = append(es, fmt.Errorf("%s.%s (%#v): Expected value to be string", key, k, v))
			return
		}
		for _, msg := range utilValidation.IsValidLabelValue(val) {
			es = append(es, fmt.Errorf("%s (%q) %s", key, val, msg))
		}
	}
	return
}

func validatePortNum(value interface{}, key string) (ws []string, es []error) {
	errors := utilValidation.IsValidPortNum(value.(int))
	if len(errors) > 0 {
		for _, err := range errors {
			es = append(es, fmt.Errorf("%s %s", key, err))
		}
	}
	return
}

func validatePortName(value interface{}, key string) (ws []string, es []error) {
	errors := utilValidation.IsValidPortName(value.(string))
	if len(errors) > 0 {
		for _, err := range errors {
			es = append(es, fmt.Errorf("%s %s", key, err))
		}
	}
	return
}
func validatePortNumOrName(value interface{}, key string) (ws []string, es []error) {
	switch value.(type) {
	case string:
		intVal, err := strconv.Atoi(value.(string))
		if err != nil {
			return validatePortName(value, key)
		}
		return validatePortNum(intVal, key)
	case int:
		return validatePortNum(value, key)

	default:
		es = append(es, fmt.Errorf("%s must be defined of type string or int on the schema", key))
		return
	}
}

func validateResourceList(value interface{}, key string) (ws []string, es []error) {
	m := value.(map[string]interface{})
	for k, value := range m {
		if _, ok := value.(int); ok {
			continue
		}

		if v, ok := value.(string); ok {
			_, err := resource.ParseQuantity(v)
			if err != nil {
				es = append(es, fmt.Errorf("%s.%s (%q): %s", key, k, v, err))
			}
			continue
		}

		err := "Value can be either string or int"
		es = append(es, fmt.Errorf("%s.%s (%#v): %s", key, k, value, err))
	}
	return
}

func validateResourceQuantity(value interface{}, key string) (ws []string, es []error) {
	if v, ok := value.(string); ok {
		_, err := resource.ParseQuantity(v)
		if err != nil {
			es = append(es, fmt.Errorf("%s.%s : %s", key, v, err))
		}
	}
	return
}

func validateModeBits(value interface{}, key string) (ws []string, es []error) {
	if !strings.HasPrefix(value.(string), "0") {
		es = append(es, fmt.Errorf("%s: value %s should start with '0' (octal numeral)", key, value.(string)))
	}
	v, err := strconv.ParseInt(value.(string), 8, 32)
	if err != nil {
		es = append(es, fmt.Errorf("%s :Cannot parse octal numeral (%#v): %s", key, value, err))
	}
	if v < 0 || v > 0777 {
		es = append(es, fmt.Errorf("%s (%#o) expects octal notation (a value between 0 and 0777)", key, v))
	}
	return
}

func validateNumberOrPercentageOfPods() schema.SchemaValidateFunc {
	return validation.StringMatch(regexp.MustCompile(`^([0-9]+|[0-9]+%|)$`), "Must be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).")
}

func validateRelativePath() schema.SchemaValidateFunc {
	return validation.StringDoesNotMatch(regexp.MustCompile("^/|\\.\\."), "May not be an absolute path. May not contain the path element '..'")
}

// validateTypeStringNullableInt provides custom error messaging for TypeString ints
// Some arguments require an int value or unspecified, empty field.
// TODO: FIXME: make this an int and use a pointer to differentiate between 0 and unset.

//lintignore:V013
func validateTypeStringNullableInt(v interface{}, k string) (ws []string, es []error) {
	value, ok := v.(string)
	if !ok {
		es = append(es, fmt.Errorf("expected type of %s to be string", k))
		return
	}

	if value == "" {
		return
	}

	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		es = append(es, fmt.Errorf("%s: cannot parse '%s' as int: %s", k, value, err))
	}

	return
}
