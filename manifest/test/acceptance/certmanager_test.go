// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
)

func TestKubernetesManifest_InstallCertManager(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)

	k8shelper.CreateNamespace(t, namespace)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))
		k8shelper.AssertResourceDoesNotExist(t, "v1", "namespaces", namespace)
	}()

	tfvars := TFVARS{
		"namespace": namespace,
	}
	tfconfig := loadTerraformConfig(t, "CertManager/certmanager.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	t.Log("CertManager has a very large manifest. This will take a few seconds to apply...")
	tf.Apply(ctx)
	t.Log("CertManager apply finished")

	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", "certificaterequests.cert-manager.io")
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", "certificates.cert-manager.io")
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", "challenges.acme.cert-manager.io")
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", "clusterissuers.cert-manager.io")
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", "issuers.cert-manager.io")
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", "orders.acme.cert-manager.io")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-cainjector")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-controller-issuers")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-controller-clusterissuers")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-controller-certificates")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-controller-orders")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-controller-challenges")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-controller-ingress-shim")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-view")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-edit")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-controller-approve:cert-manager-io")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-controller-certificatesigningrequests")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterroles", "cert-manager-webhook:subjectaccessreviews")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-cainjector")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-controller-issuers")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-controller-clusterissuers")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-controller-certificates")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-controller-orders")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-controller-challenges")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-controller-ingress-shim")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-controller-approve:cert-manager-io")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-controller-certificatesigningrequests")
	k8shelper.AssertResourceExists(t, "rbac.authorization.k8s.io/v1", "clusterrolebindings", "cert-manager-webhook:subjectaccessreviews")
	k8shelper.AssertResourceExists(t, "admissionregistration.k8s.io/v1", "mutatingwebhookconfigurations", "cert-manager-webhook")
	k8shelper.AssertResourceExists(t, "admissionregistration.k8s.io/v1", "validatingwebhookconfigurations", "cert-manager-webhook")
	k8shelper.AssertNamespacedResourceExists(t, "v1", "services", namespace, "cert-manager")
	k8shelper.AssertNamespacedResourceExists(t, "v1", "services", namespace, "cert-manager-webhook")
	k8shelper.AssertNamespacedResourceExists(t, "v1", "serviceaccounts", namespace, "cert-manager-cainjector")
	k8shelper.AssertNamespacedResourceExists(t, "v1", "serviceaccounts", namespace, "cert-manager")
	k8shelper.AssertNamespacedResourceExists(t, "v1", "serviceaccounts", namespace, "cert-manager-webhook")
	k8shelper.AssertNamespacedResourceExists(t, "apps/v1", "deployments", namespace, "cert-manager-cainjector")
	k8shelper.AssertNamespacedResourceExists(t, "apps/v1", "deployments", namespace, "cert-manager")
	k8shelper.AssertNamespacedResourceExists(t, "apps/v1", "deployments", namespace, "cert-manager-webhook")
	k8shelper.AssertNamespacedResourceExists(t, "rbac.authorization.k8s.io/v1", "roles", "kube-system", "cert-manager-cainjector:leaderelection")
	k8shelper.AssertNamespacedResourceExists(t, "rbac.authorization.k8s.io/v1", "roles", "kube-system", "cert-manager:leaderelection")
	k8shelper.AssertNamespacedResourceExists(t, "rbac.authorization.k8s.io/v1", "roles", namespace, "cert-manager-webhook:dynamic-serving")
	k8shelper.AssertNamespacedResourceExists(t, "rbac.authorization.k8s.io/v1", "rolebindings", "kube-system", "cert-manager-cainjector:leaderelection")
	k8shelper.AssertNamespacedResourceExists(t, "rbac.authorization.k8s.io/v1", "rolebindings", "kube-system", "cert-manager:leaderelection")
	k8shelper.AssertNamespacedResourceExists(t, "rbac.authorization.k8s.io/v1", "rolebindings", namespace, "cert-manager-webhook:dynamic-serving")
}
