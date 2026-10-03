// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

// Options selects the per-resource variations of the PodSpec. It must stay
// comparable: For builds each distinct value once.
type Options struct {
	// Immutable forces replacement for every field SDKv2 declared ForceNew: !isUpdatable.
	Immutable bool
	// Template selects pod-template semantics, which keep built-in tolerations on read.
	Template bool
	// RestartPolicy is the default of restart_policy.
	RestartPolicy string
	// RestartPolicyAlwaysOnly accepts only "Always" for restart_policy.
	RestartPolicyAlwaysOnly bool
	// SpecRequired requires exactly one spec block.
	SpecRequired bool
	// SpecForceNew forces replacement when the spec block is added or removed.
	SpecForceNew bool
}

// Deployment is a workload template whose pods must restart Always.
func Deployment() Options {
	return Options{Template: true, RestartPolicy: "Always", RestartPolicyAlwaysOnly: true, SpecRequired: true}
}

// DaemonSet is an updatable workload template with an optional spec block.
func DaemonSet() Options {
	return Options{Template: true, RestartPolicy: "Always"}
}

// StatefulSet is an updatable workload template.
func StatefulSet() Options {
	return Options{Template: true, RestartPolicy: "Always", SpecRequired: true}
}

// Pod is a bare, immutable pod spec.
func Pod() Options {
	return Options{Immutable: true, RestartPolicy: "Always", SpecRequired: true}
}

// Job is an immutable pod template that defaults to never restarting.
func Job() Options {
	return Options{Immutable: true, Template: true, RestartPolicy: "Never", SpecForceNew: true}
}

// CronJob's job template is updatable, unlike a Job's.
func CronJob() Options {
	return Options{Template: true, RestartPolicy: "Never", SpecForceNew: true}
}

// forceNew is a field's SDKv2 ForceNew declaration.
type forceNew uint8

const (
	updatable forceNew = iota // no ForceNew
	immutable                 // ForceNew: !isUpdatable
	alwaysNew                 // ForceNew: true
)
