// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"net"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"k8s.io/apimachinery/pkg/api/resource"
	apiValidation "k8s.io/apimachinery/pkg/api/validation"
	utilValidation "k8s.io/apimachinery/pkg/util/validation"
)

type podStringRule string

func (v podStringRule) Description(context.Context) string {
	return "must satisfy the Kubernetes " + string(v) + " constraint"
}
func (v podStringRule) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }
func (v podStringRule) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	var messages []string
	switch v {
	case "ip":
		if net.ParseIP(s) == nil {
			messages = append(messages, "must be a valid IP address")
		}
	case "name":
		messages = apiValidation.NameIsDNSSubdomain(s, false)
	case "port":
		if n, err := strconv.Atoi(s); err == nil {
			messages = utilValidation.IsValidPortNum(n)
		} else {
			messages = utilValidation.IsValidPortName(s)
		}
	case "nullable-int":
		if s != "" {
			if _, err := strconv.ParseInt(s, 10, 64); err != nil {
				messages = append(messages, "must be an integer or an empty string: "+err.Error())
			}
		}
	case "mode":
		n, err := strconv.ParseInt(s, 8, 32)
		if !strings.HasPrefix(s, "0") || err != nil || n < 0 || n > 0777 {
			messages = append(messages, "must be an octal numeral starting with 0 between 0 and 0777")
		}
	case "path":
		if path.IsAbs(s) || strings.HasPrefix(s, "..") {
			messages = append(messages, "must be a relative path that does not start with '..'")
		}
		for _, part := range strings.Split(filepath.ToSlash(s), "/") {
			if part == ".." {
				messages = append(messages, "must not contain a '..' path element")
				break
			}
		}
	case "quantity":
		if _, err := resource.ParseQuantity(s); err != nil {
			messages = append(messages, "must be a Kubernetes resource quantity: "+err.Error())
		}
	}
	for _, message := range messages {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value", message)
	}
}

type podRequiredReference struct{ child string }

func (v podRequiredReference) Description(context.Context) string {
	return "each object must provide " + v.child
}
func (v podRequiredReference) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v podRequiredReference) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for i, element := range req.ConfigValue.Elements() {
		if element.IsUnknown() {
			continue
		}
		if element.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtListIndex(i), "Missing Required Argument", "The reference must be an object.")
			continue
		}
		object := element.(types.Object)
		child := object.Attributes()[v.child]
		if child.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtListIndex(i).AtName(v.child), "Missing Required Argument", fmt.Sprintf("The argument %q is required.", v.child))
		}
	}
}
