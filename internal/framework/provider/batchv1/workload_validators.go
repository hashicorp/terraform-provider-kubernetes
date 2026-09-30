// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"net"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"k8s.io/apimachinery/pkg/api/resource"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
)

type workloadStringValidator struct {
	description string
	validate    func(string) []string
}

func (v workloadStringValidator) Description(context.Context) string {
	return v.description
}

func (v workloadStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v workloadStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, message := range v.validate(req.ConfigValue.ValueString()) {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(req.Path, message, req.ConfigValue.String()))
	}
}

func workloadPortValidator() validator.String {
	return workloadStringValidator{
		description: "must be a port number from 1 to 65535 or a valid port name",
		validate: func(value string) []string {
			if port, err := strconv.Atoi(value); err == nil {
				return utilvalidation.IsValidPortNum(port)
			}
			return utilvalidation.IsValidPortName(value)
		},
	}
}

func workloadIPAddressValidator() validator.String {
	return workloadStringValidator{
		description: "must be a valid IPv4 or IPv6 address",
		validate: func(value string) []string {
			if net.ParseIP(value) == nil {
				return []string{"must be a valid IP address"}
			}
			return nil
		},
	}
}

func workloadNullableIntValidator() validator.String {
	return workloadStringValidator{
		description: "must be empty or a signed 64-bit integer",
		validate: func(value string) []string {
			if value == "" {
				return nil
			}
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return []string{err.Error()}
			}
			return nil
		},
	}
}

func workloadModeBitsValidator() validator.String {
	return workloadStringValidator{
		description: "must be an octal numeral from 0 to 0777 starting with 0",
		validate: func(value string) []string {
			var messages []string
			if !strings.HasPrefix(value, "0") {
				messages = append(messages, "must start with 0 (octal numeral)")
			}
			mode, err := strconv.ParseInt(value, 8, 32)
			if err != nil {
				messages = append(messages, err.Error())
			}
			if mode < 0 || mode > 0777 {
				messages = append(messages, "must be between 0 and 0777")
			}
			return messages
		},
	}
}

func workloadPathValidator() validator.String {
	return workloadStringValidator{
		description: "must be relative, must not contain a .. component, and must not start with ..",
		validate: func(value string) []string {
			if path.IsAbs(value) {
				return []string{"must be a relative path"}
			}
			for _, part := range strings.Split(filepath.ToSlash(value), "/") {
				if part == ".." {
					return []string{"must not contain a .. path component"}
				}
			}
			if strings.HasPrefix(value, "..") {
				return []string{"must not start with .."}
			}
			return nil
		},
	}
}

func workloadQuantityValidator() validator.String {
	return workloadStringValidator{
		description: "must be a valid Kubernetes resource quantity",
		validate: func(value string) []string {
			if _, err := resource.ParseQuantity(value); err != nil {
				return []string{err.Error()}
			}
			return nil
		},
	}
}
