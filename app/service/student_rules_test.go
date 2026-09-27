package service

import (
	"testing"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
)

func TestValidateCreate(t *testing.T) {
	tests := []struct {
		name string
		req  model.CreateStudentReq
		want []string
	}{
		{
			name: "valid request",
			req:  model.CreateStudentReq{NIM: "123456789", Name: "John Doe", Grade: 3.50},
		},
		{
			name: "name is too short",
			req:  model.CreateStudentReq{NIM: "123456789", Name: "Jo", Grade: 3.50},
			want: []string{"name"},
		},
		{
			name: "nim length is invalid",
			req:  model.CreateStudentReq{NIM: "12345678", Name: "John Doe", Grade: 3.50},
			want: []string{"nim"},
		},
		{
			name: "grade is below minimum",
			req:  model.CreateStudentReq{NIM: "123456789", Name: "John Doe", Grade: -0.01},
			want: []string{"grade"},
		},
		{
			name: "grade is above maximum",
			req:  model.CreateStudentReq{NIM: "123456789", Name: "John Doe", Grade: 4.01},
			want: []string{"grade"},
		},
	}

	// App Validator
	appValidator := helper.NewValidator(&helper.PasswordCommonSet{})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertErrorKeys(t, helper.ValidateStruct(test.req, *appValidator), test.want)
		})
	}
}

func TestValidateReplace(t *testing.T) {
	tests := []struct {
		name string
		req  model.ReplaceStudentReq
		want []string
	}{
		{
			name: "valid request",
			req:  model.ReplaceStudentReq{Name: "Jane Doe", Grade: 4.00, IsActive: true},
		},
		{
			name: "name is too short",
			req:  model.ReplaceStudentReq{Name: "Jo", Grade: 3.50},
			want: []string{"name"},
		},
		{
			name: "grade is outside range",
			req:  model.ReplaceStudentReq{Name: "Jane Doe", Grade: -1.00},
			want: []string{"grade"},
		},
	}

	// App Validator
	appValidator := helper.NewValidator(&helper.PasswordCommonSet{})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertErrorKeys(t, helper.ValidateStruct(test.req, *appValidator), test.want)
		})
	}
}

func TestValidatePatch(t *testing.T) {
	name := "  Jane Doe  "
	grade := 3.75
	inactive := false
	invalidName := "Jo"
	invalidGrade := 4.01

	tests := []struct {
		name       string
		req        model.PatchStudentReq
		want       model.Student
		wantErrors []string
	}{
		{
			name: "updates every supplied field and trims name",
			req: model.PatchStudentReq{
				Name: &name, Grade: &grade, IsActive: &inactive,
			},
			want: model.Student{ID: 1, NIM: "123456789", Name: "Jane Doe", Grade: 3.75, IsActive: false},
		},
		{
			name:       "invalid name keeps current value",
			req:        model.PatchStudentReq{Name: &invalidName},
			want:       model.Student{ID: 1, NIM: "123456789", Name: "John Doe", Grade: 3.50, IsActive: true},
			wantErrors: []string{"name"},
		},
		{
			name:       "invalid grade keeps current value",
			req:        model.PatchStudentReq{Grade: &invalidGrade},
			want:       model.Student{ID: 1, NIM: "123456789", Name: "John Doe", Grade: 3.50, IsActive: true},
			wantErrors: []string{"grade"},
		},
		{
			name: "empty patch preserves current value",
			req:  model.PatchStudentReq{},
			want: model.Student{ID: 1, NIM: "123456789", Name: "John Doe", Grade: 3.50, IsActive: true},
		},
	}

	// App Validator
	appValidator := helper.NewValidator(&helper.PasswordCommonSet{})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := model.Student{ID: 1, NIM: "123456789", Name: "John Doe", Grade: 3.50, IsActive: true}
			errs := helper.ValidateStruct(test.req, *appValidator)
			assertErrorKeys(t, errs, test.wantErrors)

			if len(errs) == 0 {
				got := ApplyPatchStudent(current, test.req)
				if got != test.want {
					t.Errorf("student mismatch: got %+v, want %+v", got, test.want)
				}
			}
		})
	}
}

func TestIsEmptyPatch(t *testing.T) {
	grade := 3.50

	if !IsEmptyPatchStudent(model.PatchStudentReq{}) {
		t.Error("empty patch should be detected")
	}
	if IsEmptyPatchStudent(model.PatchStudentReq{Grade: &grade}) {
		t.Error("patch with a field should not be detected as empty")
	}
}

func TestCountTotalPages(t *testing.T) {
	tests := []struct {
		total int
		limit int
		want  int
	}{
		{total: 0, limit: 10, want: 0},
		{total: 1, limit: 10, want: 1},
		{total: 10, limit: 10, want: 1},
		{total: 11, limit: 10, want: 2},
		{total: 137, limit: 20, want: 7},
		{total: 10, limit: 0, want: 0},
	}

	for _, test := range tests {
		if got := CountTotalPages(test.total, test.limit); got != test.want {
			t.Errorf("total-%d limit-%d: got %d, want %d", test.total, test.limit, got, test.want)
		}
	}
}

func assertErrorKeys(t *testing.T, errs map[string]string, want []string) {
	t.Helper()

	if len(errs) != len(want) {
		t.Fatalf("error count: got %d (%v), want %d (%v)", len(errs), errs, len(want), want)
	}

	for _, key := range want {
		if _, ok := errs[key]; !ok {
			t.Errorf("expected validation error for %q, got %v", key, errs)
		}
	}
}
