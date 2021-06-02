package kubernetes

import (
	apps "k8s.io/api/apps/v1beta1"
	cert "k8s.io/api/certificates/v1beta1"
	api "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	networking "k8s.io/api/networking/v1beta1"
	rbac "k8s.io/api/rbac/v1beta1"
)

// knownLabelAnnotations is a map of known internal labels and
// annotations that we want to strip out to avoid unneccessary diffs.
// See: https://kubernetes.io/docs/reference/labels-annotations-taints
var knownLabelsAnnotations = map[string]string{
	// core
	api.LabelHostname:                            "",
	api.LabelZoneFailureDomain:                   "",
	api.LabelZoneRegion:                          "",
	api.LabelZoneFailureDomainStable:             "",
	api.LabelZoneRegionStable:                    "",
	api.LabelInstanceType:                        "",
	api.LabelInstanceTypeStable:                  "",
	api.LabelOSStable:                            "",
	api.LabelArchStable:                          "",
	api.LabelWindowsBuild:                        "",
	api.LabelNamespaceSuffixKubelet:              "",
	api.LabelNamespaceSuffixNode:                 "",
	api.LabelNamespaceNodeRestriction:            "",
	api.IsHeadlessService:                        "",
	api.BetaStorageClassAnnotation:               "",
	api.MountOptionAnnotation:                    "",
	api.ResourceDefaultNamespacePrefix:           "",
	api.ServiceAccountNameKey:                    "",
	api.ServiceAccountUIDKey:                     "",
	api.PodPresetOptOutAnnotationKey:             "",
	api.MirrorPodAnnotationKey:                   "",
	api.TolerationsAnnotationKey:                 "",
	api.TaintsAnnotationKey:                      "",
	api.SeccompPodAnnotationKey:                  "",
	api.SeccompContainerAnnotationKeyPrefix:      "",
	api.AppArmorBetaContainerAnnotationKeyPrefix: "",
	api.AppArmorBetaDefaultProfileAnnotationKey:  "",
	api.AppArmorBetaAllowedProfilesAnnotationKey: "",
	api.PreferAvoidPodsAnnotationKey:             "",
	api.NonConvertibleAnnotationPrefix:           "",
	api.AnnotationLoadBalancerSourceRangesKey:    "",
	api.EndpointsLastChangeTriggerTime:           "",
	api.MigratedPluginsAnnotationKey:             "",
	api.TaintNodeNotReady:                        "",
	api.TaintNodeUnreachable:                     "",
	api.TaintNodeUnschedulable:                   "",
	api.TaintNodeMemoryPressure:                  "",
	api.TaintNodeDiskPressure:                    "",
	api.TaintNodeNetworkUnavailable:              "",
	api.TaintNodePIDPressure:                     "",

	// networking
	networking.AnnotationIsDefaultIngressClass: "",

	// discovery
	discovery.LabelServiceName: "",
	discovery.LabelManagedBy:   "",
	discovery.LabelSkipMirror:  "",

	// certificates
	cert.KubeAPIServerClientSignerName:        "",
	cert.KubeAPIServerClientKubeletSignerName: "",
	cert.KubeletServingSignerName:             "",
	cert.LegacyUnknownSignerName:              "",

	// apps
	apps.StatefulSetPodNameLabel: "",

	// RBAC
	rbac.AutoUpdateAnnotationKey: "",

	// NOTE the annotations below are baked into the internal
	// controller package so we can't import their consts here

	// deployment
	"deployment.kubernetes.io/revision":         "",
	"deployment.kubernetes.io/revision-history": "",
	"deployment.kubernetes.io/desired-replicas": "",
	"deployment.kubernetes.io/max-replicas":     "",

	// persistentvolume
	"pv.kubernetes.io/bind-completed":               "",
	"pv.kubernetes.io/bound-by-controller":          "",
	"volume.kubernetes.io/selected-node":            "",
	"kubernetes.io/no-provisioner":                  "",
	"pv.kubernetes.io/provisioned-by":               "",
	"pv.kubernetes.io/migrated-to":                  "",
	"volume.beta.kubernetes.io/storage-provisioner": "",
	"volume.kubernetes.io/storage-resizer":          "",

	// GKE ingress
	"ingress.kubernetes.io/backends":              "",
	"ingress.kubernetes.io/https-forwarding-rule": "",
	"ingress.kubernetes.io/https-target-proxy":    "",
	"ingress.kubernetes.io/forwarding-rule":       "",
	"ingress.kubernetes.io/target-proxy":          "",
	"ingress.kubernetes.io/ssl-cert":              "",
	"ingress.kubernetes.io/url-map":               "",

	"deprecated.daemonset.template.generation": "",
}
