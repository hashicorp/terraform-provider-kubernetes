package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func flattenStatefulSetSpec(in v1beta1.StatefulSetSpec, d *schema.ResourceData) ([]interface{}, error) {
	att := make(map[string]interface{})

	if in.Replicas != nil {
		att["replicas"] = *in.Replicas
	}
	att["service_name"] = in.ServiceName
	att["selector"] = in.Selector.MatchLabels

	podSpec, err := flattenPodSpec(in.Template.Spec)
	if err != nil {
		return nil, err
	}
	att["template"] = podSpec

	volClaimTemplates := make([]map[string]interface{}, len(in.VolumeClaimTemplates), len(in.VolumeClaimTemplates))
	for i, claim := range in.VolumeClaimTemplates {
		claimState := make(map[string]interface{})
		claimState["metadata"] = flattenSubMetadata(claim.ObjectMeta, d, fmt.Sprintf("spec.0.volume_claim_templates.%d", i))
		claimState["spec"] = flattenPersistentVolumeClaimSpec(claim.Spec)
		volClaimTemplates[i] = claimState
	}
	att["volume_claim_templates"] = volClaimTemplates

	return []interface{}{att}, nil
}

func expandStatefulSetSpec(statefulSet []interface{}) (v1beta1.StatefulSetSpec, error) {
	obj := v1beta1.StatefulSetSpec{}
	if len(statefulSet) == 0 || statefulSet[0] == nil {
		return obj, nil
	}
	in := statefulSet[0].(map[string]interface{})

	obj.Replicas = ptrToInt32(int32(in["replicas"].(int)))
	obj.Selector = &metav1.LabelSelector{
		MatchLabels: expandStringMap(in["selector"].(map[string]interface{})),
	}
	obj.ServiceName = in["service_name"].(string)

	podSpec, err := expandPodSpec(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}

	volClaimTemplates := in["volume_claim_templates"].([]interface{})
	pvcTemplates := make([]v1.PersistentVolumeClaim, len(volClaimTemplates), len(volClaimTemplates))
	for i, claimTemplateRaw := range volClaimTemplates {
		claimTemplateConfig := claimTemplateRaw.(map[string]interface{})
		metadata := expandMetadata(claimTemplateConfig["metadata"].([]interface{}))
		pvcSpec, _ := expandPersistentVolumeClaimSpec(claimTemplateConfig["spec"].([]interface{}))
		claim := v1.PersistentVolumeClaim{
			ObjectMeta: metadata,
			Spec:       pvcSpec,
		}
		pvcTemplates[i] = claim
	}
	obj.VolumeClaimTemplates = pvcTemplates

	obj.Template = v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: obj.Selector.MatchLabels,
		},
		Spec: podSpec,
	}

	return obj, nil
}
