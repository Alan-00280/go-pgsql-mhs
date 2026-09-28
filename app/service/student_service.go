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
	repo         repository.StudentRepository
	perms        *helper.PermissionSet
	appValidator *helper.AppValidator
}

func NewStudentHandler(repo repository.StudentRepository, perms *helper.PermissionSet, appValidator *helper.AppValidator) *StudentHandler {
	return &StudentHandler{repo: repo, perms: perms, appValidator: appValidator}
}

// GET - Get All Students
func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	q := helper.ParseListQuery(c)

	students, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.OkList(c, "student list successfully retreived", students, &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: CountTotalPages(total, q.Limit),
	})
}

// GET - Get All Students (using cursor)
func (h *StudentHandler) ListCursor(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	q := helper.ParseCursorQuery(c)

	rows, err := h.repo.FindAfterCursor(ctx, q)

	if err != nil {
		return helper.Internal(err)
	}

	// Baris tambahan hasil limit+1 dipotong di sini. Ia hanya penanda bahwa
	// masih ada halaman berikutnya, bukan bagian dari halaman ini.
	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar students berhasil diambil", rows, meta)
}

// GET - Get a Student by ID
func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id invalid")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	student, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(err, "student")
	}

	if !CanAccessStudent(current, student.OwnerID, student.ID, h.perms, "student:read:any") {
		return helper.Forbidden("anda tidak dapat hak untuk mengakses student ini")
	}

	return helper.Ok(c, "student found", student)
}

// POST - Create a Student
func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.CreateStudentReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.Name = strings.TrimSpace(req.Name)

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	// VALIDATION
	if errs := helper.ValidateStruct(req, *h.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}

	// Keunikan username TIDAK diperiksa dengan SELECT lebih dulu.
	// Basis data sudah menjaminnya lewat UNIQUE INDEX, dan pemeriksaan
	// manual justru menyisakan celah bila dua permintaan datang bersamaan.
	baru, err := h.repo.Create(ctx, model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: true,
		OwnerID:  current.UserID,
	})
	if err != nil {
		return translateErr(err, "student")
	}

	return helper.Created(c, "student berhasil dibuat", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

// PUT - Replace an entire student data
func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id must be a positive number")
	}

	var req model.ReplaceStudentReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON Body invalid")
	}

	student, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(err, "student")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	if !CanAccessStudent(current, student.OwnerID, student.ID, h.perms, "student:update:any") {
		return helper.Forbidden("anda tidak dapat hak untuk mengganti data student ini")
	}

	// VALIDATE
	if errs := helper.ValidateStruct(req, *h.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}

	// UPDATE
	hasil, err := h.repo.Update(ctx, model.Student{
		ID: id, NIM: student.NIM, Name: req.Name, Grade: req.Grade, IsActive: req.IsActive,
	})
	if err != nil {
		return translateErr(err, "student")
	}

	return helper.Ok(c, "student successfully changed entirely", hasil)
}

// PATCH - Update a Student Data
func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.PatchStudentReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if IsEmptyPatchStudent(req) {
		return helper.BadRequest("no data changed")
	}

	student, err := h.repo.FindById(ctx, id)
	if err != nil {
		return translateErr(err, "student")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	if !CanAccessStudent(current, student.OwnerID, student.ID, h.perms, "student:update:any") {
		return helper.Forbidden("anda tidak dapat hak untuk memperbarui data student ini")
	}

	errs := helper.ValidateStruct(req, *h.appValidator)
	if len(errs) > 0 {
		return helper.Validation(errs)
	}
	newStudent := ApplyPatchStudent(student, req)

	result, err := h.repo.Update(ctx, newStudent)
	if err != nil {
		return translateErr(err, "student")
	}

	return helper.Ok(c, "student berhasil diperbarui sebagian", result)
}

// DELETE - Drop a Student
func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return translateErr(err, "student")
	}

	return helper.NoContent(c)
}
