package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func jobMetadataSchema() *schema.Schema {
	m := namespacedMetadataSchema("job", true)
	mr := m.Elem.(*schema.Resource)
	mr.Schema["labels"].Computed = true
	return m
}

func jobSpecFields() map[string]*schema.Schema {
	podTemplateFields := map[string]*schema.Schema{
		"metadata": metadataSchema("job", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec of the pods owned by the job",
			Optional:    true,
			ForceNew:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podSpecFields(false, false, false),
			},
		},
	}
	// Job and CronJob don't support "Always" as a value for "restart_policy" in Pod templates.
	// This changes the default value of "restart_policy" to "Never"
	// as expected by Job and CronJob resources.
	podTemplateFieldsSpecSchema := podTemplateFields["spec"].Elem.(*schema.Resource)
	restartPolicy := podTemplateFieldsSpecSchema.Schema["restart_policy"]
	restartPolicy.Default = "Never"

	s := map[string]*schema.Schema{
		"active_deadline_seconds": {
			Type:         schema.TypeInt,
			Optional:     true,
			ForceNew:     true,
			ValidateFunc: validatePositiveInteger,
			Description:  "Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer.",
		},
		"backoff_limit": {
			Type:         schema.TypeInt,
			Optional:     true,
			ForceNew:     true,
			ValidateFunc: validatePositiveInteger,
			Description:  "Specifies the number of retries before marking this job failed. Defaults to 6",
		},
		"completions": {
			Type:         schema.TypeInt,
			Optional:     true,
			ForceNew:     true,
			Default:      1,
			ValidateFunc: validatePositiveInteger,
			Description:  "Specifies the desired number of successfully finished pods the job should be run with. Setting to nil means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value. Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
		},
		"manual_selector": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Controls generation of pod labels and pod selectors. Leave unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template. When true, the user is responsyble for picking unique labels and specifying the selector. Failure to pick a unique label may cause this and other jobs to not function correctly. More info: https://git.k8s.io/community/contributors/design-proposals/selector-generation.md",
		},
		"parallelism": {
			Type:         schema.TypeInt,
			Optional:     true,
			ForceNew:     true,
			Default:      1,
			ValidateFunc: validatePositiveInteger,
			Description:  "Specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when ((.spec.completions - .status.successful) < .spec.parallelism), i.e. when the work left to do is less than max parallelism. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
		},
		"selector": {
			Type:        schema.TypeList,
			Description: "A label query over volumes to consider for binding.",
			Optional:    true,
			ForceNew:    true,
			Computed:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"match_expressions": {
						Type:        schema.TypeList,
						Description: "A list of label selector requirements. The requirements are ANDed.",
						Optional:    true,
						ForceNew:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"key": {
									Type:        schema.TypeString,
									Description: "The label key that the selector applies to.",
									Optional:    true,
									ForceNew:    true,
								},
								"operator": {
									Type:        schema.TypeString,
									Description: "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.",
									Optional:    true,
									ForceNew:    true,
								},
								"values": {
									Type:        schema.TypeSet,
									Description: "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
									Optional:    true,
									ForceNew:    true,
									Elem:        &schema.Schema{Type: schema.TypeString},
									Set:         schema.HashString,
								},
							},
						},
					},
					"match_labels": {
						Type:        schema.TypeMap,
						Description: "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
						Optional:    true,
						ForceNew:    true,
					},
				},
			},
		},
		"template": {
			Type:        schema.TypeList,
			Description: "Describes the pod that will be created when executing a job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
			Required:    true,
			MaxItems:    1,
			ForceNew:    true,
			Elem: &schema.Resource{
				Schema: podTemplateFields,
			},
		},
	}

	return s
}
