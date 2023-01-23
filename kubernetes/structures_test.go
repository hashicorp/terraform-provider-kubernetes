package kubernetes

import (
	"fmt"
	"testing"
)

func TestIsInternalKey(t *testing.T) {
	testCases := []struct {
		Key      string
		Expected bool
	}{
		{"", false},
		{"anyKey", false},
		{"any.hostname.io", false},
		{"any.hostname.com/with/path", false},
		{"service.beta.kubernetes.io/aws-load-balancer-backend-protocol", false},
		{"app.kubernetes.io", false},
		{"kubernetes.io", true},
		{"kubectl.kubernetes.io", true},
		{"pv.kubernetes.io/any/path", true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.Key), func(t *testing.T) {
			isInternal := isInternalKey(tc.Key)
			if tc.Expected && isInternal != tc.Expected {
				t.Fatalf("Expected %q to be internal", tc.Key)
			}
			if !tc.Expected && isInternal != tc.Expected {
				t.Fatalf("Expected %q not to be internal", tc.Key)
			}
		})
	}
}
