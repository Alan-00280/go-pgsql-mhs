package service

import (
	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
)

func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	studentID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if current.UserID == ownerID {
		return true
	}

	if current.UserID == studentID && current.Role == "student" {
		return true
	}

	return perms.Can(current.Role, anyPermission)
}
