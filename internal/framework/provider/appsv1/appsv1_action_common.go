// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"strings"
	"time"

	tfaction "github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubectl/pkg/polymorphichelpers"
)

// defaultRestartTimeout is used when the practitioner does not set the
// "timeout" attribute on a restart action.
const defaultRestartTimeout = 5 * time.Minute

// restartPollInterval is how often the rollout status is polled while
// waiting for a restart to complete.
const restartPollInterval = 5 * time.Second

// RestartActionModel is the config model shared by the daemonset, deployment
// and statefulset restart actions.
type RestartActionModel struct {
	Namespace types.String `tfsdk:"namespace"`
	Name      types.String `tfsdk:"name"`
	Timeout   types.String `tfsdk:"timeout"`
}

// restartActionSchema returns the (identical) schema shared by the
// daemonset, deployment and statefulset restart actions.
func restartActionSchema(kind string) actionschema.Schema {
	return actionschema.Schema{
		Description: fmt.Sprintf(
			"Restarts a %s by patching its pod template with a `kubectl.kubernetes.io/restartedAt` annotation, equivalent to running `kubectl rollout restart`, and waits for the rollout to complete.",
			kind,
		),
		Attributes: map[string]actionschema.Attribute{
			"namespace": actionschema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The namespace of the %s to restart.", kind),
			},
			"name": actionschema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The name of the %s to restart.", kind),
			},
			"timeout": actionschema.StringAttribute{
				Optional: true,
				Description: "The length of time to wait for the rollout to complete before giving up, " +
					"expressed as a Go duration string, e.g. \"15m\" or \"1h\". Defaults to \"5m\".",
			},
		},
	}
}

// parseRestartTimeout parses the "timeout" attribute, falling back to
// defaultRestartTimeout when it is not set.
func parseRestartTimeout(v types.String) (time.Duration, diag.Diagnostics) {
	var diags diag.Diagnostics

	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return defaultRestartTimeout, diags
	}

	d, err := time.ParseDuration(v.ValueString())
	if err != nil {
		diags.AddAttributeError(
			path.Root("timeout"),
			"Invalid Timeout",
			fmt.Sprintf("%q is not a valid duration: %s", v.ValueString(), err),
		)
		return 0, diags
	}

	return d, diags
}

// restartPatch is the strategic merge patch kubectl applies to a workload's
// pod template to trigger a rolling restart.
func restartPatch() []byte {
	now := time.Now().UTC().Format(time.RFC3339)
	return []byte(fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":%q}}}}}`,
		now,
	))
}

// waitForRolloutComplete polls get until the workload's rollout status,
// as computed by kubectl's own status viewers, reports completion or the
// timeout elapses.
func waitForRolloutComplete(
	ctx context.Context,
	timeout time.Duration,
	sendProgress func(tfaction.InvokeProgressEvent),
	gvk k8sschema.GroupVersionKind,
	get func(ctx context.Context) (runtime.Object, error),
) diag.Diagnostics {
	var diags diag.Diagnostics

	statusViewer, err := polymorphichelpers.StatusViewerFor(gvk.GroupKind())
	if err != nil {
		diags.AddError("Error Determining Rollout Status", err.Error())
		return diags
	}

	pollErr := wait.PollUntilContextTimeout(ctx, restartPollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		obj, err := get(ctx)
		if err != nil {
			return false, err
		}

		u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return false, err
		}
		u["apiVersion"] = gvk.GroupVersion().String()
		u["kind"] = gvk.Kind

		msg, done, err := statusViewer.Status(&unstructured.Unstructured{Object: u}, 0)
		if err != nil {
			return false, err
		}

		if sendProgress != nil {
			if msg = strings.TrimSpace(msg); msg != "" {
				sendProgress(tfaction.InvokeProgressEvent{Message: msg})
			}
		}

		return done, nil
	})
	if pollErr != nil {
		diags.AddError("Error Waiting for Rollout to Complete", pollErr.Error())
	}

	return diags
}
