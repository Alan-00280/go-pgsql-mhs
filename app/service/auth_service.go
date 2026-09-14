package service

// PREREQUIREMENT
// HELPER: helper currentUser(c) model.User, helper generateToken() string

// buat konstanta refreshTokenByte = 32

// buat struct AuthService dengan atribut
//   users      repository.UserRepository
//   tokens     repository.TokenRepository
//   jwt        *helper.JWTManager
//   refreshTTL time.Duration

// buat function yang mengembalikan struct *AuthService dengan parameter sama dengan atribut2 nya
// buat method untuk AuthService
// setiap method menerima *fiber.Ctx

// --- METHODS ---
// func (s *AuthService) Register() error :
//   ambil ctx, check body JSON pakai c.BodyParser, trimspace username email, check validasi, hash password, panggil s.users.Create(), kembalikan helper.Created

// func (s *AuthService) Login() error :
//   ambil ctx, cek body JSON, validasi, panggil s.users.FindByUsername(), helper.VerifyPassword, check is active, buat token pair, kembalikan helper.Success bersama token pair

// func (s *AuthService) Refresh() error :
//   ambil ctx, cek body JSON, cek string token, panggil findActive() pakai token, ambil user, Revoke(), issueTokenPair(), return helper.Success

// func (s *AuthService) Logout() error :
//   ambil ctx, cek body JSON, cek refreshToken, Revoke Token, return helper.Success

// func (s *AuthService)  Me() error :
//   ambil ctx, ambil current user pakai helper function, return helper.success dengan user

// func (s *AuthService) issueTokenPair(*fiber.Ctx, model.User) (model.TokenPair, error) :
//   generate accesstoken jwt.generateAccess(), generate refresh token use helper function, token.Save ini ngesave refresh token, return model.TokenPair
