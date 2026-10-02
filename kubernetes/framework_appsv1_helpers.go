// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// ExpandDeploymentSpecForFramework reuses the canonical SDKv2 deployment spec expander.
func ExpandDeploymentSpecForFramework(in []interface{}) (*appsv1.DeploymentSpec, error) {
	return expandDeploymentSpec(in)
}

// ExpandDaemonSetSpecForFramework reuses the canonical SDKv2 daemonset spec expander.
func ExpandDaemonSetSpecForFramework(in []interface{}) (appsv1.DaemonSetSpec, error) {
	return expandDaemonSetSpec(in)
}

// FlattenDeploymentSpecForFramework reuses the canonical SDKv2 deployment spec flattener.
func FlattenDeploymentSpecForFramework(in appsv1.DeploymentSpec) ([]interface{}, error) {
	return flattenDeploymentSpec(in, nil, nil)
}

// FlattenDaemonSetSpecForFramework reuses the canonical SDKv2 daemonset spec flattener.
func FlattenDaemonSetSpecForFramework(in appsv1.DaemonSetSpec) ([]interface{}, error) {
	return flattenDaemonSetSpec(in, nil, nil)
}

// WaitForDeploymentReplicasForFramework exposes rollout wait logic for framework resources.
func WaitForDeploymentReplicasForFramework(ctx context.Context, conn *kubernetes.Clientset, ns, name string) retry.RetryFunc {
	return waitForDeploymentReplicasFunc(ctx, conn, ns, name)
}

// WaitForDaemonSetPodsForFramework exposes daemonset rollout wait logic for framework resources.
func WaitForDaemonSetPodsForFramework(ctx context.Context, conn *kubernetes.Clientset, ns, name string) retry.RetryFunc {
	return waitForDaemonSetPodsFunc(ctx, conn, ns, name)
}

// UpgradeTemplatePodSpecWithResourcesFieldV0ForFramework applies v0->v1 resources migration.
func UpgradeTemplatePodSpecWithResourcesFieldV0ForFramework(ctx context.Context, rawState map[string]interface{}) map[string]interface{} {
	return upgradeTemplatePodSpecWithResourcesFieldV0(ctx, rawState, nil)
}
