package service

import (
	"errors"
	"strconv"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/gofiber/fiber/v2"
)

var (
	errCreditLimit = errors.New("batas SKS terlampaui")
)

type CourseService struct {
	courses repository.CourseRepository
	tx      *repository.TxManager
}

func NewCourseService(courses repository.CourseRepository, txManager *repository.TxManager) *CourseService {
	return &CourseService{
		courses: courses,
		tx:      txManager,
	}
}

// ---------- GET /courses ----------

func (s *CourseService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	courses, err := s.courses.FindAll(ctx)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(
		c,
		"daftar kelas berhasil diambil",
		courses,
	)
}

// ---------- GET /courses/mine ----------

func (s *CourseService) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courses, err := s.courses.FindByStudent(ctx, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}

	total := 0
	for _, course := range courses {
		total += course.Credits
	}

	return helper.Success(
		c,
		"KRS berhasil diambil",
		fiber.Map{
			"courses":       courses,
			"total_credits": total,
			"max_credit":    MaxCreditsPerSemester,
		},
	)
}

// ---------- GET /courses/:id ----------

func (s *CourseService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	course, err := s.courses.FindByID(ctx, id)
	if err != nil {
		return translateCourseError(err)
	}

	return helper.Success(
		c,
		"kelas ditemukan",
		course,
	)
}

func translateCourseError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("kelas tidak ditemukan")
	default:
		return helper.Internal(err)
	case errors.Is(err, repository.ErrNotEnrolled):
		return helper.NewError(fiber.StatusConflict, "NOT_ENROLLED", "Anda tidak terdaftar di kelas tersebut")
	case errors.Is(err, errCreditLimit):
		return helper.NewError(fiber.StatusConflict, "CREDIT_LIMIT_EXCEEDED", "batas "+strconv.Itoa(MaxCreditsPerSemester)+" SKS terlampaui")
	}
}

// ---------- POST /courses/:id/enroll ----------

func (s *CourseService) Enroll(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var enrolled model.Course

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		// Seluruh query di bawah ini WAJIB memakai repo, bukan s.courses.
		// s.courses terikat pada pool: query lewat s.courses berjalan di
		// luar transaction dan tidak ikut dibatalkan bila langkah
		// berikutnya gagal.
		repo := s.courses.WithTx(tx)

		// 1. Kunci KRS milik mahasiswa ini.
		// Request lain dari mahasiswa yang SAMA harus menunggu sampai
		// transaction ini selesai.
		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}

		// 2. Periksa batas SKS. Aman, karena tidak ada request lain milik
		// mahasiswa ini yang dapat menambah SKS di antara pemeriksaan
		// dan penyimpanan.
		course, err := repo.FindByID(ctx, courseID)
		if err != nil {
			return err
		}

		taken, err := repo.SumCredits(ctx, current.UserID)
		if err != nil {
			return err
		}

		if !WithinCreditLimit(taken, course.Credits) {
			return errCreditLimit
		}

		// 3. Ambil satu kursi secara atomik.
		enrolled, err = repo.IncrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}

		// 4. Catat pendaftarannya. Bila ternyata sudah terdaftar, error
		// di sini membatalkan juga penambahan kursi pada langkah 3.
		return repo.InsertEnrollment(ctx, courseID, current.UserID)
	})

	if err != nil {
		return translateCourseError(err)
	}

	return helper.Created(
		c,
		"berhasil mendaftar kelas",
		enrolled,
		"/api/v1/courses/"+strconv.Itoa(courseID),
	)
}

// ---------- DELETE /courses/:id/enroll ----------

func (s *CourseService) Drop(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		// Kunci KRS mahasiswa agar perubahan pendaftaran berjalan
		// secara berurutan untuk mahasiswa yang sama.
		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}

		// Hapus pendaftaran mahasiswa dari kelas.
		if err := repo.DeleteEnrollment(
			ctx,
			courseID,
			current.UserID,
		); err != nil {
			return err
		}

		// Kurangi jumlah peserta hanya setelah pendaftaran berhasil dihapus.
		_, err := repo.DecrementEnrolled(ctx, courseID)
		return err
	})

	if err != nil {
		return translateCourseError(err)
	}

	return helper.NoContent(c)
}
