package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"reflect"
	"testing"
)

func TestFlattenLabelSelector(t *testing.T) {
	cases := []struct {
		Input          *metav1.LabelSelector
		ExpectedOutput []interface{}
	}{
		{
			&metav1.LabelSelector{MatchLabels: map[string]string{"key": "value"}},
			[]interface{}{
				map[string]interface{}{
					"match_labels": map[string]string{"key": "value"},
				},
			},
		},
		{
			&metav1.LabelSelector{},
			[]interface{}{map[string]interface{}{}},
		},
	}

	for _, tc := range cases {
		output := flattenLabelSelector(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
