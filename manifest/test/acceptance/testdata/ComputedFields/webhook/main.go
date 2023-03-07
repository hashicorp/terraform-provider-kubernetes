// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
)

type config struct {
	certFile string
	keyFile  string
}

func run() error {
	logrusLogEntry := logrus.NewEntry(logrus.New())
	logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	logger := kwhlogrus.NewLogrus(logrusLogEntry)

	cfg := config{
		certFile: "/etc/webhook/certs/cert.pem",
		keyFile:  "/etc/webhook/certs/key.pem",
	}

	mt := mutating.MutatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}

		_, sigil := annotations["tf-k8s-acc"]
		if sigil {
			annotations["mutated"] = "true"
			obj.SetAnnotations(annotations)
		}

		return &mutating.MutatorResult{MutatedObject: obj}, nil
	})

	mcfg := mutating.WebhookConfig{
		ID:      "obj-annotate",
		Mutator: mt,
		Logger:  logger,
	}
	wh, err := mutating.NewWebhook(mcfg)
	if err != nil {
		return fmt.Errorf("error creating webhook: %w", err)
	}

	h, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: wh, Logger: logger})
	if err != nil {
		return fmt.Errorf("error creating webhook handler: %w", err)
	}

	logger.Infof("Listening on :8080")
	err = http.ListenAndServeTLS(":8080", cfg.certFile, cfg.keyFile, h)
	if err != nil {
		return fmt.Errorf("error serving webhook: %w", err)
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running app: %s\n", err)
		os.Exit(1)
	}
}
