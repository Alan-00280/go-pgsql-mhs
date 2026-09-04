package service

import (
	"strconv"
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/gofiber/fiber/v2"
)

type StudentHandler struct {
	repo repository.StudentRepository
}

func NewStudentHandler(repo repository.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

// GET - Get All Students
func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	q := helper.ParseListQuery(c)

	students, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "fail to get student list")
	}

	return helper.OkList(c, "student list successfully retreived", students, &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: CountTotalPages(total, q.Limit),
	})
}

// GET - Get a Student by ID
func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id invalid")
	}

	user, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(c, err, "can't get student data")
	}

	return helper.Ok(c, "student found", user)
}

// POST - Create a Student
func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.CreateStudentReq
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.Name = strings.TrimSpace(req.Name)

	// VALIDATION
	if errs := ValidateCreate(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	// Keunikan username TIDAK diperiksa dengan SELECT lebih dulu.
	// Basis data sudah menjaminnya lewat UNIQUE INDEX, dan pemeriksaan
	// manual justru menyisakan celah bila dua permintaan datang bersamaan.
	baru, err := h.repo.Create(ctx, model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: true,
	})
	if err != nil {
		return translateErr(c, err, "can't store student")
	}

	return helper.Created(c, "user berhasil dibuat", baru,
		"/api/v1/users/"+strconv.Itoa(baru.ID))
}

// PUT - Replace an entire student data
func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id must be a positive number")
	}

	var req model.ReplaceStudentReq
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "JSON Body invalid")
	}

	// VALIDATE
	if errs := ValidateReplace(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	// UPDATE
	hasil, err := h.repo.Update(ctx, model.Student{
		ID: id, Name: req.Name, Grade: req.Grade, IsActive: req.IsActive,
	})
	if err != nil {
		return translateErr(c, err, "can't update student")
	}

	return helper.Ok(c, "student successfully changed entirely", hasil)
}

// PATCH - Update a Student Data
func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.PatchStudentReq
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if IsEmptyPatch(req) {
		return helper.Fail(c, fiber.StatusBadRequest, "no data changed")
	}

	student, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(c, err, "gagal mengambil data user")
	}

	newStudent, errs := ValidatePatch(student, req)
	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	result, err := h.repo.Update(ctx, newStudent)
	if err != nil {
		return translateErr(c, err, "gagal memperbarui user")
	}

	return helper.Ok(c, "user berhasil diperbarui sebagian", result)
}

// DELETE - Drop a Student
func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return translateErr(c, err, "gagal menghapus user")
	}

	return helper.NoContent(c)
}
