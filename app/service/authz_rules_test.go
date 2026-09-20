package service

import (
	"testing"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
)

func TestCanAccessStudent(t *testing.T) {
	perms := helper.NewPermissionSet(map[string][]string{
		"staff": {"student:read:any", "student:update:any"},
		"admin": {"student:delete"},
	})

	t.Run("owner can access own student data", func(t *testing.T) {
		current := model.AuthUser{UserID: 7, Role: "staff"}
		if !CanAccessStudent(current, 7, perms, "student:read:any") {
			t.Fatal("owner should be allowed to access their own student data")
		}
	})

	t.Run("role with required permission can access", func(t *testing.T) {
		current := model.AuthUser{UserID: 11, Role: "staff"}
		if !CanAccessStudent(current, 9, perms, "student:update:any") {
			t.Fatal("user with required permission should be allowed")
		}
	})

	t.Run("user without permission is denied", func(t *testing.T) {
		current := model.AuthUser{UserID: 11, Role: "staff"}
		if CanAccessStudent(current, 9, perms, "student:view-all") {
			t.Fatal("user without matching permission should be denied")
		}
	})
}

func TestCanAccessUser(t *testing.T) {
	perms := helper.NewPermissionSet(map[string][]string{
		"admin": {"user:read:any"},
		"staff": {"user:read"},
	})

	t.Run("owner can access own user data", func(t *testing.T) {
		current := model.AuthUser{UserID: 5, Role: "staff"}
		if !CanAccessUser(current, 5, perms, "user:read:any") {
			t.Fatal("owner should always be allowed to access their own data")
		}
	})

	t.Run("user with permission can access another user", func(t *testing.T) {
		current := model.AuthUser{UserID: 99, Role: "admin"}
		if !CanAccessUser(current, 5, perms, "user:read:any") {
			t.Fatal("admin with any-permission should be allowed")
		}
	})

	t.Run("user without permission is denied", func(t *testing.T) {
		current := model.AuthUser{UserID: 88, Role: "staff"}
		if CanAccessUser(current, 5, perms, "user:read:any") {
			t.Fatal("user without matching permission should be denied")
		}
	})
}

func TestValidateAssignRole(t *testing.T) {
	perms := helper.NewPermissionSet(map[string][]string{
		"admin":   {"role:assign"},
		"staff":   {"user:read"},
		"student": {"student:read:any"},
	})

	t.Run("valid role passes", func(t *testing.T) {
		current := model.AuthUser{UserID: 1, Role: "admin"}
		err := ValidateAssignRole(current, 2, model.AssignRoleRequest{Role: "staff"}, perms)
		if len(err) != 0 {
			t.Fatalf("expected no validation errors, got %v", err)
		}
	})

	t.Run("blank role is rejected", func(t *testing.T) {
		current := model.AuthUser{UserID: 1, Role: "admin"}
		err := ValidateAssignRole(current, 2, model.AssignRoleRequest{Role: "   "}, perms)
		if got := err["role"]; got != "role wajib diisi" {
			t.Fatalf("blank role validation = %q, want %q", got, "role wajib diisi")
		}
	})

	t.Run("unknown role is rejected", func(t *testing.T) {
		current := model.AuthUser{UserID: 1, Role: "admin"}
		err := ValidateAssignRole(current, 2, model.AssignRoleRequest{Role: "guest"}, perms)
		if got := err["role"]; got == "" || got[:len("guest")] != "guest" {
			t.Fatalf("expected unknown-role error, got %v", err)
		}
	})

	t.Run("user cannot change own role", func(t *testing.T) {
		current := model.AuthUser{UserID: 5, Role: "admin"}
		err := ValidateAssignRole(current, 5, model.AssignRoleRequest{Role: "staff"}, perms)
		if got := err["role"]; got != "user tidak dapat mengubah role miliknya sendiri" {
			t.Fatalf("own-role validation = %q, want %q", got, "user tidak dapat mengubah role miliknya sendiri")
		}
	})
}
