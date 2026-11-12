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
	repo         repository.UserRepository
	perms        *helper.PermissionSet
	appValidator *helper.AppValidator
}

func NewUserHandler(repo repository.UserRepository, perms *helper.PermissionSet, appValidator *helper.AppValidator) *UserHandler {
	return &UserHandler{repo: repo, perms: perms, appValidator: appValidator}
}

// ListAll godoc
//
// @Summary      Daftar user dengan cursor pagination
// @Description  Membutuhkan permission user:list. Mendukung content negotiation JSON (default) atau CSV melalui header Accept: text/csv.
// @Tags         users
// @Produce      json
// @Produce      text/csv
// @Param        search     query     string  false  "Cari username atau email"
// @Param        role       query     string  false  "Filter berdasarkan role"
// @Param        is_active  query     bool    false  "Filter status aktif"
// @Param        cursor     query     string  false  "Cursor halaman berikutnya (meta.next_cursor)"
// @Success      200        {object}  model.WebResponseCursor{data=[]model.User}
// @Failure      401        {object}  model.ErrorRespone
// @Failure      403        {object}  model.ErrorRespone
// @Failure      406        {object}  model.ErrorRespone  "Header Accept tidak didukung"
// @Security     BearerAuth
// @Router       /users [get]
func (h *UserHandler) ListAll(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	// q := helper.ParseListQuery(c)
	q := helper.ParseCursorQuery(c)

	// users, total, err := h.repo.FindAll(ctx, q)
	// if err != nil {
	// 	return helper.Internal(err)
	// }

	// return helper.OkList(c, "berhasil mendapatkan semua user", users, &model.Meta{
	// 	Page:       q.Page,
	// 	Limit:      q.Limit,
	// 	TotalPages: CountTotalPages(total, q.Limit),
	// 	Total:      total,
	// })

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

	if format == helper.FormatCSV {
		return helper.WriteUsersCSV(c, rows)
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar user berhasil diambil", rows, meta)
}

// Get godoc
//
// @Summary      Detail satu user berdasarkan ID
// @Description  Diakses oleh pemilik akun atau user dengan permission user:read:any.
// @Tags         users
// @Produce      json
// @Param        id   path      int  true  "ID user"
// @Success      200  {object}  model.WebResponse{data=model.User}
// @Failure      400  {object}  model.ErrorRespone  "ID tidak valid"
// @Failure      401  {object}  model.ErrorRespone
// @Failure      403  {object}  model.ErrorRespone
// @Failure      404  {object}  model.ErrorRespone
// @Security     BearerAuth
// @Router       /users/{id} [get]
func (h *UserHandler) Get(c *fiber.Ctx) error {
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

	if !CanAccessUser(current, id, h.perms, "user:read:any") {
		return helper.Forbidden("anda tidak memiliki hak untuk mengakses pengguna ini")
	}

	user, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return translateErr(err, "user")
	}

	return helper.Ok(c, "berhasil mendapatkan user", user)
}

// Create godoc
//
// @Summary      Membuat user baru (admin)
// @Description  Membutuhkan permission user:update:any.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      model.CreateUserRequest  true  "Data user baru"
// @Success      201   {object}  model.WebResponse{data=model.User}
// @Header       201   {string}  Location  "URL user yang baru dibuat"
// @Failure      400   {object}  model.ErrorRespone  "JSON tidak valid"
// @Failure      401   {object}  model.ErrorRespone
// @Failure      403   {object}  model.ErrorRespone
// @Failure      409   {object}  model.ErrorRespone  "Data sudah digunakan"
// @Failure      422   {object}  model.ErrorRespone  "Validasi gagal"
// @Security     BearerAuth
// @Router       /users [post]
func (h *UserHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("json is not valid")
	}

	req.Username = strings.TrimSpace(req.Username)

	// if errs := ValidateCreateUser(req); len(errs) > 0 {
	// 	return helper.Validation(errs)
	// }

	if errs := helper.ValidateStruct(req, *h.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}

	hashedPassword, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(err)
	}

	new, err := h.repo.Create(ctx, model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     "user",
	})
	if err != nil {
		return translateErr(err, "user")
	}

	return helper.Created(c, "user successfully created!", new, "/api/v1/users/"+strconv.Itoa(new.ID))
}

// Replace godoc
//
// @Summary      Memperbarui seluruh data user (PUT)
// @Description  Diakses oleh pemilik akun atau user dengan permission user:update:any.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path      int                       true  "ID user"
// @Param        body  body      model.ReplaceUserRequest  true  "Data lengkap user"
// @Success      200   {object}  model.WebResponse{data=model.User}
// @Failure      400   {object}  model.ErrorRespone  "ID atau JSON tidak valid"
// @Failure      401   {object}  model.ErrorRespone
// @Failure      403   {object}  model.ErrorRespone
// @Failure      404   {object}  model.ErrorRespone
// @Failure      422   {object}  model.ErrorRespone  "Validasi gagal"
// @Security     BearerAuth
// @Router       /users/{id} [put]
func (h *UserHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id invalid")
	}

	var req model.ReplaceUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON invalid")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	if !CanAccessUser(current, id, h.perms, "user:update:any") {
		return helper.Forbidden("anda tidak dapat hak untuk menggantikan data pengguna ini")
	}

	user, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return helper.NotFound("user tidak ditemukan")
	}

	// if errs := ValidateReplaceUser(req); len(errs) > 0 {
	// 	return helper.Validation(errs)
	// }

	if errs := helper.ValidateStruct(req, *h.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}

	updated, err := h.repo.Update(ctx, user)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Ok(c, "berhasil memperbarui data pengguna secara penuh", updated)
}

// Patch godoc
//
// @Summary      Memperbarui sebagian data user (PATCH)
// @Description  Diakses oleh pemilik akun atau user dengan permission user:update:any.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path      int                     true  "ID user"
// @Param        body  body      model.PatchUserRequest  true  "Field user yang diperbarui"
// @Success      200   {object}  model.WebResponse{data=model.User}
// @Failure      400   {object}  model.ErrorRespone  "ID, JSON tidak valid, atau body kosong"
// @Failure      401   {object}  model.ErrorRespone
// @Failure      403   {object}  model.ErrorRespone
// @Failure      404   {object}  model.ErrorRespone
// @Failure      409   {object}  model.ErrorRespone  "Data sudah digunakan"
// @Failure      422   {object}  model.ErrorRespone  "Validasi gagal"
// @Security     BearerAuth
// @Router       /users/{id} [patch]
func (h *UserHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id invalid")
	}

	var req model.PatchUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON invalid")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	if !CanAccessUser(current, id, h.perms, "user:update:any") {
		return helper.Forbidden("anda tidak dapat hak untuk memperbarui data pengguna ini")
	}

	user, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return translateErr(err, "user")
	}

	if IsEmptyPatchUser(req) {
		return helper.BadRequest("Request Body Kosong!")
	}

	if errs := helper.ValidateStruct(req, *h.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}
	updated := ValidatePatchUser(user, req)

	updated_user, err := h.repo.Update(ctx, updated)
	if err != nil {
		return translateErr(err, "user")
	}

	return helper.Ok(c, "berhasil memperbarui user", updated_user)
}

// Delete godoc
//
// @Summary      Menghapus user berdasarkan ID
// @Description  Membutuhkan permission user:delete dan tidak dapat menghapus akun sendiri.
// @Tags         users
// @Param        id   path  int  true  "ID user"
// @Success      204  "User berhasil dihapus; tanpa body"
// @Failure      400  {object}  model.ErrorRespone  "ID tidak valid"
// @Failure      401  {object}  model.ErrorRespone
// @Failure      403  {object}  model.ErrorRespone
// @Failure      404  {object}  model.ErrorRespone
// @Security     BearerAuth
// @Router       /users/{id} [delete]
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("can't delete user: ID Invalid")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	if current.UserID == id {
		return helper.Forbidden("tidak memiliki hak untuk menghapus user milik diri sendiri")
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return translateErr(err, "user")
	}

	return helper.NoContent(c)
}

// AssignRole godoc
//
// @Summary      Mengubah role milik user
// @Description  Membutuhkan permission role:assign. Tidak dapat mengubah role akun sendiri.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path      int                      true  "ID user"
// @Param        body  body      model.AssignRoleRequest  true  "Role baru"
// @Success      200   {object}  model.WebResponse{data=model.User}
// @Failure      400   {object}  model.ErrorRespone  "ID atau JSON tidak valid"
// @Failure      401   {object}  model.ErrorRespone
// @Failure      403   {object}  model.ErrorRespone
// @Failure      422   {object}  model.ErrorRespone  "Role tidak valid atau target adalah akun sendiri"
// @Security     BearerAuth
// @Router       /users/{id}/role [patch]
func (h *UserHandler) AssignRole(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("can't delete user: ID Invalid")
	}

	var userAssignRole model.AssignRoleRequest
	if err := c.BodyParser(&userAssignRole); err != nil {
		return helper.BadRequest("JSON invalid")
	}

	current, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("can't verify your identity")
	}

	errs := ValidateAssignRole(current, id, userAssignRole, h.perms)
	if len(errs) > 0 {
		return helper.Validation(errs)
	}

	result, err := h.repo.UpdateRole(ctx, id, userAssignRole.Role)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Ok(c, "berhasil mengubah role", result)
}
