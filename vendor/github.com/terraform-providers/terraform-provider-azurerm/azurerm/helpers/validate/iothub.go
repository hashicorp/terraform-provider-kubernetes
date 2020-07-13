package validate

import (
	"fmt"
	"regexp"
)

func IoTHubName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	// Portal: The value must contain only alphanumeric characters or the following: -
	if matched := regexp.MustCompile(`^[0-9a-zA-Z-]{1,}$`).Match([]byte(value)); !matched {
		errors = append(errors, fmt.Errorf("%q may only contain alphanumeric characters and dashes", k))
	}

	return warnings, errors
}

func IoTHubConsumerGroupName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	// Portal: The value must contain only alphanumeric characters or the following: - . _
	if matched := regexp.MustCompile(`^[0-9a-zA-Z-._]{1,}$`).Match([]byte(value)); !matched {
		errors = append(errors, fmt.Errorf("%q may only contain alphanumeric characters and dashes, periods and underscores", k))
	}

	return warnings, errors
}

func IoTHubEndpointName(v interface{}, _ string) (warnings []string, errors []error) {
	value := v.(string)

	reservedNames := []string{
		"events",
		"operationsMonitoringEvents",
		"fileNotifications",
		"$default",
	}

	for _, name := range reservedNames {
		if name == value {
			errors = append(errors, fmt.Errorf("The reserved endpoint name %s could not be used as a name for a custom endpoint", name))
		}
	}

	return warnings, errors
}
