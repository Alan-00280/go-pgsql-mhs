package helper

import (
	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/gofiber/fiber/v2"
)

const LocalsAuthUser = "authUser"

func CurentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(LocalsAuthUser).(model.AuthUser)
	return user, ok
}

func RequestID(c *fiber.Ctx) string {
	reqID, _ := c.Locals("requestid").(string)
	return reqID
}
