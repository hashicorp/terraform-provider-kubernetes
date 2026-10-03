// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// WaitForDeploymentReplicasForFramework exposes rollout wait logic for framework resources.
func WaitForDeploymentReplicasForFramework(ctx context.Context, conn *kubernetes.Clientset, ns, name string) retry.RetryFunc {
	return waitForDeploymentReplicasFunc(ctx, conn, ns, name)
}

// WaitForDaemonSetPodsForFramework exposes daemonset rollout wait logic for framework resources.
func WaitForDaemonSetPodsForFramework(ctx context.Context, conn *kubernetes.Clientset, ns, name string) retry.RetryFunc {
	return waitForDaemonSetPodsFunc(ctx, conn, ns, name)
}
