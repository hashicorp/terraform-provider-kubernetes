// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_CustomResource_OAPIv3(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	kind := strings.Title(randString(8))
	plural := strings.ToLower(kind) + "s"
	group := "terraform.io"
	version := "v1"
	groupVersion := group + "/" + version
	crd := fmt.Sprintf("%s.%s", plural, group)

	name := strings.ToLower(randName())
	namespace := "default" //randName()

	tfvars := TFVARS{
		"name":          name,
		"namespace":     namespace,
		"kind":          kind,
		"plural":        plural,
		"group":         group,
		"group_version": groupVersion,
		"cr_version":    version,
	}

	step1 := tfhelper.RequireNewWorkingDir(ctx, t)
	step1.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		step1.Destroy(ctx)
		step1.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)
	}()

	// Step 1: Create a structural CRD with a fairly complex schema
	// (inspired by the Prostgres Operator)
	tfconfig := loadTerraformConfig(t, "CustomResourceOAPI3/custom_resource_definition.tf", tfvars)
	step1.SetConfig(ctx, string(tfconfig))
	step1.Init(ctx)
	step1.Apply(ctx)
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)

	// wait for API to finish ingesting the CRD
	time.Sleep(5 * time.Second) //lintignore:R018

	reattachInfo2, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}
	step2 := tfhelper.RequireNewWorkingDir(ctx, t)
	step2.SetReattachInfo(ctx, reattachInfo2)
	defer func() {
		step2.Destroy(ctx)
		step2.Close()
		k8shelper.AssertResourceDoesNotExist(t, groupVersion, kind, name)
	}()

	// Step 2: create a CR of the type defined by the CRD above
	tfconfig = loadTerraformConfig(t, "CustomResourceOAPI3/custom_resource.tf", tfvars)
	step2.SetConfig(ctx, string(tfconfig))
	step2.Init(ctx)
	step2.Apply(ctx)

	s2, err := step2.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s2)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test_cr.object.metadata.name":           name,
		"kubernetes_manifest.test_cr.object.metadata.namespace":      namespace,
		"kubernetes_manifest.test_cr.object.spec.teamId":             "test",
		"kubernetes_manifest.test_cr.object.spec.volume.size":        "1Gi",
		"kubernetes_manifest.test_cr.object.spec.users.foo_user":     []interface{}{"superuser"},
		"kubernetes_manifest.test_cr.object.spec.users.bar_user":     []interface{}{},
		"kubernetes_manifest.test_cr.object.spec.users.mike":         []interface{}{"superuser", "createdb"},
		"kubernetes_manifest.test_cr.object.spec.numberOfInstances":  json.Number("2"),
		"kubernetes_manifest.test_cr.object.spec.databases.foo":      "devdb",
		"kubernetes_manifest.test_cr.object.spec.postgresql.version": "12",
	})
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.additionalVolumes")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.allowedSourceRanges")

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.clone")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.clone.cluster")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.clone.s3_access_key_id")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.clone.s3_endpoint")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.clone.s3_force_path_style")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.clone.s3_secret_access_key")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.clone.s3_wal_path")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.clone.timestamp")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.clone.uid")

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.connectionPooler")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.connectionPooler.dockerImage")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.connectionPooler.maxDBConnections")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.connectionPooler.mode")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.connectionPooler.numberOfInstances")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.connectionPooler.schema")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.connectionPooler.user")
	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.connectionPooler.resources")

	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.dockerImage")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.enableConnectionPooler")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.enableLogicalBackup")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.enableMasterLoadBalancer")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.enableReplicaConnectionPooler")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.enableReplicaLoadBalancer")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.enableShmVolume")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.initContainers")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.init_containers")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.logicalBackupSchedule")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.maintenanceWindows")

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.nodeAffinity")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution")
	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution")

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni.initdb")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni.loop_wait")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni.maximum_lag_on_failover")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni.pg_hba")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni.retry_timeout")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni.slots")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni.synchronous_mode")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni.synchronous_mode_strict")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.patroni.ttl")

	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.podAnnotations")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.podPriorityClassName")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.pod_priority_class_name")

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.postgresql")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.postgresql.parameters")

	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.preparedDatabases")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.replicaLoadBalancer")

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.resources")
	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.resources.limits")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.resources.limits.cpu")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.resources.limits.memory")
	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.resources.requests")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.resources.requests.cpu")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.resources.requests.memory")

	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.schedulerName")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.serviceAnnotations")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.sidecars")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.spiloFSGroup")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.spiloRunAsGroup")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.spiloRunAsUser")

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.standby")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.standby.s3_wal_path")

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test_cr.object.spec.tls")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.tls.caFile")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.tls.caSecretName")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.tls.certificateFile")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.tls.privateKeyFile")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.tls.secretName")

	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.tolerations")
	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test_cr.object.spec.useLoadBalancer")

}
