// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mitchellh/go-testing-interface"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol6WithError(New("test")),
}

func testAccPreCheck(t testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

func ParallelTest(t testing.T, tc resource.TestCase) {
	if os.Getenv("TF_X_KUBERNETES_CODEGEN_PLUGIN6") == "1" {
		tc.ProviderFactories = nil
		tc.ProtoV6ProviderFactories = testAccProtoV6ProviderFactories
		for i, ri := range tc.IDRefreshIgnore {
			tc.IDRefreshIgnore[i] = singleNestedPath(ri)
		}
		for i, step := range tc.Steps {
			for i, vi := range step.ImportStateVerifyIgnore {
				step.ImportStateVerifyIgnore[i] = singleNestedPath(vi)
			}
			tc.Steps[i].Config = convertBlockToObject(step.Config)
		}

		// The PreCheck func checks for k8s env vars and configures the singleton instance testAccProvider
		// this is needed for CheckDestroy functions which use the singleton for a k8s connection.
		// The internal/framework currently only supports KUBE_CONFIG_PATHS (plural) which is not in the PreCheck
		// temporarily set KUBE_CONFIG_PATH to avoid failure (which is in the PreCheck)
		os.Setenv("KUBE_CONFIG_PATH", os.Getenv("KUBE_CONFIG_PATHS"))
		defer os.Unsetenv("KUBE_CONFIG_PATH")
	}

	resource.ParallelTest(t, tc)
}

func TestCheckResourceAttr(name, key, value string) resource.TestCheckFunc {
	if os.Getenv("TF_X_KUBERNETES_CODEGEN_PLUGIN6") == "1" {
		key = singleNestedPath(key)
	}

	return resource.TestCheckResourceAttr(name, key, value)
}

func TestCheckResourceAttrSet(name, key string) resource.TestCheckFunc {
	if os.Getenv("TF_X_KUBERNETES_CODEGEN_PLUGIN6") == "1" {
		key = singleNestedPath(key)
	}

	return resource.TestCheckResourceAttrSet(name, key)
}

func singleNestedPath(path string) string {
	var newParts []string
	parts := strings.Split(path, ".")
	for _, p := range parts {
		if !isInt(p) {
			newParts = append(newParts, p)
		}
	}
	return strings.Join(newParts, ".")
}

func isInt(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

func convertBlockToObject(config string) string {
	// Regular expression to match block declarations, excluding top-level blocks
	re := regexp.MustCompile(`(?m)^(\s+)(\w+)\s*{`)

	// Replace block declarations with object assignments
	return re.ReplaceAllStringFunc(config, func(match string) string {
		parts := re.FindStringSubmatch(match)
		indentation := parts[1]
		blockName := parts[2]

		// List of top-level blocks that should not be converted
		topLevelBlocks := []string{"resource", "data", "module", "variable", "output", "locals", "terraform"}

		for _, topBlock := range topLevelBlocks {
			if blockName == topBlock {
				return match // Return the original match for top-level blocks
			}
		}

		return indentation + blockName + " = {"
	})
}
