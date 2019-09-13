package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesNetworkPolicy_basic(t *testing.T) {
	var conf api.NetworkPolicy
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_network_policy.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesNetworkPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNetworkPolicyConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyExists("kubernetes_network_policy.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one"}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.TestLabelFour", "four"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three", "TestLabelFour": "four"}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyConfig_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyExists("kubernetes_network_policy.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyConfig_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyExists("kubernetes_network_policy.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.1742479128", "webfront"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.2902841359", "api"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.port", "8125"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.namespace_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.namespace_selector.0.match_labels.name", "default"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyConfig_specModified_allow_all_namespaces(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyExists("kubernetes_network_policy.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.1742479128", "webfront"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.2902841359", "api"),
					resource.TestCheckNoResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_labels"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.port", "8125"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.namespace_selector.#", "1"),
					resource.TestCheckNoResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.pod_selector.0.match_expressions"),
					resource.TestCheckNoResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.namespace_selector.0.match_labels"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyConfig_specModified_deny_other_namespaces(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyExists("kubernetes_network_policy.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.#", "1"),
					resource.TestCheckNoResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions"),
					resource.TestCheckNoResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_labels"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.port", "8125"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.pod_selector.#", "1"),
					resource.TestCheckNoResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.pod_selector.0.match_expressions"),
					resource.TestCheckNoResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.pod_selector.0.match_labels"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyConfig_specModified_pod_selector(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyExists("kubernetes_network_policy.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.1742479128", "webfront"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.2902841359", "api"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.port", "statsd"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.ip_block.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.pod_selector.0.match_labels.app", "myapp"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.0", "Ingress"),
				),
			},
			{
				Config: testAccKubernetesNetworkPolicyConfig_withEgress(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyExists("kubernetes_network_policy.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.1742479128", "webfront"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.2902841359", "api"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.port", "statsd"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.ip_block.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.pod_selector.0.match_labels.app", "myapp"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.ports.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.ports.0.port", "statsd"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.ports.0.protocol", "UDP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.0", "Ingress"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.1", "Egress"),
				),
			},
		},
	})
}

func TestAccKubernetesNetworkPolicy_withEgressAtCreation(t *testing.T) {
	var conf api.NetworkPolicy
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_network_policy.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesNetworkPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNetworkPolicyConfig_withEgress(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNetworkPolicyExists("kubernetes_network_policy.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_network_policy.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.1742479128", "webfront"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.pod_selector.0.match_expressions.0.values.2902841359", "api"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.port", "http"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.port", "statsd"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.ports.1.protocol", "UDP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.ip_block.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.pod_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.ingress.0.from.1.pod_selector.0.match_labels.app", "myapp"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.ports.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.ports.0.port", "statsd"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.ports.0.protocol", "UDP"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.0.cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.0.except.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.0.except.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.ip_block.0.except.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.egress.0.to.0.pod_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.0", "Ingress"),
					resource.TestCheckResourceAttr("kubernetes_network_policy.test", "spec.0.policy_types.1", "Egress"),
				),
			},
		},
	})
}

func TestAccKubernetesNetworkPolicy_importBasic(t *testing.T) {
	resourceName := "kubernetes_network_policy.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesNetworkPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNetworkPolicyConfig_basic(name),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckKubernetesNetworkPolicyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_network_policy" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.NetworkingV1().NetworkPolicies(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Network Policy still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesNetworkPolicyExists(n string, obj *api.NetworkPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.NetworkingV1().NetworkPolicies(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesNetworkPolicyConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_network_policy" "test" {
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

func testAccKubernetesNetworkPolicyConfig_metaModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_network_policy" "test" {
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
    ingress      {}
	  policy_types = [ "Ingress" ]
  }
}
`, name)
}

func testAccKubernetesNetworkPolicyConfig_specModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_network_policy" "test" {
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

func testAccKubernetesNetworkPolicyConfig_specModified_allow_all_namespaces(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_network_policy" "test" {
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
      }

      ports {
        port     = "8125"
        protocol = "UDP"
      }

      from {
        namespace_selector {}
      }
    }
    policy_types = [ "Ingress" ]
  }
}
	`, name)
}

func testAccKubernetesNetworkPolicyConfig_specModified_deny_other_namespaces(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_network_policy" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  spec {
    pod_selector {}

    ingress {
      ports {
        port     = "http"
      }

      ports {
        port     = "8125"
        protocol = "UDP"
      }

      from {
          pod_selector {}
      }
    }

    policy_types = [ "Ingress" ]
  }
}
	`, name)
}
func testAccKubernetesNetworkPolicyConfig_specModified_pod_selector(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_network_policy" "test" {
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

func testAccKubernetesNetworkPolicyConfig_withEgress(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_network_policy" "test" {
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
