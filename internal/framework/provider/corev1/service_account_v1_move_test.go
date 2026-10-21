// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"testing"
)

// MoveState tests: moving state from the deprecated kubernetes_service_account and
// kubernetes_default_service_account aliases, which stay on SDKv2, to the Framework resources.
//
// Distinct from the migration tests next door, which upgrade state in place under the same
// type name. There the Framework decodes SDKv2 state itself; here MoveState translates it
// field by field, so each stored shape is a separate branch to cover.
//
// Shapes and helpers come from service_account_v1_migration_test.go so both families
// describe the same stored state. Without MoveState these fail outright: Terraform moves state
// across resource types only when the destination provider implements it.

// The baseline shape: no optional fields, so both metadata maps are stored null.
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, true, "minimum", serviceAccountExpectEmpty)
}

// name is server-assigned and RequiresReplace. If the move dropped it or left it unknown, the
// plan would replace the ServiceAccount. It is also the only shape where SDKv2 stores a real
// generate_name instead of "".
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned_generateName(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, true, "generate-name", serviceAccountExpectEmpty)
}

// annotations populated, labels absent: one map stored null and one not.
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned_annotations(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, true, "annotations", serviceAccountExpectEmpty)
}

// The mirror: labels populated, annotations absent.
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned_labels(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, true, "labels", serviceAccountExpectEmpty)
}

// A key the filtering rules would strip, declared in config so it must survive the move and
// the Read that follows it.
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned_declaredInternalLabel(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, true, "declared-internal-label", serviceAccountExpectEmpty)
}

// `labels = {}` and `annotations = {}`. SDKv2 stores them as null, MoveState carries the null
// across, and the first plan updates to the {} the configuration asks for. This is the same
// one-off update the in-place upgrade produces; the post-apply plan proves it settles.
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned_emptyValues(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, true, "explicit-empty", serviceAccountExpectUpdate)
}

// Every optional field populated, named by generate_name. The named equivalent runs from
// every release in TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned_releases.
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned_completeGenerateName(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, true, "complete-generate-name", serviceAccountExpectEmpty)
}

// Every attribute populated, moved from the alias of every supported release. It also requires
// state, outputs, identity and the live object (including resourceVersion) to be unchanged.
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned_releases(t *testing.T) {
	testAccServiceAccountReleaseMatrix(t, "alias-move")
}

// The pre-identity tests are the only move coverage of state written before resource identity
// existed. There the source identity is absent, so MoveState builds the identity it writes.
// The fully populated pre-identity shape runs in the release matrix.
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversionedPreIdentity_basic(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2PreIdentityVersion, true, "minimum", serviceAccountExpectEmpty)
}

func TestAccKubernetesServiceAccountV1_MoveStateFromUnversionedPreIdentity_basicGenerateName(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2PreIdentityVersion, true, "generate-name", serviceAccountExpectEmpty)
}

// Same hazard as the in-place twin, reached through MoveState: the moved state is free of the
// ignored key, so the first update is where an unguarded diff would replace the live map.
func TestAccKubernetesServiceAccountV1_MoveStateFromUnversioned_ignoredMetadataSurvivesUpdate(t *testing.T) {
	testAccServiceAccountIgnoredMetadataSurvivesUpdate(t, true)
}
