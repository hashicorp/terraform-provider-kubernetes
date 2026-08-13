// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"

	tfaction "github.com/hashicorp/terraform-plugin-framework/action"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	kappsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

var (
	_ tfaction.Action              = (*DeploymentRestartAction)(nil)
	_ tfaction.ActionWithConfigure = (*DeploymentRestartAction)(nil)
)

// DeploymentRestartAction implements the kubernetes_deployment_restart
// action, which triggers a rolling restart of a Deployment, equivalent to
// running `kubectl rollout restart deployment/<name>`.
type DeploymentRestartAction struct {
	SDKv2Meta func() any
}

func NewDeploymentRestartAction() tfaction.Action {
	return &DeploymentRestartAction{}
}

func (a *DeploymentRestartAction) Metadata(ctx context.Context, req tfaction.MetadataRequest, resp *tfaction.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_restart"
}

func (a *DeploymentRestartAction) Schema(ctx context.Context, req tfaction.SchemaRequest, resp *tfaction.SchemaResponse) {
	resp.Schema = restartActionSchema("Deployment")
}

func (a *DeploymentRestartAction) Configure(ctx context.Context, req tfaction.ConfigureRequest, resp *tfaction.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	a.SDKv2Meta = req.ProviderData.(func() any)
}

func (a *DeploymentRestartAction) Invoke(ctx context.Context, req tfaction.InvokeRequest, resp *tfaction.InvokeResponse) {
	var data RestartActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := parseRestartTimeout(data.Timeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := a.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Error Initializing Kubernetes Client", err.Error())
		return
	}

	ns := data.Namespace.ValueString()
	name := data.Name.ValueString()

	resp.SendProgress(tfaction.InvokeProgressEvent{
		Message: fmt.Sprintf("Restarting deployment %s/%s", ns, name),
	})

	if _, err := conn.AppsV1().Deployments(ns).Patch(ctx, name, k8stypes.StrategicMergePatchType, restartPatch(), metav1.PatchOptions{}); err != nil {
		resp.Diagnostics.AddError("Error Restarting Deployment", err.Error())
		return
	}

	resp.Diagnostics.Append(waitForRolloutComplete(
		ctx,
		timeout,
		resp.SendProgress,
		kappsv1.SchemeGroupVersion.WithKind("Deployment"),
		func(ctx context.Context) (runtime.Object, error) {
			return conn.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		},
	)...)
}
