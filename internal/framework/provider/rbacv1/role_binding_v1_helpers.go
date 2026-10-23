// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	rbacv1api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ── RoleRef helpers ───────────────────────────────────────────────────────────

// expandRoleRef converts a RoleRefModel to a Kubernetes RoleRef API object.
func expandRoleRef(m RoleRefModel) rbacv1api.RoleRef {
	return rbacv1api.RoleRef{
		APIGroup: m.APIGroup.ValueString(),
		Kind:     m.Kind.ValueString(),
		Name:     m.Name.ValueString(),
	}
}

// flattenRoleRef converts a Kubernetes RoleRef API object to a RoleRefModel.
func flattenRoleRef(in rbacv1api.RoleRef) RoleRefModel {
	return RoleRefModel{
		APIGroup: types.StringValue(in.APIGroup),
		Kind:     types.StringValue(in.Kind),
		Name:     types.StringValue(in.Name),
	}
}

// ── Subject helpers ───────────────────────────────────────────────────────────

// expandSubjects converts a slice of SubjectModel to Kubernetes Subject API objects.
func expandSubjects(in []SubjectModel) []rbacv1api.Subject {
	subjects := make([]rbacv1api.Subject, 0, len(in))
	for _, s := range in {
		subject := rbacv1api.Subject{
			Kind:      s.Kind.ValueString(),
			Name:      s.Name.ValueString(),
			APIGroup:  s.APIGroup.ValueString(),
			Namespace: s.Namespace.ValueString(),
		}
		subjects = append(subjects, subject)
	}
	return subjects
}

// flattenSubjectsWithPrior keeps a prior-state
// api_group of "" when the API returns the canonical rbac.authorization.k8s.io for a
// User or Group subject: the API server defaults an empty api_group for those kinds,
// so the two values are equivalent and rewriting the configured "" would cause a
// perpetual diff.
func flattenSubjectsWithPrior(in []rbacv1api.Subject, prior []SubjectModel) []SubjectModel {
	result := make([]SubjectModel, len(in))
	for i, s := range in {
		result[i] = flattenSubject(s)
		if i >= len(prior) {
			continue
		}
		p := prior[i]
		if p.APIGroup.IsNull() || p.APIGroup.IsUnknown() || p.APIGroup.ValueString() != "" {
			continue
		}
		if p.Kind.ValueString() != in[i].Kind || p.Name.ValueString() != in[i].Name {
			continue
		}
		if (in[i].Kind == rbacv1api.UserKind || in[i].Kind == rbacv1api.GroupKind) &&
			in[i].APIGroup == rbacv1api.GroupName {
			result[i].APIGroup = types.StringValue("")
		}
	}
	return result
}

func flattenSubject(s rbacv1api.Subject) SubjectModel {
	return SubjectModel{
		Kind:      types.StringValue(s.Kind),
		Name:      types.StringValue(s.Name),
		APIGroup:  types.StringValue(s.APIGroup),
		Namespace: types.StringValue(s.Namespace),
	}
}

// applySubjectComputedFields resolves unknown computed fields in plan subjects from the
// Kubernetes API response, while preserving all configured values.
//
// subject.api_group is Optional+Computed with no default. When a subject is configured
// without api_group (e.g. a ServiceAccount), the plan carries an unknown value for that
// field. After the API call succeeds, Kubernetes returns a concrete value. This helper
// fills in only those unknown slots; configured (known) values are left untouched.
//
// The number of subject blocks is never changed: Terraform requires the applied block
// count to equal the planned count, so any extra API subjects are ignored here and
// surface as drift on the next Read instead.
func applySubjectComputedFields(plan *[]SubjectModel, apiSubjects []rbacv1api.Subject) {
	for i := range *plan {
		if i >= len(apiSubjects) {
			break
		}
		s := &(*plan)[i]
		api := apiSubjects[i]

		// Resolve api_group only when the plan left it unknown (omitted by the caller).
		if s.APIGroup.IsUnknown() {
			s.APIGroup = types.StringValue(api.APIGroup)
		}
	}
}

// ── Subject change helper ─────────────────────────────────────────────────────

// subjectsChanged mirrors SDKv2 d.HasChange("subject"). An unknown planned value comes from an unconfigured computed
// field and keeps the state value, so it does not count as a change.
func subjectsChanged(state, plan []SubjectModel) bool {
	if len(state) != len(plan) {
		return true
	}
	for i := range plan {
		if !subjectFieldUnchanged(state[i].APIGroup, plan[i].APIGroup) ||
			!subjectFieldUnchanged(state[i].Kind, plan[i].Kind) ||
			!subjectFieldUnchanged(state[i].Name, plan[i].Name) ||
			!subjectFieldUnchanged(state[i].Namespace, plan[i].Namespace) {
			return true
		}
	}
	return false
}

func roleBindingIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(rbacAPIVersion),
			Kind:       types.StringValue(roleBindingKind),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
	}
}

// Seed only empty managed maps from the API, so adding the first managed key
// does not replace a map containing keys owned by another controller.
func roleBindingMetadataPatchOps(state, plan common.MetadataModel, current metav1.ObjectMeta) kubernetes.PatchOperations {
	ops := make(kubernetes.PatchOperations, 0)
	for _, field := range []struct {
		name        string
		state, plan types.Map
		current     map[string]string
	}{
		{"annotations", state.Annotations, plan.Annotations, current.Annotations},
		{"labels", state.Labels, plan.Labels, current.Labels},
	} {
		oldValues := common.ExpandMapForPatch(field.state)
		newValues := common.ExpandMapForPatch(field.plan)
		if len(oldValues) == 0 && len(newValues) == 0 {
			continue
		}
		if len(oldValues) == 0 {
			for key, value := range field.current {
				oldValues[key] = value
				if _, configured := newValues[key]; !configured {
					newValues[key] = value
				}
			}
		}
		ops = append(ops, kubernetes.DiffStringMap("/metadata/"+field.name, oldValues, newValues)...)
	}
	return ops
}

func subjectFieldUnchanged(state, plan types.String) bool {
	return plan.IsUnknown() || plan.Equal(state)
}
