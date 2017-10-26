package kubernetes

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
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

func validatePositiveInteger(value interface{}, key string) (ws []string, es []error) {
	v := value.(int)
	if v <= 0 {
		es = append(es, fmt.Errorf("%s must be greater than 0", key))
	}
	return
}

func validateDNSPolicy(value interface{}, key string) (ws []string, es []error) {
	v := value.(string)
	if v != "ClusterFirst" && v != "Default" {
		es = append(es, fmt.Errorf("%s must be either ClusterFirst or Default", key))
	}
	return
}

func validateRestartPolicy(value interface{}, key string) (ws []string, es []error) {
	v := value.(string)
	switch v {
	case "Always", "OnFailure", "Never":
		return
	default:
		es = append(es, fmt.Errorf("%s must be one of Always, OnFailure or Never ", key))
	}
	return
}

func validateTerminationGracePeriodSeconds(value interface{}, key string) (ws []string, es []error) {
	v := value.(int)
	if v < 0 {
		es = append(es, fmt.Errorf("%s must be greater than or equal to 0", key))
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

func validateAttributeValueDoesNotContain(searchString string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		input := v.(string)
		if strings.Contains(input, searchString) {
			errors = append(errors, fmt.Errorf(
				"%q must not contain %q",
				k, searchString))
		}
		return
	}
}

func validateAttributeValueIsIn(validValues []string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		input := v.(string)
		isValid := false
		for _, s := range validValues {
			if s == input {
				isValid = true
				break
			}
		}
		if !isValid {
			errors = append(errors, fmt.Errorf(
				"%q must contain a value from %#v, got %q",
				k, validValues, input))
		}
		return

	}
}
