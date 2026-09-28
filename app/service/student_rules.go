package service

import (
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
)

func ApplyPatchStudent(current model.Student, req model.PatchStudentReq) model.Student {
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
	}

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
