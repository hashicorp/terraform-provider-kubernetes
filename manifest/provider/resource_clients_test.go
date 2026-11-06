// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"sort"
	"testing"
)

func TestConnectionBlockIsWriteOnlyThroughout(t *testing.T) {
	block := connectionBlockSchema().Block

	declared := map[string][]string{"": {}}
	for _, attr := range block.Attributes {
		if !attr.WriteOnly {
			t.Errorf("connection.%s is not write-only", attr.Name)
			continue
		}
		declared[""] = append(declared[""], attr.Name)
	}
	for _, nested := range block.BlockTypes {
		declared[nested.TypeName] = []string{}
		for _, attr := range nested.Block.Attributes {
			if !attr.WriteOnly {
				t.Errorf("connection.%s.%s is not write-only", nested.TypeName, attr.Name)
				continue
			}
			declared[nested.TypeName] = append(declared[nested.TypeName], attr.Name)
		}
	}

	if len(declared) != len(writeOnlyConnectionAttributes) {
		t.Fatalf("schema declares %d blocks, writeOnlyConnectionAttributes covers %d",
			len(declared), len(writeOnlyConnectionAttributes))
	}
	for name, got := range declared {
		want, ok := writeOnlyConnectionAttributes[name]
		if !ok {
			t.Errorf("writeOnlyConnectionAttributes has no entry for block %q", name)
			continue
		}
		got, want = append([]string{}, got...), append([]string{}, want...)
		sort.Strings(got)
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("block %q: schema has %v, writeOnlyConnectionAttributes has %v", name, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("block %q: schema has %v, writeOnlyConnectionAttributes has %v", name, got, want)
				break
			}
		}
	}
}
