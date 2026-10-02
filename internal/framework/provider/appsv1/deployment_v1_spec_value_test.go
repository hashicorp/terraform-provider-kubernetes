// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podtemplate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// deploymentSpecListValue must produce exactly what reflection-based
// types.ListValueFrom produced, including null versus empty nested lists.
func TestDeploymentSpecListValueMatchesReflection(t *testing.T) {
	ctx := context.Background()
	podSpec := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx", Ports: []corev1.ContainerPort{{ContainerPort: 80}}}}}
	templateSpec, d := podtemplate.FlattenSpec(ctx, podSpec, templateSpecNull(), path.Root("spec"))
	if d.HasError() {
		t.Fatal(d)
	}
	selector, d := flattenWorkloadSelector(ctx, &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}}, nil)
	if d.HasError() {
		t.Fatal(d)
	}
	strategy, d := flattenDeploymentStrategy(ctx, appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType})
	if d.HasError() {
		t.Fatal(d)
	}
	metadata, d := flattenTemplateMetadata(ctx, metav1.ObjectMeta{Labels: map[string]string{"app": "web"}}, nil)
	if d.HasError() {
		t.Fatal(d)
	}
	full := deploymentSpecModel{
		MinReadySeconds:         types.Int64Value(3),
		Paused:                  types.BoolValue(false),
		ProgressDeadlineSeconds: types.Int64Value(600),
		Replicas:                types.StringValue("2"),
		RevisionHistoryLimit:    types.Int64Null(),
		Selector:                selector,
		Strategy:                strategy,
		Template:                []deploymentTemplateModel{{Metadata: metadata, Spec: templateSpec}},
	}
	noSelector := full
	noSelector.Selector = nil
	emptySelector := full
	emptySelector.Selector = []deploymentSelectorModel{}
	noTemplateMetadata := full
	noTemplateMetadata.Template = []deploymentTemplateModel{{Spec: templateSpec}}
	noTemplate := full
	noTemplate.Template = nil
	emptyTemplate := full
	emptyTemplate.Template = []deploymentTemplateModel{}
	expressions, d := flattenWorkloadSelector(ctx, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
		{Key: "tier", Operator: metav1.LabelSelectorOpIn, Values: []string{"fe", "be"}},
		{Key: "canary", Operator: metav1.LabelSelectorOpDoesNotExist},
	}}, nil)
	if d.HasError() {
		t.Fatal(d)
	}
	withExpressions := full
	withExpressions.Selector = expressions

	for name, model := range map[string]deploymentSpecModel{
		"full": full, "nil selector": noSelector, "empty selector": emptySelector,
		"nil template metadata": noTemplateMetadata, "nil template": noTemplate,
		"empty template": emptyTemplate, "match expressions": withExpressions,
	} {
		t.Run(name, func(t *testing.T) {
			want, wantDiags := types.ListValueFrom(ctx, deploymentSpecListType().ElemType, []deploymentSpecModel{model})
			got, gotDiags := deploymentSpecListValue(ctx, model)
			if wantDiags.HasError() != gotDiags.HasError() {
				t.Fatalf("diagnostics differ: reflection=%v direct=%v", wantDiags, gotDiags)
			}
			if !want.Equal(got) {
				t.Fatalf("values differ:\nreflection: %s\ndirect:     %s", want, got)
			}
		})
	}
}
