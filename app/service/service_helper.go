package service

import (
	"errors"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
)

func translateErr(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		// return helper.Fail(c, fiber.StatusNotFound, "data can't be found")
		return helper.NotFound(entity + " can't be found")
	case errors.Is(err, repository.ErrDuplicate):
		// return helper.Fail(c, fiber.StatusConflict, "data already used")
		return helper.Conflict("data already used")
	default:
		// return helper.Fail(c, fiber.StatusInternalServerError, generalMessage)
		return nil
	}
}
