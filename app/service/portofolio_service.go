package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/gofiber/fiber/v2"
)

type PortofolioService struct {
	portofolios  repository.PortofolioRepository
	students     repository.StudentRepository
	perms        *helper.PermissionSet
	appValidator *helper.AppValidator
}

func NewPortofolioService(
	portofolios repository.PortofolioRepository,
	students repository.StudentRepository,
	perms *helper.PermissionSet,
	appValidator *helper.AppValidator,
) *PortofolioService {
	return &PortofolioService{
		portofolios:  portofolios,
		students:     students,
		perms:        perms,
		appValidator: appValidator,
	}
}

// GET /students/:id/portfolio
func (s *PortofolioService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id must be a positive number")
	}

	if _, err := s.getStudent(ctx, studentID); err != nil {
		return err
	}

	portofolio, found, err := s.findByStudent(ctx, studentID)
	if err != nil {
		return translatePortofolioError(err)
	}
	if !found {
		return helper.NotFound("portofolio tidak ditemukan")
	}

	return helper.Success(c, "portofolio berhasil diambil", portofolio)
}

// PUT /students/:id/portfolio
func (s *PortofolioService) Put(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id must be a positive number")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}
	if _, err := s.getStudent(ctx, studentID); err != nil {
		return err
	}

	var req model.CreatePortofolioReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON Body invalid")
	}
	req.Headline = strings.TrimSpace(req.Headline)
	req.Bio = strings.TrimSpace(req.Bio)
	for i := range req.Certificates {
		req.Certificates[i].Title = strings.TrimSpace(req.Certificates[i].Title)
		req.Certificates[i].Publisher = strings.TrimSpace(req.Certificates[i].Publisher)
	}
	if errs := helper.ValidateStruct(req, *s.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}

	portofolio, found, err := s.findByStudent(ctx, studentID)
	if err != nil {
		return translatePortofolioError(err)
	}

	if !found {
		created, err := s.portofolios.Create(ctx, model.Portofolio{
			UserID:       current.UserID,
			StudentID:    studentID,
			Headline:     req.Headline,
			Bio:          req.Bio,
			Certificates: certificatesToModel(req.Certificates),
		})
		if err != nil {
			return translatePortofolioError(err)
		}
		return helper.Created(c, "portofolio berhasil dibuat", created, "/api/v1/students/"+c.Params("id")+"/portfolio")
	}

	updated, err := s.portofolios.Update(ctx, portofolio.ID, current.UserID,
		s.perms.Can(current.Role, "student:update:any"), model.UpdatePortofolioReq{
			Headline:     req.Headline,
			Bio:          req.Bio,
			Certificates: req.Certificates,
		})
	if err != nil {
		return translatePortofolioError(err)
	}
	return helper.Success(c, "portofolio berhasil diperbaharui", updated)
}

// POST /students/:id/portfolio/projects
func (s *PortofolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id must be a positive number")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}
	if _, err := s.getStudent(ctx, studentID); err != nil {
		return err
	}

	var body model.AddProjectsReq
	if err := c.BodyParser(&body); err != nil {
		return helper.BadRequest("JSON Body invalid")
	}
	req := model.CreateProjectReq{
		Title:        body.Project.Title,
		Technologies: body.Project.Technologies,
		Link:         body.Project.Link,
		Year:         body.Project.Year,
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Link = strings.TrimSpace(req.Link)
	for i := range req.Technologies {
		req.Technologies[i] = strings.TrimSpace(req.Technologies[i])
	}
	if errs := helper.ValidateStruct(req, *s.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}

	portofolio, found, err := s.findByStudent(ctx, studentID)
	if err != nil {
		return translatePortofolioError(err)
	}
	if !found {
		return helper.NotFound("portofolio tidak ditemukan")
	}

	updated, err := s.portofolios.AddProject(
		ctx,
		portofolio.ID,
		current.UserID,
		s.perms.Can(current.Role, "student:update:any"),
		model.Project{
			Title:        req.Title,
			Technologies: req.Technologies,
			Link:         req.Link,
			Year:         req.Year,
		},
	)
	if err != nil {
		return translatePortofolioError(err)
	}

	project := updated.Projects[len(updated.Projects)-1]
	location := "/api/v1/students/" + c.Params("id") + "/portfolio/projects/" + project.ID
	return helper.Created(c, "proyek berhasil ditambahkan", project, location)
}

// DELETE /students/:id/portfolio/projects/:projectId
func (s *PortofolioService) DeleteProject(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id must be a positive number")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}
	if _, err := s.getStudent(ctx, studentID); err != nil {
		return err
	}

	portofolio, found, err := s.findByStudent(ctx, studentID)
	if err != nil {
		return translatePortofolioError(err)
	}
	if !found {
		return helper.NotFound("portofolio tidak ditemukan")
	}

	if err := s.portofolios.DeleteProject(
		ctx,
		portofolio.ID,
		c.Params("projectId"),
		current.UserID,
		s.perms.Can(current.Role, "student:update:any"),
	); err != nil {
		return translatePortofolioError(err)
	}

	return helper.NoContent(c)
}

// GET /portfolios/stats/technologies
func (s *PortofolioService) TechSummary(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	summary, err := s.portofolios.TechSummary(ctx)
	if err != nil {
		return translatePortofolioError(err)
	}
	return helper.Success(c, "ringkasan teknologi berhasil diambil", summary)
}

func (s *PortofolioService) getStudent(
	ctx context.Context,
	studentID int,
) (model.Student, error) {
	student, err := s.students.FindById(ctx, studentID)
	if err != nil {
		return model.Student{}, translateErr(err, "student")
	}
	return student, nil
}

func (s *PortofolioService) findByStudent(
	ctx context.Context,
	studentID int,
) (model.Portofolio, bool, error) {
	portofolios, err := s.portofolios.Get(ctx, model.PortofolioQuery{
		StudentID: &studentID,
		Limit:     1,
	})
	if err != nil {
		return model.Portofolio{}, false, err
	}
	if len(portofolios) == 0 {
		return model.Portofolio{}, false, nil
	}
	return portofolios[0], true, nil
}

func certificatesToModel(certificates []model.CreateCertificateReq) []model.Certificate {
	result := make([]model.Certificate, 0, len(certificates))
	for _, certificate := range certificates {
		result = append(result, model.Certificate{
			Title:     certificate.Title,
			Publisher: certificate.Publisher,
			Date:      certificate.Date,
		})
	}
	return result
}

func translatePortofolioError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id portofolio atau proyek tidak valid")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("data portofolio tidak ditemukan")
	case errors.Is(err, repository.ErrNotOwner):
		return helper.Forbidden("hanya pemilik portofolio yang boleh melakukan ini")
	case errors.Is(err, repository.ErrDuplicate), errors.Is(err, repository.ErrAlreadySubmit):
		return helper.Conflict("portofolio mahasiswa sudah terdaftar")
	case errors.Is(err, repository.ErrProjectsLimit):
		return helper.NewError(fiber.StatusConflict, "PROJECT_LIMIT", "jumlah proyek sudah mencapai batas")
	default:
		return helper.Internal(err)
	}
}
