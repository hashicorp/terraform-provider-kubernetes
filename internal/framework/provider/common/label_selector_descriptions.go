// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

// Descriptions of the fields of a Kubernetes label selector, as the SDKv2
// resources documented them.
const (
	LabelSelectorMatchLabelsDescription      = "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed."
	LabelSelectorMatchExpressionsDescription = "A list of label selector requirements. The requirements are ANDed."
	LabelSelectorKeyDescription              = "The label key that the selector applies to."
	LabelSelectorOperatorDescription         = "A key's relationship to a set of values. Valid operators are `In`, `NotIn`, `Exists` and `DoesNotExist`."
	LabelSelectorValuesDescription           = "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch."
)
