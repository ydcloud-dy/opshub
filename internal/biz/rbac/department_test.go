package rbac

import (
	"context"
	"errors"
	"testing"
)

type departmentRepoStub struct {
	departments []*SysDepartment
	created     *SysDepartment
	updated     *SysDepartment
}

func (repo *departmentRepoStub) Create(_ context.Context, dept *SysDepartment) error {
	repo.created = dept
	return nil
}

func (repo *departmentRepoStub) Update(_ context.Context, dept *SysDepartment) error {
	repo.updated = dept
	return nil
}

func (repo *departmentRepoStub) Delete(_ context.Context, _ uint) error {
	return nil
}

func (repo *departmentRepoStub) GetByID(_ context.Context, id uint) (*SysDepartment, error) {
	for _, dept := range repo.departments {
		if dept.ID == id {
			return dept, nil
		}
	}
	return nil, errors.New("department not found")
}

func (repo *departmentRepoStub) GetTree(_ context.Context) ([]*SysDepartment, error) {
	return repo.departments, nil
}

func (repo *departmentRepoStub) GetAll(_ context.Context) ([]*SysDepartment, error) {
	return repo.departments, nil
}

func TestDepartmentUseCaseCreateValidatesHierarchy(t *testing.T) {
	company := &SysDepartment{DeptType: 1}
	company.ID = 1
	center := &SysDepartment{ParentID: 1, DeptType: 2}
	center.ID = 2
	department := &SysDepartment{ParentID: 2, DeptType: 3}
	department.ID = 3

	tests := []struct {
		name    string
		dept    *SysDepartment
		wantErr bool
	}{
		{name: "company without parent", dept: &SysDepartment{DeptType: 1}},
		{name: "center under company", dept: &SysDepartment{ParentID: 1, DeptType: 2}},
		{name: "center without parent", dept: &SysDepartment{DeptType: 2}, wantErr: true},
		{name: "department under company", dept: &SysDepartment{ParentID: 1, DeptType: 3}},
		{name: "department under center", dept: &SysDepartment{ParentID: 2, DeptType: 3}},
		{name: "department under department", dept: &SysDepartment{ParentID: 3, DeptType: 3}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &departmentRepoStub{departments: []*SysDepartment{company, center, department}}
			useCase := NewDepartmentUseCase(repo)
			err := useCase.Create(context.Background(), test.dept)
			if test.wantErr && err == nil {
				t.Fatal("expected hierarchy validation error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDepartmentUseCaseUpdateRejectsCycle(t *testing.T) {
	company := &SysDepartment{DeptType: 1}
	company.ID = 1
	department := &SysDepartment{ParentID: 1, DeptType: 3}
	department.ID = 3
	legacyCenter := &SysDepartment{ParentID: 3, DeptType: 2}
	legacyCenter.ID = 4

	repo := &departmentRepoStub{departments: []*SysDepartment{company, department, legacyCenter}}
	useCase := NewDepartmentUseCase(repo)
	updatedDepartment := &SysDepartment{ParentID: 4, DeptType: 3}
	updatedDepartment.ID = 3
	err := useCase.Update(context.Background(), updatedDepartment)
	if err == nil {
		t.Fatal("expected cycle validation error")
	}
}

func TestDepartmentParentOptionsIncludeType(t *testing.T) {
	company := &SysDepartment{Name: "Company", DeptType: 1}
	company.ID = 1
	center := &SysDepartment{Name: "Center", ParentID: 1, DeptType: 2}
	center.ID = 2

	repo := &departmentRepoStub{departments: []*SysDepartment{company, center}}
	useCase := NewDepartmentUseCase(repo)
	options, err := useCase.GetParentOptions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(options) != 1 || options[0].DeptType != 1 {
		t.Fatalf("unexpected company option: %#v", options)
	}
	if len(options[0].Children) != 1 || options[0].Children[0].DeptType != 2 {
		t.Fatalf("unexpected center option: %#v", options[0].Children)
	}
}
