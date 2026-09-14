package service

import (
	"strconv"
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	repo repository.UserRepository
}

func NewUserHandler(repo repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) ListAll() error {
	return nil
}

func (h *UserHandler) Get() error {
	return nil
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "json is not valid")
	}

	req.Username = strings.TrimSpace(req.Username)

	if errs := ValidateCreateUser(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	hashedPassword, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "can't create user")
	}

	new, err := h.repo.Create(ctx, model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
	})
	if err != nil {
		return translateErr(c, err, "can't create user")
	}

	return helper.Created(c, "user successfully created!", new, "/api/v1/users/"+strconv.Itoa(new.ID))
}

func (h *UserHandler) Replace() error {
	return nil
}

func (h *UserHandler) Update() error {
	return nil
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "can't delete user: ID Invalid")
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return translateErr(c, err, "can't delete user")
	}

	return helper.NoContent(c)
}
