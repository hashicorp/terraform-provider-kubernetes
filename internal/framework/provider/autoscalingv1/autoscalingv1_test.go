// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv1_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
)

func sdkv2providerMeta() (func() any, error) {
	p := kubernetes.Provider()
	if diags := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); diags.HasError() {
		return nil, fmt.Errorf("configuring Kubernetes provider: %v", diags)
	}
	return p.Meta, nil
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		meta, err := sdkv2providerMeta()
		if err != nil {
			return nil, err
		}
		return providerserver.NewProtocol6WithError(provider.New("test", meta))()
	},
}

var testAccMuxProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

func externalProvider(version string) map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: version},
	}
}

func testAccPreCheck(t *testing.T) {
	ctx := context.TODO()
	hasFileCfg := (os.Getenv("KUBE_CTX_AUTH_INFO") != "" && os.Getenv("KUBE_CTX_CLUSTER") != "") ||
		os.Getenv("KUBE_CTX") != "" ||
		os.Getenv("KUBE_CONFIG_PATH") != ""
	hasUserCredentials := os.Getenv("KUBE_USER") != "" && os.Getenv("KUBE_PASSWORD") != ""
	hasClientCert := os.Getenv("KUBE_CLIENT_CERT_DATA") != "" && os.Getenv("KUBE_CLIENT_KEY_DATA") != ""
	hasStaticCfg := (os.Getenv("KUBE_HOST") != "" &&
		os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") != "") &&
		(hasUserCredentials || hasClientCert || os.Getenv("KUBE_TOKEN") != "")

	if !hasFileCfg && !hasStaticCfg && !hasUserCredentials {
		t.Fatalf("File config (KUBE_CTX_AUTH_INFO and KUBE_CTX_CLUSTER) or static configuration"+
			"(%s) or (%s) must be set for acceptance tests",
			strings.Join([]string{
				"KUBE_HOST",
				"KUBE_USER",
				"KUBE_PASSWORD",
				"KUBE_CLUSTER_CA_CERT_DATA",
			}, ", "),
			strings.Join([]string{
				"KUBE_HOST",
				"KUBE_CLIENT_CERT_DATA",
				"KUBE_CLIENT_KEY_DATA",
				"KUBE_CLUSTER_CA_CERT_DATA",
			}, ", "),
		)
	}

	diags := kubernetes.Provider().Configure(ctx, sdkv2.NewResourceConfigRaw(nil))
	if diags.HasError() {
		t.Fatal(diags[0].Summary)
	}
}

func testAccClient() (*client.Clientset, error) {
	meta, err := sdkv2providerMeta()
	if err != nil {
		return nil, err
	}
	return meta().(kubernetes.KubeClientsets).MainClientset()
}

func testAccHorizontalPodAutoscalerCheckDestroy(state *terraform.State) error {
	conn, err := testAccClient()
	if err != nil {
		return err
	}
	for _, module := range state.Modules {
		for _, rs := range module.Resources {
			if rs.Type != "kubernetes_horizontal_pod_autoscaler_v1" && rs.Type != "kubernetes_horizontal_pod_autoscaler" {
				continue
			}
			namespace, name, ok := strings.Cut(rs.Primary.ID, "/")
			if !ok {
				return fmt.Errorf("invalid HorizontalPodAutoscaler ID %q", rs.Primary.ID)
			}
			_, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(context.Background(), name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("checking destruction of HorizontalPodAutoscaler %q: %w", rs.Primary.ID, err)
			}
			return fmt.Errorf("HorizontalPodAutoscaler %q still exists", rs.Primary.ID)
		}
	}
	return nil
}

// testAccHorizontalPodAutoscalerCheckExists also fails when the object was replaced, by comparing
// its UID with the one recorded by an earlier step.
func testAccHorizontalPodAutoscalerCheckExists(address string, priorUID *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %q not found in state", address)
		}
		namespace, name, ok := strings.Cut(rs.Primary.ID, "/")
		if !ok {
			return fmt.Errorf("invalid HorizontalPodAutoscaler ID %q", rs.Primary.ID)
		}
		conn, err := testAccClient()
		if err != nil {
			return err
		}
		hpa, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		uid := string(hpa.UID)
		if uid == "" || rs.Primary.Attributes["metadata.0.uid"] != uid {
			return fmt.Errorf("HorizontalPodAutoscaler %q has mismatched API and state UID", rs.Primary.ID)
		}
		if priorUID != nil {
			if *priorUID != "" && *priorUID != uid {
				return fmt.Errorf("HorizontalPodAutoscaler %q was replaced: UID changed from %q to %q", rs.Primary.ID, *priorUID, uid)
			}
			*priorUID = uid
		}
		return nil
	}
}
