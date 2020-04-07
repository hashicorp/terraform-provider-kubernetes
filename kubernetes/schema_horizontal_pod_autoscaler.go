package kubernetes

import "github.com/hashicorp/terraform-plugin-sdk/helper/schema"

func metricTargetFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"average_utilization": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "averageUtilization is the target value of the average of the resource metric across all relevant pods, represented as a percentage of the requested value of the resource for the pods. Currently only valid for Resource metric source type",
			},
			"average_value": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "averageValue is the target value of the average of the metric across all relevant pods (as a quantity)",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "type represents whether the metric type is Utilization, Value, or AverageValue",
			},
			"value": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "value is the target value of the metric (as a quantity).",
			},
		},
	}
}

func resourceMetricSourceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "name is the name of the resource in question.",
			},
			"target": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        metricTargetFields(),
				Description: "Target specifies the target value for the given metric",
			},
		},
	}
}

func metricIdentifierFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "name is the name of the given metric",
			},
			"selector": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "selector is the string-encoded form of a standard kubernetes label selector for the given metric When set, it is passed as an additional parameter to the metrics server for more specific metrics scoping. When unset, just the metricName will be used to gather metrics.",
				Elem: &schema.Resource{
					Schema: labelSelectorFields(true),
				},
			},
		},
	}
}

func podsMetricSourceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"metric": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem:        metricIdentifierFields(),
				Description: "metric identifies the target metric by name and selector",
			},
			"target": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        metricTargetFields(),
				Description: "target specifies the target value for the given metric",
			},
		},
	}
}

func externalMetricSourceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"metric": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem:        metricIdentifierFields(),
				Description: "metric identifies the target metric by name and selector",
			},
			"target": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        metricTargetFields(),
				Description: "target specifies the target value for the given metric",
			},
		},
	}
}

func crossVersionObjectReferenceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"api_version": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "API version of the referent",
			},
			"kind": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names",
			},
		},
	}
}

func objectMetricSourceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"described_object": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem:     crossVersionObjectReferenceFields(),
				// NOTE Description is undocumented in K8s API docs
			},
			"metric": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem:        metricIdentifierFields(),
				Description: "metric identifies the target metric by name and selector",
			},
			"target": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        metricTargetFields(),
				Description: "target specifies the target value for the given metric",
			},
		},
	}
}

func metricSpecFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"external": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        externalMetricSourceFields(),
				Description: "",
			},
			"object": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        objectMetricSourceFields(),
				Description: "",
			},
			"pods": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        podsMetricSourceFields(),
				Description: "",
			},
			"resource": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        resourceMetricSourceFields(),
				Description: "",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: `type is the type of metric source. It should be one of "Object", "Pods", "External" or "Resource", each mapping to a matching field in the object.`,
			},
		},
	}
}
