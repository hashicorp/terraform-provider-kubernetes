// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubemanifest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// DefaultOpTimeout is the fallback for create/update/delete when the user does not set
// a timeouts{} block.
const DefaultOpTimeout = 20 * time.Minute

// OpTimeout resolves the effective timeout for a CRUD operation. It never returns a
// value shorter than floor, so a long readiness wait cannot be cut off by the (shorter)
// default operation timeout. Pass floor=0 when there is no readiness wait.
func OpTimeout(ctx context.Context, tv timeouts.Value, kind string, floor time.Duration) (time.Duration, diag.Diagnostics) {
	var d time.Duration
	var diags diag.Diagnostics
	switch kind {
	case "create":
		d, diags = tv.Create(ctx, DefaultOpTimeout)
	case "update":
		d, diags = tv.Update(ctx, DefaultOpTimeout)
	case "delete":
		d, diags = tv.Delete(ctx, DefaultOpTimeout)
	default:
		d = DefaultOpTimeout
	}
	if floor > d {
		d = floor
	}
	return d, diags
}

// ApplyErrDiag turns a failed apply into a helpful (summary, detail) diagnostic. SSA
// field-ownership conflicts (HTTP 409) are the common case and get actionable guidance.
// resource is the Terraform resource type name used to prefix the summary.
func ApplyErrDiag(resource string, err error) (string, string) {
	if apierrors.IsConflict(err) {
		return resource + ": field manager conflict",
			fmt.Sprintf("Server-Side Apply reported a field-ownership conflict:\n\n%s\n\n"+
				"Another field manager already owns one or more of these fields. Either set "+
				"`force_conflicts = true` to take ownership, or use a distinct `field_manager` "+
				"if you intend to co-own the object with another controller.", err.Error())
	}
	return resource + ": apply failed", err.Error()
}

// IsImmutableErr reports whether an apply error is a Kubernetes rejection of an update
// to an immutable field (which requires object replacement, not update).
func IsImmutableErr(err error) bool {
	if err == nil || (!apierrors.IsInvalid(err) && !apierrors.IsBadRequest(err) && !apierrors.IsForbidden(err)) {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{
		"immutable",
		"are forbidden",
		"is forbidden: updates to",
		"cannot be changed",
		"may not be changed",
		"field is immutable",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// WaitForDeleted blocks until the named object is gone (NotFound) or ctx expires. On
// timeout it surfaces any finalizers still holding the object.
func WaitForDeleted(ctx context.Context, ri dynamic.ResourceInterface, name string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		_, err := ri.Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		select {
		case <-ctx.Done():
			if live, gerr := ri.Get(context.Background(), name, metav1.GetOptions{}); gerr == nil {
				if fin, _, _ := unstructured.NestedStringSlice(live.Object, "metadata", "finalizers"); len(fin) > 0 {
					return fmt.Errorf("timed out waiting for %q to be deleted; object still has finalizers: %s",
						name, strings.Join(fin, ", "))
				}
			}
			return fmt.Errorf("timed out waiting for %q to be deleted", name)
		case <-ticker.C:
		}
	}
}
