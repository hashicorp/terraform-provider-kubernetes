// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

func serviceAccountAutomaticTokens(conn *clientset.Clientset) (bool, error) {
	server, err := conn.ServerVersion()
	if err != nil {
		return false, err
	}
	actual, err := version.NewVersion(server.String())
	if err != nil {
		return false, err
	}
	return actual.LessThan(version.Must(version.NewVersion("1.24.0"))), nil
}

func serviceAccountWaitForDefaultSecret(ctx context.Context, conn *clientset.Clientset, name string, config corev1.ServiceAccount, timeout time.Duration) (string, error) {
	automatic, err := serviceAccountAutomaticTokens(conn)
	if err != nil || !automatic {
		return "", err
	}
	configured := make(map[string]bool, len(config.Secrets))
	for _, reference := range config.Secrets {
		configured[reference.Name] = true
	}
	var tokenName string
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		account, err := conn.CoreV1().ServiceAccounts(config.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return retry.NonRetryableError(err)
		}
		if len(account.Secrets) == len(config.Secrets) {
			return retry.RetryableError(fmt.Errorf("waiting for default secret of %q to appear", config.Namespace+"/"+name))
		}
		candidates := make(map[string]bool)
		for _, reference := range account.Secrets {
			if !configured[reference.Name] {
				candidates[reference.Name] = true
			}
		}
		secrets, err := conn.CoreV1().Secrets(config.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("type=%s", corev1.SecretTypeServiceAccountToken)})
		if err != nil {
			return retry.NonRetryableError(err)
		}
		var matches []string
		for _, secret := range secrets.Items {
			if candidates[secret.Name] && secret.Type == corev1.SecretTypeServiceAccountToken {
				matches = append(matches, secret.Name)
			}
		}
		switch len(matches) {
		case 0:
			return retry.RetryableError(fmt.Errorf("expected 1 generated service account token, 0 found"))
		case 1:
			tokenName = matches[0]
			return nil
		default:
			return retry.NonRetryableError(fmt.Errorf("expected 1 generated service account token, %d found", len(matches)))
		}
	})
	return tokenName, err
}

func serviceAccountDiscoverDefaultSecret(ctx context.Context, conn *clientset.Clientset, account *corev1.ServiceAccount) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	// Disabling automatic token creation does not remove tokens retained from
	// older clusters. Import must discover those references regardless of version.
	var matches []string
	for _, reference := range account.Secrets {
		if !strings.HasPrefix(reference.Name, account.Name+"-token-") {
			continue
		}
		secret, err := conn.CoreV1().Secrets(account.Namespace).Get(ctx, reference.Name, metav1.GetOptions{})
		if err != nil {
			diags.AddWarning("Unable to discover default service account token", fmt.Sprintf("Unable to fetch secret %s/%s: %s", account.Namespace, reference.Name, err))
			return "", diags
		}
		if secret.Type == corev1.SecretTypeServiceAccountToken &&
			secret.Annotations[corev1.ServiceAccountNameKey] == account.Name &&
			secret.Annotations[corev1.ServiceAccountUIDKey] == string(account.UID) {
			matches = append(matches, reference.Name)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], diags
	case 0:
		// Token Secrets are not created automatically on modern Kubernetes clusters.
	default:
		diags.AddWarning("Unable to discover default secret name.", "There is more than one service account token associated to the service account.")
	}
	return "", diags
}
