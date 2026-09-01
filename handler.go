package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/gofiber/fiber/v2"
)

var students []model.Student
var nextID = 1

type StudentHandler struct {
	repo repository.StudentRepository
}

func NewStudentHandler(repo repository.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

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

func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))

	if err != nil || id < 1 {
		return 0, false
	}

	return id, true
}

// GET - Get All Students
func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	q := parseListQuery(c)

	students, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "fail to get student list")
	}

	totalPages := 0
	if q.Limit > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}

	return okList(c, "student list successfully retreived", students, &model.Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
}

// GET - Get a Student by ID
func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id invalid")
	}

	user, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(c, err, "can't get student data")
	}

	return ok(c, "student found", user)
}

// POST - Create a Student
func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	var req model.CreateStudentReq
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.Name = strings.TrimSpace(req.Name)

	errs := map[string]string{}
	if len(req.Name) < 3 {
		errs["name"] = "Nama harus lebih dari 3 karakter"
	}
	if len(req.NIM) != model.NIM_LENGTH {
		errs["nim"] = "NIM harus memiliki panjang 9 karakter"
	}
	for _, s := range students {
		if req.NIM == s.NIM {
			return fail(c, fiber.StatusConflict, "NIM telah terdaftar")
		}
	}
	if req.Grade > model.MAX_GRADE || req.Grade < 0.00 {
		errs["grade"] = "Nilai melebihi rentang 0.00 - 4.00"
	}

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	// Keunikan username TIDAK diperiksa dengan SELECT lebih dulu.
	// Basis data sudah menjaminnya lewat UNIQUE INDEX, dan pemeriksaan
	// manual justru menyisakan celah bila dua permintaan datang bersamaan.
	baru, err := h.repo.Create(ctx, model.Student{
		ID:       nextID,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: true,
	})
	if err != nil {
		return translateErr(c, err, "can't store student")
	}

	return created(c, "user berhasil dibuat", baru,
		"/api/v1/users/"+strconv.Itoa(baru.ID))
}

// PUT - Replace an entire student data
func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id must be a positive number")
	}

	var req model.ReplaceStudentReq
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "JSON Body invalid")
	}

	// VALIDATE
	errs := map[string]string{}
	if len(req.Name) < 3 {
		errs["name"] = "Nama harus lebih dari 3 karakter"
	}
	if req.Grade > model.MAX_GRADE || req.Grade < 0.00 {
		errs["grade"] = "Nilai melebihi rentang 0.00 - 4.00"
	}

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	// UPDATE
	hasil, err := h.repo.Update(ctx, model.Student{
		ID: id, Name: req.Name, Grade: req.Grade, IsActive: req.IsActive,
	})
	if err != nil {
		return translateErr(c, err, "can't update student")
	}

	return ok(c, "student successfully changed entirely", hasil)
}

// PATCH - Update a Student Data
func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.PatchStudentReq
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if req.Name == nil && req.Grade == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "no data changed")
	}

	student, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(c, err, "gagal mengambil data user")
	}

	if req.Name != nil {
		errs := map[string]string{}

		*req.Name = strings.TrimSpace(*req.Name)
		if len(*req.Name) < 3 {
			errs["name"] = "Nama harus lebih dari 3 karakter"
		}

		if len(errs) > 0 {
			return failValidation(c, errs)
		}

		student.Name = *req.Name
	}

	if req.Grade != nil {
		errs := map[string]string{}

		if *req.Grade > model.MAX_GRADE || *req.Grade < 0.00 {
			errs["grade"] = "Nilai melebihi rentang 0.00 - 4.00"
		}

		if len(errs) > 0 {
			return failValidation(c, errs)
		}

		student.Grade = *req.Grade
	}

	if req.IsActive != nil {
		student.IsActive = *req.IsActive
	}

	result, err := h.repo.Update(ctx, student)
	if err != nil {
		return translateErr(c, err, "gagal memperbarui user")
	}

	return ok(c, "user berhasil diperbarui sebagian", result)
}

// DELETE - Drop a Student
func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return translateErr(c, err, "gagal menghapus user")
	}

	return noContent(c)
}
