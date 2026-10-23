// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

// subjectsEqual is order-sensitive because subject is a list.
func subjectsEqual(a, b []SubjectModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Kind.Equal(b[i].Kind) ||
			!a[i].Name.Equal(b[i].Name) ||
			!a[i].APIGroup.Equal(b[i].APIGroup) ||
			!a[i].Namespace.Equal(b[i].Namespace) {
			return false
		}
	}
	return true
}
