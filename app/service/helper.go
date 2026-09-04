package service

func translateErr(c *fiber.Ctx, err error, generalMessage string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return fail(c, fiber.StatusNotFound, "student can't be found")
	case errors.Is(err, repository.ErrDuplicate):
		return fail(c, fiber.StatusConflict, "nim already used")
	default:
		return fail(c, fiber.StatusInternalServerError, generalMessage)
	}
}
