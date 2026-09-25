package service

import (
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
)

// func ValidateCreateStudent(req model.CreateStudentReq) map[string]string {
// 	errs := map[string]string{}

// 	req.Name = strings.TrimSpace(req.Name)
// 	req.NIM = strings.TrimSpace(req.NIM)

// 	if len(req.Name) < 3 {
// 		errs["name"] = "Nama minimal terdiri dari 3 karakter"
// 	}
// 	if len(req.NIM) != model.NIM_LENGTH {
// 		errs["nim"] = "NIM harus memiliki panjang 9 karakter"
// 	}
// 	if req.Grade > model.MAX_GRADE || req.Grade < 0.00 {
// 		errs["grade"] = "Nilai melebihi rentang 0.00 - 4.00"
// 	}

// 	return errs
// }

// func ValidateReplaceStudent(req model.ReplaceStudentReq) map[string]string {
// 	errs := map[string]string{}

// 	if len(req.Name) < 3 {
// 		errs["name"] = "Nama minimal terdiri dari 3 karakter"
// 	}
// 	if req.Grade > model.MAX_GRADE || req.Grade < 0.00 {
// 		errs["grade"] = "Nilai melebihi rentang 0.00 - 4.00"
// 	}

// 	return errs
// }

func ApplyPatchStudent(current model.Student, req model.PatchStudentReq) model.Student {
	// errs := map[string]string{}

	// if req.Name != nil {
	// 	*req.Name = strings.TrimSpace(*req.Name)

	// 	if len(*req.Name) < 3 {
	// 		errs["name"] = "Nama minimal 3 karakter"
	// 	} else {
	// 		current.Name = *req.Name
	// 	}
	// }
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
	}

	// if req.Grade != nil {
	// 	if *req.Grade > model.MAX_GRADE || *req.Grade < 0.00 {
	// 		errs["grade"] = "Nilai melebihi rentang 0.00 - 4.00"
	// 	} else {
	// 		current.Grade = *req.Grade
	// 	}
	// }
	if req.Grade != nil {
		current.Grade = *req.Grade
	}

	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current
}

func IsEmptyPatchStudent(req model.PatchStudentReq) bool {
	return req.IsActive == nil && req.Grade == nil && req.Name == nil
}
