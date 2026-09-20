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
	repo  repository.UserRepository
	perms *helper.PermissionSet
}

func NewUserHandler(repo repository.UserRepository, perms *helper.PermissionSet) *UserHandler {
	return &UserHandler{repo: repo, perms: perms}
}

func (h *UserHandler) ListAll(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	q := helper.ParseListQuery(c)

	users, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "tidak dapat mendapatkan seluruh user")
	}

	return helper.OkList(c, "berhasil mendapatkan semua user", users, &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		TotalPages: CountTotalPages(total, q.Limit),
		Total:      total,
	})
}

func (h *UserHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		helper.Fail(c, fiber.StatusBadRequest, "id invalid")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusInternalServerError, "can't verifying your identity")
	}

	if !CanAccessUser(current, id, h.perms, "user:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "anda tidak dapat hak untuk mengakses pengguna ini")
	}

	user, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return translateErr(c, err, "gagal memperoleh user")
	}

	return helper.Ok(c, "berhasil mendapatkan user", user)
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

func (h *UserHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id invalid")
	}

	var req model.ReplaceUserRequest
	if err := c.BodyParser(&req); err != nil {
		helper.Fail(c, fiber.StatusInternalServerError, "JSON invalid")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusInternalServerError, "can't verifying your identity")
	}

	if !CanAccessUser(current, id, h.perms, "user:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "anda tidak dapat hak untuk menggantikan data pengguna ini")
	}

	user, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
	}

	if errs := ValidateReplaceUser(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	updated, err := h.repo.Update(ctx, user)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui data pengguna sepenuhnya")
	}

	return helper.Ok(c, "berhasil memperbarui data pengguna secara penuh", updated)
}

func (h *UserHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id invalid")
	}

	var req model.PatchUserRequest
	if err := c.BodyParser(&req); err != nil {
		helper.Fail(c, fiber.StatusBadRequest, "JSON invalid")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusInternalServerError, "can't verifying your identity")
	}

	if !CanAccessUser(current, id, h.perms, "user:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "anda tidak dapat hak untuk memperbarui data pengguna ini")
	}

	user, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return translateErr(c, err, "gagal mendapatkan user")
	}

	updated, errs := ValidatePatchUser(user, req)
	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	updated_user, err := h.repo.Update(ctx, updated)
	if err != nil {
		return translateErr(c, err, "gagal memperbarui")
	}

	return helper.Ok(c, "berhasil memperbarui user", updated_user)
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
