// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesNetworkPolicyV1_basic(t *testing.T) {
	var conf networkingv1.NetworkPolicy
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_network_policy_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNetworkPolicyV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNetworkPolicyV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesNetworkPolicyV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyV1Config_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.1", "webfront"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.0", "api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.port", "8125"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.namespace_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.namespace_selector.0.match_labels.name", "default"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyV1Config_endPorts(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.1", "webfront"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.0", "api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.port", "8126"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.end_port", "9000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.namespace_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.namespace_selector.0.match_labels.name", "default"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.0.port", "10000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.0.end_port", "65535"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyV1Config_specModified_allow_all_namespaces(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.1", "webfront"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.0", "api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.port", "8125"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.namespace_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.0.match_expressions.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.namespace_selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyV1Config_specModified_deny_other_namespaces(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.port", "8125"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.0.match_expressions.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyV1Config_specModified_pod_selector(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.1", "webfront"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.0", "api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.port", "statsd"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.ip_block.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.pod_selector.0.match_labels.app", "myapp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyV1Config_withEgress(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.1", "webfront"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.0", "api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.port", "statsd"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.ip_block.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.pod_selector.0.match_labels.app", "myapp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.0.port", "statsd"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.0.protocol", "UDP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.0", "Ingress"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.1", "Egress"),
				),
			},
		},
	})
}

func TestAccKubernetesNetworkPolicyV1_withEgressAtCreation(t *testing.T) {
	var conf networkingv1.NetworkPolicy
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_network_policy_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNetworkPolicyV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNetworkPolicyV1Config_withEgress(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.1", "webfront"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_selector.0.match_expressions.0.values.0", "api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.port", "statsd"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.ip_block.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.pod_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress.0.from.1.pod_selector.0.match_labels.app", "myapp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.0.port", "statsd"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.ports.0.protocol", "UDP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.egress.0.to.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.0", "Ingress"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.policy_types.1", "Egress"),
				),
			},
		},
	})
}

func testAccCheckKubernetesNetworkPolicyV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_network_policy_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Network Policy still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesNetworkPolicyV1Exists(n string, obj *networkingv1.NetworkPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesNetworkPolicyV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_network_policy_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"

    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }
  }

  spec {
    pod_selector {}

    policy_types = ["Ingress"]
  }
}
`, name)
}

func testAccKubernetesNetworkPolicyV1Config_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_network_policy_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"

    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
  }

  spec {
    pod_selector {}
    ingress {}
    policy_types = ["Ingress"]
  }
}
`, name)
}

func testAccKubernetesNetworkPolicyV1Config_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_network_policy_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  spec {
    pod_selector {
      match_expressions {
        key      = "name"
        operator = "In"
        values   = ["webfront", "api"]
      }
    }

    ingress {
      ports {
        port = "http"
      }
      ports {
        port     = "8125"
        protocol = "UDP"
      }

      from {
        namespace_selector {
          match_labels = {
            name = "default"
          }
        }
      }
    }

    policy_types = ["Ingress"]
  }
}
`, name)
}

func testAccKubernetesNetworkPolicyV1Config_endPorts(name string) string {
	return fmt.Sprintf(`resource "kubernetes_network_policy_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  spec {
    pod_selector {
      match_expressions {
        key      = "name"
        operator = "In"
        values   = ["webfront", "api"]
      }
    }

    ingress {
      ports {
        port     = "8126"
        protocol = "TCP"
        end_port = "9000"
      }

      from {
        namespace_selector {
          match_labels = {
            name = "default"
          }
        }
      }
    }
    egress {
      ports {
        port     = "10000"
        protocol = "TCP"
        end_port = "65535"
      }
    }
    policy_types = ["Ingress"]
  }
}
`, name)
}

func testAccKubernetesNetworkPolicyV1Config_specModified_allow_all_namespaces(name string) string {
	return fmt.Sprintf(`resource "kubernetes_network_policy_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  spec {
    pod_selector {
      match_expressions {
        key      = "name"
        operator = "In"
        values   = ["webfront", "api"]
      }
    }

    ingress {
      ports {
        port = "http"
      }

      ports {
        port     = "8125"
        protocol = "UDP"
      }

      from {
        namespace_selector {}
      }
    }
    policy_types = ["Ingress"]
  }
}
`, name)
}

func testAccKubernetesNetworkPolicyV1Config_specModified_deny_other_namespaces(name string) string {
	return fmt.Sprintf(`resource "kubernetes_network_policy_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  spec {
    pod_selector {}

    ingress {
      ports {
        port = "http"
      }

      ports {
        port     = "8125"
        protocol = "UDP"
      }

      from {
        pod_selector {}
      }
    }

    policy_types = ["Ingress"]
  }
}
`, name)
}
func testAccKubernetesNetworkPolicyV1Config_specModified_pod_selector(name string) string {
	return fmt.Sprintf(`resource "kubernetes_network_policy_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  spec {
    pod_selector {
      match_expressions {
        key      = "name"
        operator = "In"
        values   = ["webfront", "api"]
      }
    }

    ingress {
      ports {
        port     = "http"
        protocol = "TCP"
      }
      ports {
        port     = "statsd"
        protocol = "UDP"
      }
      from {
        ip_block {
          cidr = "10.0.0.0/8"
          except = [
            "10.0.0.0/24",
            "10.0.1.0/24",
          ]
        }
      }
      from {
        pod_selector {
          match_labels = {
            app = "myapp"
          }
        }
      }
    }

    policy_types = ["Ingress"]
  }
}
`, name)
}

func testAccKubernetesNetworkPolicyV1Config_withEgress(name string) string {
	return fmt.Sprintf(`resource "kubernetes_network_policy_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  spec {
    pod_selector {
      match_expressions {
        key      = "name"
        operator = "In"
        values   = ["webfront", "api"]
      }
    }

    ingress {
      ports {
        port     = "http"
        protocol = "TCP"
      }
      ports {
        port     = "statsd"
        protocol = "UDP"
      }
      from {
        ip_block {
          cidr = "10.0.0.0/8"
          except = [
            "10.0.0.0/24",
            "10.0.1.0/24",
          ]
        }
      }
      from {
        pod_selector {
          match_labels = {
            app = "myapp"
          }
        }
      }
    }

    egress {
      ports {
        port     = "statsd"
        protocol = "UDP"
      }
      to {
        ip_block {
          cidr = "10.0.0.0/8"
          except = [
            "10.0.0.0/24",
            "10.0.1.0/24",
          ]
        }
      }
    }

    policy_types = ["Ingress", "Egress"]
  }
}
`, name)
}
