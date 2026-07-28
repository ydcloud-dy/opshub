package rbac

import (
	"testing"

	rbacbiz "github.com/ydcloud-dy/opshub/internal/biz/rbac"
)

func TestCollectDepartmentScopeIDs(t *testing.T) {
	departments := []*rbacbiz.SysDepartment{
		department(1, 0),
		department(2, 1),
		department(3, 2),
		department(4, 1),
		department(5, 0),
	}

	tests := []struct {
		name string
		root uint
		want []uint
	}{
		{name: "root includes every descendant", root: 1, want: []uint{1, 2, 4, 3}},
		{name: "nested department includes its subtree", root: 2, want: []uint{2, 3}},
		{name: "leaf includes itself", root: 3, want: []uint{3}},
		{name: "unrelated organization stays isolated", root: 5, want: []uint{5}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := collectDepartmentScopeIDs(departments, test.root)
			if !equalUintSlices(got, test.want) {
				t.Fatalf("collectDepartmentScopeIDs() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCollectDepartmentScopeIDsStopsAtCycle(t *testing.T) {
	departments := []*rbacbiz.SysDepartment{
		department(6, 7),
		department(7, 6),
	}

	got := collectDepartmentScopeIDs(departments, 6)
	want := []uint{6, 7}
	if !equalUintSlices(got, want) {
		t.Fatalf("collectDepartmentScopeIDs() = %v, want %v", got, want)
	}
}

func department(id, parentID uint) *rbacbiz.SysDepartment {
	item := &rbacbiz.SysDepartment{ParentID: parentID}
	item.ID = id
	return item
}

func equalUintSlices(left, right []uint) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
