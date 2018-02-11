package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func deploymentSpecFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"min_ready_seconds": {
			Type:         schema.TypeInt,
			Description:  "Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready)",
			Optional:     true,
			Default:      0,
			ValidateFunc: validatePositiveInteger,
		},
		"paused": {
			Type:        schema.TypeBool,
			Description: "Whether the deployment is paused",
			Optional:    true,
			Default:     false,
		},
		"progress_deadline_seconds": {
			Type:         schema.TypeInt,
			Description:  "The maximum time in seconds for a deployment to make progress before it is considered to be failed.",
			Optional:     true,
			Default:      600,
			ValidateFunc: validatePositiveInteger,
		},
		"replicas": {
			Type:         schema.TypeInt,
			Description:  "The number of desired replicas. Defaults to 1.",
			Optional:     true,
			Default:      1,
			ValidateFunc: validatePositiveInteger,
		},
		"revision_history_limit": {
			Type:         schema.TypeInt,
			Description:  "The number of old ReplicaSets to retain to allow rollback.",
			Optional:     true,
			Default:      10,
			ValidateFunc: validatePositiveInteger,
		},
		"selector": labelSelectorSchema("pods"),
		"strategy": {
			Type:        schema.TypeList,
			Description: "The deployment strategy to use to replace existing pods with new ones.",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"rolling_update": {
						Type:        schema.TypeList,
						Description: "Rolling update config params. Present only if strategy type = 'RollingUpdate'.",
						Optional:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"max_surge": {
									Description: "The maximum number of pods that can be scheduled above the desired number of pods. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). This can not be 0 if max_unavailable is 0. Absolute number is calculated from percentage by rounding up. Defaults to 25%",
									Default:     "25%",
								},
								"max_unavailable": {
									Description: "The maximum number of pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). Absolute number is calculated from percentage by rounding down. This can not be 0 if max_surge is 0. Defaults to 25%.",
									Default:     "25%",
								},
							},
						},
					},
					"type": {
						Type:         schema.TypeString,
						Description:  `Type of deployment. Can be "Recreate" or "RollingUpdate".`,
						Optional:     true,
						ValidateFunc: validateDeploymentStrategyType,
						Default:      "RollingUpdate",
					},
				},
			},
		},
		"template": {
			Type:        schema.TypeList,
			Description: "Template describes the pods that will be created.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podSpecFields(true),
			},
		},
	}

	return s
}
