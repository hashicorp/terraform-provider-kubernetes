// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKeepNumberSpelling(t *testing.T) {
	t.Parallel()

	keepInt, keepOctal, keepIntOrString := KeepIntSpelling, KeepOctalSpelling, KeepIntOrStringSpelling
	for name, tc := range map[string]struct {
		keep           func(prior, current types.String) types.String
		prior, current types.String
		want           types.String
	}{
		"leading zero":           {keepInt, types.StringValue("01000"), types.StringValue("1000"), types.StringValue("01000")},
		"plus sign":              {keepInt, types.StringValue("+1000"), types.StringValue("1000"), types.StringValue("+1000")},
		"changed value":          {keepInt, types.StringValue("01000"), types.StringValue("2000"), types.StringValue("2000")},
		"removed value":          {keepInt, types.StringValue("01000"), types.StringValue(""), types.StringValue("")},
		"null prior":             {keepInt, types.StringNull(), types.StringValue("1000"), types.StringValue("1000")},
		"unknown prior":          {keepInt, types.StringUnknown(), types.StringValue("1000"), types.StringValue("1000")},
		"octal leading zeros":    {keepOctal, types.StringValue("00644"), types.StringValue("0644"), types.StringValue("00644")},
		"octal changed":          {keepOctal, types.StringValue("00644"), types.StringValue("0600"), types.StringValue("0600")},
		"int or string padded":   {keepIntOrString, types.StringValue("01"), types.StringValue("1"), types.StringValue("01")},
		"int or string percent":  {keepIntOrString, types.StringValue("25%"), types.StringValue("25%"), types.StringValue("25%")},
		"int is not its percent": {keepIntOrString, types.StringValue("01"), types.StringValue("1%"), types.StringValue("1%")},
		"padded percent differs": {keepIntOrString, types.StringValue("01%"), types.StringValue("1%"), types.StringValue("1%")},
	} {
		t.Run(name, func(t *testing.T) {
			if got := tc.keep(tc.prior, tc.current); !got.Equal(tc.want) {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
