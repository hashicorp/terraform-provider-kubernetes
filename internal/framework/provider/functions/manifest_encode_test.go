// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestManifestEncode(t *testing.T) {
	t.Parallel()

	outputName := "test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testManifestEncodeConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput(outputName, `apiVersion: v1
data:
  test: test
kind: ConfigMap
metadata:
  name: test
  namespace: null
`),
				),
			},
		},
	})
}

func TestManifestEncodeMulti(t *testing.T) {
	t.Parallel()

	outputName := "test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testManifestEncodeMultiConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput(outputName, `---
apiVersion: v1
data:
  test: test
immutable: false
kind: ConfigMap
metadata:
  name: test
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    k8s-app: fluentd-logging
  name: fluentd-elasticsearch2
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: fluentd-elasticsearch
  template:
    metadata:
      labels:
        name: fluentd-elasticsearch
        something: helloworld
    spec:
      containers:
      - image: quay.io/fluentd_elasticsearch/fluentd:v2.5.2
        name: fluentd-elasticsearch
        resources:
          limits:
            cpu: 1.5
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 200Mi
        volumeMounts:
        - mountPath: /var/log
          name: varlog
      terminationGracePeriodSeconds: 30
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
        operator: Exists
      volumes:
      - hostPath:
          path: /var/log
        name: varlog
`),
				),
			},
		},
	})
}

func testManifestEncodeConfig() string {
	return `
locals {
  single_manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata   = {
      name = "test"
      namespace = null
    }
    data = {
      "test" = "test"
    }
  }
}

output "test" {
  value = provider::kubernetes::manifest_encode(local.single_manifest)
}`
}

func testManifestEncodeMultiConfig() string {
	return `
locals {
  multi_manifest = [
    {
      apiVersion = "v1"
      kind       = "ConfigMap"
      metadata   = {
        name = "test"
      }
      data = {
        "test" = "test"
      }
      immutable = false
    },
    {
      apiVersion = "apps/v1"
      kind = "DaemonSet"
      metadata = {
        name      = "fluentd-elasticsearch2"
        namespace = "kube-system"
        labels    = {
          "k8s-app" = "fluentd-logging"
        }
      }
      spec = {
        selector = {
          matchLabels = {
            name = "fluentd-elasticsearch"
          }
        }
        template = {
          metadata = {
            labels = {
              "something" = "helloworld"
              "name"      = "fluentd-elasticsearch"
            }
          }
          spec = {
            tolerations = [
              {
                key      = "node-role.kubernetes.io/control-plane"
                operator = "Exists"
                effect   = "NoSchedule"
              },
              {
                key      = "node-role.kubernetes.io/master"
                operator = "Exists"
                effect   = "NoSchedule"
              }
            ]
            containers = [
              {
                name      = "fluentd-elasticsearch"
                image     = "quay.io/fluentd_elasticsearch/fluentd:v2.5.2"
                resources = {
                  limits = {
                    cpu    = 1.5
                    memory = "200Mi"
                  }
                  requests = {
                    cpu    = "100m"
                    memory = "200Mi"
                  }
                }
                volumeMounts = [
                  {
                    mountPath = "/var/log"
                    name      = "varlog"
                  }
                ]
              }
            ]
            terminationGracePeriodSeconds = 30
            volumes = [
              {
                name = "varlog"
                hostPath = {
                  path = "/var/log"
                }
              }
            ]
          }
        }
      }
    }
  ]
}

output "test" {
  value = provider::kubernetes::manifest_encode(local.multi_manifest)
}`
}
