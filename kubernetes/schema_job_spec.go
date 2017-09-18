package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func jobSpecFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"active_deadline_seconds": {
			Type:         schema.TypeInt,
			Optional:     true,
			ValidateFunc: validatePositiveInteger,
			Description:  "Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer.",
		},
		"completions": {
			Type:         schema.TypeInt,
			Optional:     true,
			ValidateFunc: validatePositiveInteger,
			Description:  "Specifies the desired number of successfully finished pods the job should be run with. Setting to nil means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value. Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
		},
		"manual_selector": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Controls generation of pod labels and pod selectors. Leave unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template. When true, the user is responsyble for picking unique labels and specifying the selector. Failure to pick a unique label may cause this and other jobs to not function correctly. More info: https://git.k8s.io/community/contributors/design-proposals/selector-generation.md",
		},
		"parallelism": {
			Type:         schema.TypeInt,
			Optional:     true,
			ValidateFunc: validatePositiveInteger,
			Description:  "Specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when ((.spec.completions - .status.successful) < .spec.parallelism), i.e. when the work left to do is less than max parallelism. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
		},
		"selector": {
			Type:        schema.TypeMap,
			Optional:    true,
			Description: "A label query over pods that should match the pod count. Normally, the system sets this field for you. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors",
		},
		"template": {
			Type:        schema.TypeList,
			Description: "Describes the pod that will be created when executing a job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podSpecFields(true),
			},
		},
	}

	return s
}
