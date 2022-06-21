package v1

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestFlattenObjectRef(t *testing.T) {
	cases := []struct {
		Input          *v1.ObjectReference
		ExpectedOutput []interface{}
	}{
		{
			&v1.ObjectReference{
				Name:      "demo",
				Namespace: "default",
			},
			[]interface{}{
				map[string]interface{}{
					"name":      "demo",
					"namespace": "default",
				},
			},
		},
		{
			&v1.ObjectReference{},
			[]interface{}{map[string]interface{}{}},
		},
	}

	for _, tc := range cases {
		output := flattenObjectRef(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}
