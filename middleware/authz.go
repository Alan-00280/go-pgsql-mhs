package middleware

import (
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/gofiber/fiber/v2"
)

// middleware RequirePermission(*helper.PermissionSet, permission string)
// melakukan pengecekan dari user.role dan permission yang dibutuhkan
// dipasang di endpoint tertentu
// user diambil dari context (dicek ada atau enggak terlebih dahulu)
func RequirePermission(perms *helper.PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurentUser(c)
		if !ok {
			return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
		}

		if !perms.Can(user.Role, permission) {
			return helper.Fail(c, fiber.StatusForbidden, "user dengan role "+user.Role+" tidak memiliki hak "+permission)
		}

		return c.Next()
	}
}

// middleware RequireRole(roles ...string)
// metode lain untuk melindungi
// yaitu menambah beberapa role sebagai syarat ke endpoint
func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{})
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		user, ok := helper.CurentUser(c)
		if !ok {
			return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
		}

		_, granted := allowed[user.Role]
		if !granted {
			return helper.Fail(c, fiber.StatusForbidden, "role "+user.Role+" tidak memiliki akses ini")
		}

		return c.Next()
	}
}
