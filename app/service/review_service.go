package service

import (
	"errors"
	"strings"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/gofiber/fiber/v2"
)

type ReviewService struct {
	reviews      repository.ReviewRepository
	courses      repository.CourseRepository
	perms        *helper.PermissionSet
	appValidator *helper.AppValidator
}

func NewReviewService(
	reviews repository.ReviewRepository,
	courses repository.CourseRepository,
	perms *helper.PermissionSet,
	appValidator *helper.AppValidator,
) *ReviewService {
	return &ReviewService{
		reviews:      reviews,
		courses:      courses,
		perms:        perms,
		appValidator: appValidator,
	}
}

// POST /courses/:id/reviews

// Create godoc
//
// @Summary      Menulis ulasan untuk sebuah kelas
// @Description  Hanya peserta kelas. Satu mahasiswa hanya boleh menulis satu ulasan per kelas.
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id    path      int                        true  "ID kelas"
// @Param        body  body      model.CreateReviewRequest  true  "Isi ulasan"
// @Success      201   {object}  model.WebResponse{data=model.Review}
// @Failure      403   {object}  model.ErrorRespone  "NOT_ENROLLED"
// @Failure      409   {object}  model.ErrorRespone  "ALREADY_REVIEWED"
// @Failure      422   {object}  model.ErrorRespone
// @Security     BearerAuth
// @Router       /courses/{id}/reviews [post]
func (s *ReviewService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id must be a positive number")
	}

	var req model.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON Body invalid")
	}

	req.Comment = strings.TrimSpace(req.Comment)
	req.Tags = NormalizeTags(req.Tags)

	if errs := helper.ValidateStruct(req, *s.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}
	if _, err := s.courses.FindByID(ctx, id); err != nil {
		return translateReviewError(err)
	}

	enrolled, err := s.courses.IsEnrolled(ctx, id, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}

	if !enrolled {
		return helper.NewError(fiber.StatusForbidden, "NOT_ENROLLED", "hanya peserta kelas yang boleh menulis ulasan")
	}

	created, err := s.reviews.Create(ctx, model.Review{
		CourseID: id,
		UserID:   current.UserID,
		Username: current.Username,
		Rating:   req.Rating,
		Comment:  req.Comment,
		Tags:     req.Tags,
	})
	if err != nil {
		return translateReviewError(err)
	}

	return helper.Created(c, "ulasan berhasil disimpan", created, "/api/v1/reviews/"+created.ID)
}

// GET /courses/:id/reviews

// ListByCourse godoc
//
// @Summary      Daftar ulasan sebuah kelas, terbaru lebih dulu
// @Description  Cursor pagination: kirim meta.next_before sebagai parameter before untuk halaman berikutnya.
// @Tags         reviews
// @Produce      json
// @Param        id      path      int     true   "ID kelas"
// @Param        tag     query     string  false  "Saring berdasarkan tag"
// @Param        before  query     string  false  "Cursor: id ulasan terakhir yang sudah dilihat"
// @Param        limit   query     int     false  "Jumlah per halaman, 1 sampai 50"  default(10)
// @Success      200     {object}  model.ReviewPage
// @Failure      400     {object}  model.ErrorRespone
// @Security     BearerAuth
// @Router       /courses/{id}/reviews [get]
func (s *ReviewService) ListByCourse(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id must be a positive number")
	}

	q := model.ReviewQuery{
		CourseID: id,
		Tag:      strings.ToLower(strings.TrimSpace(c.Query("tag"))),
		Before:   c.Query("before"),
		Limit:    c.QueryInt("limit", 10),
	}
	if q.Limit < 1 || q.Limit > 50 {
		q.Limit = 10
	}

	want := q.Limit
	q.Limit = want + 1

	reviews, err := s.reviews.FindByCourse(ctx, q)
	if err != nil {
		return translateReviewError(err)
	}

	hasMore := len(reviews) > want
	if hasMore {
		reviews = reviews[:want]
	}

	next := ""
	if hasMore {
		next = reviews[len(reviews)-1].ID
	}

	return c.JSON(model.ReviewPage{
		Success: true,
		Message: "daftar ulasan berhasil diambil",
		Data:    reviews,
		Meta: model.ReviewPageMeta{
			Limit:      want,
			HasMore:    hasMore,
			NextBefore: next,
		},
	})
}

// GET /courses/:id/reviews/summary
//
// Summary godoc
//
// @Summary      Ringkasan statistik ulasan sebuah kelas
// @Tags         reviews
// @Produce      json
// @Param        id   path      int  true  "ID kelas"
// @Success      200  {object}  model.WebResponse{data=model.ReviewSummary}
// @Failure      400  {object}  model.ErrorRespone  "ID harus berupa angka positif"
// @Failure      401  {object}  model.ErrorRespone
// @Failure      404  {object}  model.ErrorRespone  "Kelas tidak ditemukan"
// @Security     BearerAuth
// @Router       /courses/{id}/reviews/summary [get]
func (s *ReviewService) Summary(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id must be a positive number")
	}

	if _, err := s.courses.FindByID(ctx, id); err != nil {
		return translateReviewError(err)
	}

	summary, err := s.reviews.Summary(ctx, id)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, "ringkasan ulasan berhasil diambil", summary)
}

// GET /reviews/:id
//
// Get godoc
//
// @Summary      Detail satu ulasan berdasarkan ID
// @Tags         reviews
// @Produce      json
// @Param        id   path      string  true  "ID ulasan (ObjectID)"
// @Success      200  {object}  model.WebResponse{data=model.Review}
// @Failure      400  {object}  model.ErrorRespone  "ID ulasan tidak valid"
// @Failure      401  {object}  model.ErrorRespone
// @Failure      404  {object}  model.ErrorRespone  "Ulasan tidak ditemukan"
// @Security     BearerAuth
// @Router       /reviews/{id} [get]
func (s *ReviewService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	review, err := s.reviews.FindByID(ctx, c.Params("id"))
	if err != nil {
		return translateReviewError(err)
	}

	return helper.Success(c, "ulasan ditemulan", review)
}

// PATCH /reviews/:id
//
// Update godoc
//
// @Summary      Mengubah ulasan milik sendiri
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id    path      string                     true  "ID ulasan (ObjectID)"
// @Param        body  body      model.UpdateReviewRequest  true  "Field ulasan yang diperbarui"
// @Success      200   {object}  model.WebResponse{data=model.Review}
// @Failure      400   {object}  model.ErrorRespone  "ID, JSON tidak valid, atau tidak ada field yang diubah"
// @Failure      401   {object}  model.ErrorRespone
// @Failure      403   {object}  model.ErrorRespone  "Bukan pemilik ulasan"
// @Failure      404   {object}  model.ErrorRespone
// @Failure      422   {object}  model.ErrorRespone  "Validasi gagal"
// @Security     BearerAuth
// @Router       /reviews/{id} [patch]
func (s *ReviewService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	var req model.UpdateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON Body invalid")
	}

	if req.Rating == nil && req.Comment == nil && req.Tags == nil {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	if req.Comment != nil {
		trimmed := strings.TrimSpace(*req.Comment)
		req.Comment = &trimmed
	}

	if req.Rating != nil {
		normalized := NormalizeTags(*req.Tags)
		req.Tags = &normalized
	}

	if errs := helper.ValidateStruct(req, *s.appValidator); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.reviews.UpdateOwn(ctx, c.Params("id"), current.UserID, req)
	if err != nil {
		return translateReviewError(err)
	}

	return helper.Success(c, "ulasan berhasil diperbaharui", updated)
}

// DELETE /reviews/:id
//
// Delete godoc
//
// @Summary      Menghapus ulasan
// @Description  Pemilik ulasan atau moderator (permission review:moderate) dapat menghapus ulasan.
// @Tags         reviews
// @Param        id   path  string  true  "ID ulasan (ObjectID)"
// @Success      204  "Ulasan berhasil dihapus; tanpa body"
// @Failure      400  {object}  model.ErrorRespone  "ID ulasan tidak valid"
// @Failure      401  {object}  model.ErrorRespone
// @Failure      403  {object}  model.ErrorRespone  "Bukan pemilik ulasan"
// @Failure      404  {object}  model.ErrorRespone
// @Security     BearerAuth
// @Router       /reviews/{id} [delete]
func (s *ReviewService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	anyOwner := s.perms.Can(current.Role, "review:moderate")
	if err := s.reviews.Delete(ctx, c.Params("id"), current.UserID, anyOwner); err != nil {
		return translateReviewError(err)
	}

	return helper.NoContent(c)
}

// POST /reviews/:id/replies
//
// Reply godoc
//
// @Summary      Menambahkan balasan ke sebuah ulasan
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id    path      string              true  "ID ulasan (ObjectID)"
// @Param        body  body      model.ReplyRequest  true  "Isi balasan"
// @Success      201   {object}  model.WebResponse{data=model.Review}
// @Header       201   {string}  Location  "URL ulasan induk"
// @Failure      400   {object}  model.ErrorRespone  "ID atau JSON tidak valid"
// @Failure      401   {object}  model.ErrorRespone
// @Failure      404   {object}  model.ErrorRespone
// @Failure      409   {object}  model.ErrorRespone  "REPLY_LIMIT"
// @Failure      422   {object}  model.ErrorRespone  "Validasi gagal"
// @Security     BearerAuth
// @Router       /reviews/{id}/replies [post]
func (s *ReviewService) Reply(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	var req model.ReplyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON Body invalid")
	}

	req.Body = strings.TrimSpace(req.Body)
	if errs := helper.ValidateStruct(req, *s.appValidator); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.reviews.AddReply(ctx, c.Params("id"), model.Reply{
		UserID:   current.UserID,
		Username: current.Username,
		Body:     req.Body,
	})
	if err != nil {
		return translateReviewError(err)
	}

	return helper.Created(c, "balasan berhasil ditambahkan", updated, "/api/v1/reviews/"+c.Params("id"))
}

// POST /revies/:id/helpful
//
// MarkHelpful godoc
//
// @Summary      Menandai ulasan sebagai membantu (helpful)
// @Tags         reviews
// @Produce      json
// @Param        id   path      string  true  "ID ulasan (ObjectID)"
// @Success      200  {object}  model.WebResponse{data=model.Review}
// @Failure      400  {object}  model.ErrorRespone  "ID ulasan tidak valid"
// @Failure      401  {object}  model.ErrorRespone
// @Failure      404  {object}  model.ErrorRespone
// @Failure      409  {object}  model.ErrorRespone  "OWN_REVIEW atau ALREADY_VOTED"
// @Security     BearerAuth
// @Router       /reviews/{id}/helpful [post]
func (s *ReviewService) MarkHelpful(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	updated, err := s.reviews.MarkHelpful(ctx, c.Params("id"), current.UserID)
	if err != nil {
		return translateReviewError(err)
	}

	return helper.Success(c, "penilaian berhasil masuk", updated)
}

func translateReviewError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id ulasan tidak valid")

	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("data tidak ditemukan")

	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(
			fiber.StatusConflict,
			"ALREADY_REVIEWED",
			"Anda sudah menulis ulasan untuk kelas ini, ubah ulasan yang ada",
		)

	case errors.Is(err, repository.ErrNotOwner):
		return helper.Forbidden("hanya pemilik ulasan yang boleh melakukan ini")

	case errors.Is(err, repository.ErrOwnReview):
		return helper.NewError(
			fiber.StatusConflict,
			"OWN_REVIEW",
			"tidak dapat menilai ulasan sendiri",
		)

	case errors.Is(err, repository.ErrAlreadyVoted):
		return helper.NewError(
			fiber.StatusConflict,
			"ALREADY_VOTED",
			"Anda sudah menilai ulasan ini",
		)

	case errors.Is(err, repository.ErrReplyLimit):
		return helper.NewError(
			fiber.StatusConflict,
			"REPLY_LIMIT",
			"jumlah balasan sudah mencapai batas",
		)

	default:
		return helper.Internal(err)
	}
}
