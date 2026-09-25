package service

import (
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
)

// func ValidateCreateUser(req model.CreateUserRequest) map[string]string {
// 	errs := map[string]string{}

// 	if len(req.Username) < 3 {
// 		errs["username"] = "nama harus lebih dari 3 karakter"
// 	}

// 	parts := strings.Split(req.Email, "@")
// 	domain := parts[1]

// 	if len(parts) != 2 || !strings.Contains(domain, ".") {
// 		errs["email"] = "email tidak valid"
// 	}

// 	if len(req.Password) < 8 {
// 		errs["password"] = "password minimal 8 karakter"
// 	}

// 	return errs
// }

// func ValidateReplaceUser(req model.ReplaceUserRequest) map[string]string {
// 	errs := map[string]string{}

// 	if len(req.Username) < 3 {
// 		errs["username"] = "nama harus lebih dari 3 karakter"
// 	}

// 	parts := strings.Split(req.Email, "@")
// 	domain := parts[1]
// 	if len(parts) != 2 || !strings.Contains(domain, ".") {
// 		errs["email"] = "email tidak valid"
// 	}

// 	return errs
// }

func ValidatePatchUser(current model.User, req model.PatchUserRequest) (model.User, map[string]string) {
	errs := map[string]string{}

	if req.Username != nil {
		current.Username = strings.TrimSpace(*req.Username)
	}

	if req.Email != nil {
		current.Email = *req.Email
	}

	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current, errs
}

func IsEmptyPatchUser(req model.PatchUserRequest) bool {
	return req.Username == nil && req.Email == nil && req.IsActive == nil
}
