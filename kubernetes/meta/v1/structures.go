package v1

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/structures"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IdParts(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		err := fmt.Errorf("Unexpected ID format (%q), expected %q.", id, "namespace/name")
		return "", "", err
	}

	return parts[0], parts[1], nil
}

func BuildId(meta metav1.ObjectMeta) string {
	return meta.Namespace + "/" + meta.Name
}

func BuildIdWithVersionKind(meta metav1.ObjectMeta, apiVersion, kind string) string {
	id := fmt.Sprintf("apiVersion=%v,kind=%v,name=%s",
		apiVersion, kind, meta.Name)
	if meta.Namespace != "" {
		id += fmt.Sprintf(",namespace=%v", meta.Namespace)
	}
	return id
}

func ExpandMetadata(in []interface{}) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{}
	if len(in) < 1 {
		return meta
	}
	m := in[0].(map[string]interface{})

	if v, ok := m["annotations"].(map[string]interface{}); ok && len(v) > 0 {
		meta.Annotations = structures.ExpandStringMap(m["annotations"].(map[string]interface{}))
	}

	if v, ok := m["labels"].(map[string]interface{}); ok && len(v) > 0 {
		meta.Labels = structures.ExpandStringMap(m["labels"].(map[string]interface{}))
	}

	if v, ok := m["generate_name"]; ok {
		meta.GenerateName = v.(string)
	}
	if v, ok := m["name"]; ok {
		meta.Name = v.(string)
	}
	if v, ok := m["namespace"]; ok {
		meta.Namespace = v.(string)
	}

	return meta
}

func PatchMetadata(keyPrefix, pathPrefix string, d *schema.ResourceData) structures.PatchOperations {
	ops := make([]structures.PatchOperation, 0, 0)
	if d.HasChange(keyPrefix + "annotations") {
		oldV, newV := d.GetChange(keyPrefix + "annotations")
		diffOps := structures.DiffStringMap(pathPrefix+"annotations", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}
	if d.HasChange(keyPrefix + "labels") {
		oldV, newV := d.GetChange(keyPrefix + "labels")
		diffOps := structures.DiffStringMap(pathPrefix+"labels", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}
	return ops
}

func FlattenMetadata(meta metav1.ObjectMeta, d *schema.ResourceData, providerMetadata interface{}, metaPrefix ...string) []interface{} {
	m := make(map[string]interface{})
	prefix := ""
	if len(metaPrefix) > 0 {
		prefix = metaPrefix[0]
	}

	configAnnotations := d.Get(prefix + "metadata.0.annotations").(map[string]interface{})
	ignoreAnnotations := providerMetadata.(provider.Meta).IgnoredAnnotations()
	annotations := removeInternalKeys(meta.Annotations, configAnnotations)
	m["annotations"] = removeKeys(annotations, configAnnotations, ignoreAnnotations)
	if meta.GenerateName != "" {
		m["generate_name"] = meta.GenerateName
	}

	configLabels := d.Get(prefix + "metadata.0.labels").(map[string]interface{})
	ignoreLabels := providerMetadata.(provider.Meta).IgnoredLabels()
	labels := removeInternalKeys(meta.Labels, configLabels)
	m["labels"] = removeKeys(labels, configLabels, ignoreLabels)
	m["name"] = meta.Name
	m["resource_version"] = meta.ResourceVersion
	m["uid"] = fmt.Sprintf("%v", meta.UID)
	m["generation"] = meta.Generation

	if meta.Namespace != "" {
		m["namespace"] = meta.Namespace
	}

	return []interface{}{m}
}

func removeInternalKeys(m map[string]string, d map[string]interface{}) map[string]string {
	for k := range m {
		if IsInternalKey(k) && !IsKeyInMap(k, d) {
			delete(m, k)
		}
	}
	return m
}

// removeKeys removes given Kubernetes metadata(annotations and labels) keys.
// In that case, they won't be available in the TF state file and will be ignored during apply/plan operations.
func removeKeys(m map[string]string, d map[string]interface{}, ignoreKubernetesMetadataKeys []string) map[string]string {
	for k := range m {
		if ignoreKey(k, ignoreKubernetesMetadataKeys) && !IsKeyInMap(k, d) {
			delete(m, k)
		}
	}
	return m
}

func IsKeyInMap(key string, d map[string]interface{}) bool {
	if d == nil {
		return false
	}
	for k := range d {
		if k == key {
			return true
		}
	}
	return false
}

func IsInternalKey(annotationKey string) bool {
	u, err := url.Parse("//" + annotationKey)
	if err != nil {
		return false
	}

	// allow user specified application specific keys
	if u.Hostname() == "app.kubernetes.io" {
		return false
	}

	// allow AWS load balancer configuration annotations
	if u.Hostname() == "service.beta.kubernetes.io" {
		return false
	}

	// internal *.kubernetes.io keys
	if strings.HasSuffix(u.Hostname(), "kubernetes.io") {
		return true
	}

	// Specific to DaemonSet annotations, generated & controlled by the server.
	if strings.Contains(annotationKey, "deprecated.daemonset.template.generation") {
		return true
	}
	return false
}

// ignoreKey reports whether the Kubernetes metadata(annotations and labels) key contains
// any match of the regular expression pattern from the expressions slice.
func ignoreKey(key string, expressions []string) bool {
	for _, e := range expressions {
		if ok, _ := regexp.MatchString(e, key); ok {
			return true
		}
	}

	return false
}

func FlattenLabelSelectorRequirementList(l []metav1.LabelSelectorRequirement) []interface{} {
	att := make([]map[string]interface{}, len(l))
	for i, v := range l {
		m := map[string]interface{}{}
		m["key"] = v.Key
		m["values"] = structures.NewStringSet(schema.HashString, v.Values)
		m["operator"] = string(v.Operator)
		att[i] = m
	}
	return []interface{}{att}
}
