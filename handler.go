package main

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/gofiber/fiber/v2"
)

var students []model.Student
var nextID = 1

func findUserIdx(id int) int {
	for i := range students {
		if students[i].ID == id {
			return i
		}
	}
	return -1
}

func findMatch(s model.Student, key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(strings.ToLower(s.Name), key) ||
		strings.Contains(strings.ToLower(s.NIM), key)
}

func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))

	if err != nil || id < 1 {
		return 0, false
	}

	return id, true
}

// GET - List Students with filter and search
func listStudents(c *fiber.Ctx) error {
	q := parseListQuery(c)
	result := []model.Student{}

	// filter & search
	for _, s := range students {

		if q.IsActive != nil && s.IsActive != *q.IsActive {
			continue
		}
		if q.Search != "" && !findMatch(s, q.Search) {
			continue
		}
		if q.GradeFilter != nil && !(s.Grade >= q.StartGrade) && !(s.Grade <= q.EndGrade) {
			continue
		}

		result = append(result, s)
	}

	// sort
	sort.SliceStable(result, func(i, j int) bool {
		var less bool

		switch q.Sort {
		case "nim":
			less = result[i].NIM < result[j].NIM
		case "name":
			less = result[i].Name < result[j].Name
		case "grade":
			less = result[i].Grade < result[j].Grade
		default:
			less = result[i].ID < result[j].ID
		}

		if q.Order == "desc" {
			less = !less
		}

		return less
	})

	// paginate
	total := len(result)
	totalPage := (total + q.Limit - 1) / q.Limit
	start := (q.Page - 1) * q.Limit

	if start > total {
		start = total
	}

	end := start + q.Limit
	if end > total {
		end = total
	}

	return okList(c, "daftar Mahasiswa berhasil diambil", result[start:end], &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: totalPage,
	})
}

// GET - Find a Student by ID
func getStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id invalid")
	}

	i := findUserIdx(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student not found")
	}

	return ok(c, "Mahasiswa berhasil ditemukan", students[i])
}

// POST - Create a Student
func createStudent(c *fiber.Ctx) error {
	// PARSE
	var req model.CreateStudentReq
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "Data JSON Invalid")
	}

	errs := map[string]string{}
	req.Name = strings.TrimSpace(req.Name)
	req.NIM = strings.TrimSpace(req.NIM)

	// VALIDATE
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

	// CREATE
	newStudent := model.Student{
		ID:       nextID,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: true,
	}
	nextID++

	students = append(students, newStudent)

	// RETURN
	return created(c, "Mahasiswa berhasil dibuat", newStudent, "/api/v1/users/"+strconv.Itoa(newStudent.ID))
}

// PUT - Replace a Student Data
func replaceStudent(c *fiber.Ctx) error {
	// PARSE
	var req model.ReplaceStudentReq
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "Data JSON Invalid")
	}

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id invalid")
	}

	i := findUserIdx(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student not found")
	}

	req.Name = strings.TrimSpace(req.Name)

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

	// CHANGE
	student := &students[i]

	student.Name = req.Name
	student.Grade = req.Grade
	student.IsActive = req.IsActive

	// RETURN
	return ok(c, "Data mahasiswa diganti seluruhnya", student)
}

// PATCH - Update a Student Data
func patchStudent(c *fiber.Ctx) error {
	// PARSE
	var req model.PatchStudentReq
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "Data JSON Invalid")
	}

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id invalid")
	}

	i := findUserIdx(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student not found")
	}

	// VALIDATE
	if req.Name == nil && req.Grade == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "no data changed")
	}

	// IF VALIDATE CHANGE
	student := &students[i]

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

	// RETURN
	return ok(c, "Data mahasiswa berhasil diganti", student)
}

// DELETE - Delete a  student
func deleteStudent(c *fiber.Ctx) error {
	// VALIDATE
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id invalid")
	}

	i := findUserIdx(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student not found")
	}

	// DELETE
	students = slices.Delete(students, i, i+1)

	return noContent(c)
}
