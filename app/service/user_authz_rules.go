package service

import (
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
)

// TODO file ini mendefinisikan proses pengecekan user berhak apa untuk data nya milik diri sendiri

// func CanAccessUser(model.AuthUser, targetID int, *helper.PermissionSet, anyPermission string) bool
// memeriksa id user terautentikasi dengan target id (/:id)
// diijinkan ke target id sama
// diijinkan ke permission *:any
func CanAccessUser(
	current model.AuthUser,
	targetID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if current.UserID == targetID {
		return true
	}

	return perms.Can(current.Role, anyPermission)
}

// func ValidateAssignRole(model.AuthUser, targetID int, model.AssignRoleRequest, *helper.PermissionSet) map[string]string
// memeriksa current id dengan target id
//
//	jika sama ditolak (mengembalikan map error)
//
// memeriksa validitas nama role
// trim space role
func ValidateAssignRole(
	current model.AuthUser,
	targetID int,
	req model.AssignRoleRequest,
	perms *helper.PermissionSet,
) map[string]string {
	errs := map[string]string{}

	// req.Role = strings.TrimSpace(req.Role)
	// if req.Role == "" {
	// 	errs["role"] = "role wajib diisi"
	// 	return errs
	// }

	if !perms.IsKnownRoles(req.Role) {
		errs["role"] = req.Role + " tidak termasuk dalam role valid: " + strings.Join(perms.KnownRoles(), ", ")
	}

	if current.UserID == targetID {
		errs["role"] = "user tidak dapat mengubah role miliknya sendiri"
	}

	return errs
}
