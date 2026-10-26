package service

import (
	"strconv"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/gofiber/fiber/v2"
)

type SupervisorServices struct {
	supervisors  repository.SupervisorRepository
	tx           *repository.TxManager
	appValidator *helper.AppValidator
}

func NewSupervisorServices(supervisors repository.SupervisorRepository, txManager *repository.TxManager, appValidator *helper.AppValidator) *SupervisorServices {
	return &SupervisorServices{
		supervisors:  supervisors,
		tx:           txManager,
		appValidator: appValidator,
	}
}

// ---------- GET /supervisors ----------

func (s *SupervisorServices) List(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	supervisors, err := s.supervisors.FindAll(ctx)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(
		c,
		"daftar supervisor berhasil diambil",
		supervisors,
	)
}

// ---------- POST /supervisors/mine ----------

func (s *SupervisorServices) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	_, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.StudentIDReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	supervisor, err := s.supervisors.FindByStudent(ctx, req.StudentID)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(
		c,
		"Supervisor berhasil diambil",
		fiber.Map{
			"supervisor": supervisor,
		},
	)
}

// ---------- GET /supervisors/:id ----------

func (s *SupervisorServices) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	supervisor, err := s.supervisors.FindByID(ctx, id)
	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.Success(
		c,
		"supervisor ditemukan",
		supervisor,
	)
}

// -- POST /supervisors/:id/assign
// dengan body {"student_id": ...}, dijaga permission baru (misalnya supervision:manage).
// Gunakan UPDATE bersyarat untuk kuota dan kunci baris students untuk mencegah satu mahasiswa ditetapkan ke dua pembimbing secara serentak.
func (s *SupervisorServices) Assign(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	_, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	supervisorID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.AssignSupervisorReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	var supervised model.Supervisor

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		// Seluruh query di bawah ini WAJIB memakai repo, bukan s.courses.
		// s.courses terikat pada pool: query lewat s.courses berjalan di
		// luar transaction dan tidak ikut dibatalkan bila langkah
		// berikutnya gagal.
		repo := s.supervisors.WithTx(tx)

		// 1. Kunci KRS milik mahasiswa ini.
		// Request lain dari mahasiswa yang SAMA harus menunggu sampai
		// transaction ini selesai.
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		// 2. Periksa apakah student sudah memiliki supervisor
		_, err := repo.FindByStudent(ctx, req.StudentID)
		if err == nil {
			return errCreditLimit
		}

		// 3. Ambil satu secara atomik.
		supervised, err = repo.IncrementSupervision(ctx, supervisorID)
		if err != nil {
			return err
		}

		// 4. Catat pendaftarannya. Bila ternyata sudah terdaftar, error
		// di sini membatalkan juga penambahan kursi pada langkah 3.
		return repo.InsertSupervision(ctx, supervisorID, req.StudentID)
	})

	if err != nil {
		return translateSupervisorError(err)
	}

	return helper.Created(
		c,
		"berhasil mendaftar supervisor",
		supervised,
		"/api/v1/supervisors/"+strconv.Itoa(supervisorID),
	)
}

// -- POST /supervisions/switch
// untuk mengganti pembimbing.
// Wajib atomik — bila pembimbing baru penuh, mahasiswa tetap dengan pembimbing lama — dan wajib bebas deadlock.
func (s *SupervisorServices) Switch(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.SwitchSupervisorReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req, *s.appValidator); errs != nil {
		return helper.Validation(errs)
	}

	if req.FromSupervisorID == req.ToSupervisorID {
		return helper.Validation(map[string]string{
			"to_course_id": "tidak boleh sama dengan from_course_id",
		})
	}

	var joined model.Supervisor

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)

		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}

		// Kunci KEDUA supervisor lebih dulu, dalam urutan id menaik. Tanpa
		// langkah ini, mahasiswa A yang pindah 1 -> 2 dan mahasiswa B
		// yang pindah 2 -> 1 pada saat yang sama akan saling menunggu
		// kunci milik satu sama lain: deadlock.
		if err := repo.LockSupervisor(
			ctx,
			req.FromSupervisorID,
			req.ToSupervisorID,
		); err != nil {
			return err
		}

		_, err := repo.FindByID(ctx, req.FromSupervisorID)
		if err != nil {
			return err
		}

		_, err = repo.FindByID(ctx, req.ToSupervisorID)
		if err != nil {
			return err
		}

		if err := repo.DeleteSupervision(
			ctx,
			req.FromSupervisorID,
			req.StudentID,
		); err != nil {
			return err
		}

		if _, err := repo.DecrementSupervision(
			ctx,
			req.FromSupervisorID,
		); err != nil {
			return err
		}

		joined, err = repo.IncrementSupervision(ctx, req.ToSupervisorID)
		if err != nil {
			return err
		}

		return repo.InsertSupervision(
			ctx,
			req.ToSupervisorID,
			current.UserID,
		)
	})

	if err != nil {
		return translateCourseError(err)
	}

	return helper.Success(
		c,
		"berhasil pindah supervisor",
		joined,
	)
}

// -- PATCH /supervisors/:id
// untuk mengubah kuota dengan optimistic locking.
func (s *SupervisorServices) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.UpdateQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req, *s.appValidator); errs != nil {
		return helper.Validation(errs)
	}

	// Satu pernyataan UPDATE saja, sehingga tidak perlu WithTx:
	// setiap pernyataan tunggal di PostgreSQL sudah atomik dengan sendirinya.
	updated, err := s.supervisors.UpdateQuota(
		ctx,
		id,
		req.Quota,
		req.Version,
	)
	if err != nil {
		return translateCourseError(err)
	}

	return helper.Success(
		c,
		"kuota berhasil diubah",
		updated,
	)
}

func translateSupervisorError(err error) error {
	switch {
	default:
		return helper.Internal(err)
	}
}
