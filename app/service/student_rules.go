package service

import (
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
)

func ValidateCreate(req model.CreateStudentReq) map[string]string {
	errs := map[string]string{}

	if len(req.Name) < 3 {
		errs["name"] = "Nama harus lebih dari 3 karakter"
	}
	if len(req.NIM) != model.NIM_LENGTH {
		errs["nim"] = "NIM harus memiliki panjang 9 karakter"
	}
	if req.Grade > model.MAX_GRADE || req.Grade < 0.00 {
		errs["grade"] = "Nilai melebihi rentang 0.00 - 4.00"
	}

	return errs
}

func ValidateReplace(req model.ReplaceStudentReq) map[string]string {
	errs := map[string]string{}

	if len(req.Name) < 3 {
		errs["name"] = "Nama harus lebih dari 3 karakter"
	}
	if req.Grade > model.MAX_GRADE || req.Grade < 0.00 {
		errs["grade"] = "Nilai melebihi rentang 0.00 - 4.00"
	}

	return errs
}

func ValidatePatch(current model.Student, req model.PatchStudentReq) (model.Student, map[string]string) {
	errs := map[string]string{}

	if req.Name != nil {
		*req.Name = strings.TrimSpace(*req.Name)

		if len(*req.Name) < 3 {
			errs["name"] = "Nama harus lebih dari 3 karakter"
		} else {
			current.Name = *req.Name
		}
	}

	if req.Grade != nil {
		if *req.Grade > model.MAX_GRADE || *req.Grade < 0.00 {
			errs["grade"] = "Nilai melebihi rentang 0.00 - 4.00"
		} else {
			current.Grade = *req.Grade
		}
	}

	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current, errs
}

func IsEmptyPatch(req model.PatchStudentReq) bool {
	return req.IsActive == nil && req.Grade == nil && req.Name == nil
}

// make the total page even without decimal number
func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}

	return (total + limit - 1) / limit
}
