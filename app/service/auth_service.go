package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/gofiber/fiber/v2"
)

// PREREQUIREMENT
// HELPER: helper currentUser(c) model.User, helper generateToken() string

// buat konstanta refreshTokenByte = 32
const refreshTokenByte = 32

// buat struct AuthHandler dengan atribut
//
//	users      repository.UserRepository
//	tokens     repository.TokenRepository
//	jwt        *helper.JWTManager
//	refreshTTL time.Duration
type AuthHandler struct {
	users      repository.UserRepository
	tokens     repository.TokenRepository
	jwt        *helper.JWTManager
	refreshTTL time.Duration
	perms      *helper.PermissionSet
	appValidator *helper.AppValidator
}

// buat function yang mengembalikan struct *AuthService dengan parameter sama dengan atribut2 nya
func NewAuthHandler(
	users repository.UserRepository,
	tokens repository.TokenRepository,
	jwt *helper.JWTManager,
	refreshTTL time.Duration,
	perms *helper.PermissionSet,
	appValidator *helper.AppValidator,
) *AuthHandler {
	return &AuthHandler{
		users:      users,
		tokens:     tokens,
		jwt:        jwt,
		refreshTTL: refreshTTL,
		perms:      perms,
		appValidator: appValidator,
	}
}

// buat method untuk AuthService
// setiap method menerima *fiber.Ctx

// --- METHODS ---
// func (s *AuthService) Register() error :
//
//	 POST /auth/register
//		ambil ctx, check body JSON pakai c.BodyParser, trimspace username email, check validasi, hash password, panggil s.users.Create(), kembalikan helper.Created

// Register godoc
//
// @Summary      Mendaftarkan akun baru
// @Description  Role akun baru selalu "user". Password minimal 8 karakter dan memuat huruf serta angka.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.RegisterReq  true  "Data akun baru"
// @Success      201   {object}  model.WebResponse{data=model.User}
// @Failure      400   {object}  model.ErrorRespone  "Body bukan JSON yang valid"
// @Failure      409   {object}  model.ErrorRespone  "Username sudah dipakai"
// @Failure      422   {object}  model.ErrorRespone  "Validasi gagal; detail per field ada di fields"
// @Router       /auth/register [post]
func (s *AuthHandler) Register(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.RegisterReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON invalid!")
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if errs := helper.ValidateStruct(req, *s.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}

	hash_password, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(err)
	}

	user, err := s.users.Create(ctx, model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hash_password,
		Role:     "user",
		IsActive: true,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Conflict("username telah terdaftar")
		}

		return helper.Internal(err)
	}

	return helper.Created(c, "user berhasil didaftarkan", user, "/api/v1/users/"+strconv.Itoa(user.ID))
}

// func (s *AuthService) Login() error :
//
//	 POST /auth/loign
//		ambil ctx, cek body JSON, validasi, panggil s.users.FindByUsername(), helper.VerifyPassword, check is active, buat token pair, kembalikan helper.Success bersama token pair

// Login godoc
//
// @Summary      Login dan mendapatkan pasangan token
// @Description  Dibatasi 5 kali gagal per menit per IP.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.LoginReq  true  "Username dan password"
// @Success      200   {object}  model.WebResponse{data=model.TokenPair}
// @Failure      401   {object}  model.ErrorRespone  "Username atau password salah"
// @Failure      429   {object}  model.ErrorRespone  "Terlalu banyak percobaan login"
// @Router       /auth/login [post]
func (s *AuthHandler) Login(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.LoginReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON invalid!")
	}

	if errs := helper.ValidateStruct(req, *s.appValidator); len(errs) > 0 {
		return helper.Validation(errs)
	}

	user, err := s.users.FindByUsername(ctx, req.Username)
	if err != nil {
		helper.VerifyDummyPassword(req.Password)
		return helper.Unauthorized("password / username salah")
	}

	if !helper.VerifyPassword(user.Password, req.Password) {
		return helper.Unauthorized("password / username salah")
	}

	if !user.IsActive {
		return helper.Forbidden("akun dinonaktifkan")
	}

	// create pair
	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Ok(c, "login berhasil", pair)
}

// func (s *AuthService) Refresh() error :
//
//	 POST /auth/refresh
//		ambil ctx, cek body JSON, cek string token, panggil findActive() pakai token, ambil user, Revoke(), issueTokenPair(), return helper.Success

// Refresh godoc
//
// @Summary      Menukar refresh token dengan pasangan token baru
// @Description  Refresh token lama langsung dicabut (rotation).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.RefreshReq  true  "Refresh token"
// @Success      200   {object}  model.WebResponse{data=model.TokenPair}
// @Failure      401   {object}  model.ErrorRespone  "Refresh token tidak valid, kedaluwarsa, atau sudah dicabut"
// @Router       /auth/refresh [post]
func (s *AuthHandler) Refresh(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.RefreshReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON invalid!")
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		return helper.BadRequest("refresh token kosong")
	}

	hash := helper.SHA256Hex(req.RefreshToken)
	refresh_token, err := s.tokens.FindActive(ctx, hash)
	if err != nil {
		return helper.Unauthorized("token refresh tidak valid / kedaluarwsa")
	}

	user, err := s.users.FindByID(ctx, refresh_token.UserID)
	if err != nil || !user.IsActive {
		return helper.Unauthorized("akun tidak bisa dipakai")
	}

	if err := s.tokens.Revoke(ctx, refresh_token.TokenHash); err != nil {
		return helper.Internal(err)
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Ok(c, "berhasil membuat token baru", pair)
}

// func (s *AuthService) Logout() error :
//
//	 POST /auth/logout
//		ambil ctx, cek body JSON, cek refreshToken, Revoke Token, return helper.Success
//
// Logout godoc
//
// @Summary      Logout dan mencabut refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.RefreshReq  true  "Refresh token yang dicabut"
// @Success      200   {object}  model.WebResponse
// @Failure      400   {object}  model.ErrorRespone  "Body bukan JSON yang valid"
// @Router       /auth/logout [post]
func (s *AuthHandler) Logout(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	var req model.RefreshReq
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("JSON invalid!")
	}

	if strings.TrimSpace(req.RefreshToken) != "" {
		_ = s.tokens.Revoke(ctx, helper.SHA256Hex(req.RefreshToken))
	}

	return helper.Ok(c, "berhasil logout", nil)
}

// func (s *AuthService)  Me() error :
//
//	 GET /auth/me
//		ambil ctx, ambil current user pakai helper function, return helper.success dengan user
//
// Me godoc
//
// @Summary      Profil user yang sedang login beserta daftar permission-nya
// @Tags         auth
// @Produce      json
// @Success      200  {object}  model.WebResponse{data=model.MeResponse}
// @Failure      401  {object}  model.ErrorRespone  "Belum terautentikasi atau user tidak ditemukan"
// @Security     BearerAuth
// @Router       /auth/me [get]
func (s *AuthHandler) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()

	authUser, ok := helper.CurentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	user, err := s.users.FindByID(ctx, authUser.UserID)
	if err != nil {
		return helper.Unauthorized("user tidak ditemukan")
	}

	return helper.Ok(c, "user berhasil ditemukan", model.MeResponse{
		User:        user,
		Permissions: s.perms.PermissionsOf(user.Role),
	})
}

// func (s *AuthService) issueTokenPair(*fiber.Ctx, model.User) (model.TokenPair, error) :
//
//	generate accesstoken jwt.generateAccess(), generate refresh token use helper function, token.Save ini ngesave refresh token, return model.TokenPair
func (s *AuthHandler) issueTokenPair(ctx context.Context, user model.User) (model.TokenPair, error) {
	acces_token, err := s.jwt.GenerateAccessToken(user)
	if err != nil {
		return model.TokenPair{}, err
	}

	refresh_token, err := helper.RandomToken(refreshTokenByte)
	if err != nil {
		return model.TokenPair{}, err
	}

	if err := s.tokens.Save(ctx, model.RefreshToken{
		UserID:    user.ID,
		TokenHash: helper.SHA256Hex(refresh_token),
		ExpiredAt: time.Now().Add(s.refreshTTL),
	}); err != nil {
		return model.TokenPair{}, err
	}

	return model.TokenPair{
		AccessToken:  acces_token,
		RefreshToken: refresh_token,
		TokenType:    "Bearer",
		ExpiredIn:    int(s.refreshTTL.Seconds()),
	}, nil
}
