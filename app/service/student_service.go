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
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentHandler(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentHandler {
	return &StudentHandler{repo: repo, perms: perms}
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

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusInternalServerError, "can't verifying your identity")
	}

	student, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(c, err, "can't get student data")
	}

	if !CanAccessStudent(current, student.OwnerID, h.perms, "student:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "anda tidak dapat hak untuk mengakses student ini")
	}

	return helper.Ok(c, "student found", student)
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
	if errs := ValidateCreateStudent(req); len(errs) > 0 {
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
		"/api/v1/students/"+strconv.Itoa(baru.ID))
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

	student, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(c, err, "can't get student data")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusInternalServerError, "can't verify your identity")
	}

	if !CanAccessStudent(current, student.OwnerID, h.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "anda tidak dapat hak untuk mengganti data student ini")
	}

	// VALIDATE
	if errs := ValidateReplaceStudent(req); len(errs) > 0 {
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

	if IsEmptyPatchStudent(req) {
		return helper.Fail(c, fiber.StatusBadRequest, "no data changed")
	}

	student, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(c, err, "gagal mengambil data student")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusInternalServerError, "can't verify your identity")
	}

	if !CanAccessStudent(current, student.OwnerID, h.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "anda tidak dapat hak untuk memperbarui data student ini")
	}

	newStudent, errs := ValidatePatchStudent(student, req)
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
		return translateErr(c, err, "gagal menghapus student")
	}

	return helper.NoContent(c)
}
