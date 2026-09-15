# API Students - PostgreSQL Backend

REST API berbasis Go, Fiber, dan PostgreSQL untuk autentikasi user serta pengelolaan data mahasiswa.

## Daftar Isi

- [Prasyarat](#prasyarat)
- [Struktur Database](#struktur-database)
- [Instalasi dan Konfigurasi](#instalasi-dan-konfigurasi)
- [Menjalankan Aplikasi](#menjalankan-aplikasi)
- [Format Response](#format-response)
- [Autentikasi](#autentikasi)
- [API Endpoints](#api-endpoints)
- [Validasi dan Error](#validasi-dan-error)
- [Testing](#testing)
- [Struktur Project](#struktur-project)

## Prasyarat

- Go 1.21 atau lebih baru
- PostgreSQL
- Git

## Struktur Database

Database tidak dibuat atau dimigrasikan otomatis oleh aplikasi. Jalankan migration secara berurutan.

### Migration

```bash
psql -U postgres -d mhs_mgg_tiga -f migrations/001_create_students.sql
psql -U postgres -d mhs_mgg_tiga -f migrations/002_create_users.sql
psql -U postgres -d mhs_mgg_tiga -f migrations/003_auth.sql
```

### Tabel `students`

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | SERIAL | Primary key |
| `nim` | CHAR(9) | Wajib, unik, panjang 9 karakter |
| `name` | VARCHAR(255) | Nama mahasiswa |
| `grade` | NUMERIC(3,2) | Nilai antara 0.00 dan 4.00 |
| `is_active` | BOOLEAN | Default `true` |
| `created_at` | TIMESTAMP | Default waktu saat record dibuat |

Index yang dibuat: `idx_students_nim`, `idx_students_is_active`, dan `idx_students_is_active_created_at`.

### Tabel `users`

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | SERIAL | Primary key |
| `username` | VARCHAR(50) | Wajib, unik tanpa membedakan huruf besar/kecil |
| `email` | VARCHAR(255) | Wajib |
| `password` | VARCHAR(255) | Hash bcrypt, tidak dikirim dalam response |
| `is_active` | BOOLEAN | Default `true` |
| `role` | VARCHAR(20) | Default `user` |
| `created_at` | TIMESTAMPTZ | Default `NOW()` |

Migration `003_auth.sql` juga membuat tabel `refresh_tokens`. Nilai refresh token disimpan dalam bentuk SHA-256 hash, bukan token asli.

| Kolom | Keterangan |
|---|---|
| `id` | Primary key |
| `user_id` | Foreign key ke `users.id` |
| `token_hash` | Hash refresh token, unik |
| `expires_at` | Waktu kedaluwarsa |
| `revoked_at` | Waktu token dicabut, dapat bernilai NULL |
| `created_at` | Waktu record dibuat |

## Instalasi dan Konfigurasi

```bash
git clone <repository-url>
cd mhs-mgg-tiga
go mod download
```

Buat file `.env` pada root project. Aplikasi membaca konfigurasi melalui `godotenv` dan environment sistem.

| Variable | Default | Keterangan |
|---|---|---|
| `APP_NAME` | `api-backend` | Nama aplikasi Fiber |
| `APP_PORT` | `3000` | Port HTTP |
| `ALLOWED_ORIGINS` | `http://localhost:5173/` | Origin yang diizinkan CORS |
| `DB_USER` | `postgres` | User PostgreSQL |
| `DB_PASSWORD` | kosong | Password PostgreSQL |
| `DB_HOST` | `127.0.0.1` | Host PostgreSQL |
| `DB_PORT` | `5432` | Port PostgreSQL |
| `DB_NAME` | `database_name` | Nama database |
| `DB_SSLMODE` | `disable` | Mode SSL PostgreSQL |
| `DB_MAX_CONNS` | `10` | Maksimum koneksi pool |
| `JWT_SECRET` | tidak ada | Secret minimal 32 karakter |
| `JWT_ISSUER` | `be-prak` | Nilai issuer JWT |
| `JWT_ACCESS_TTL_MINUTES` | `15` | Masa berlaku access token |
| `JWT_REFRESH_TTL_DAYS` | `7` | Masa berlaku refresh token |

Contoh konfigurasi:

```env
APP_NAME=api-backend
APP_PORT=3000
ALLOWED_ORIGINS=http://localhost:5173/
DB_USER=postgres
DB_PASSWORD=
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=mhs_mgg_tiga
DB_SSLMODE=disable
DB_MAX_CONNS=10
JWT_SECRET=ganti-dengan-secret-minimal-32-karakter
JWT_ISSUER=be-prak
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7
```

Jangan commit `.env` ke repository.

## Menjalankan Aplikasi

```bash
go run .
```

Server berjalan pada `http://localhost:3000` secara default. Aplikasi melakukan ping PostgreSQL saat membuat connection pool; jika koneksi gagal, aplikasi tidak dapat berjalan normal.

## Format Response

Response sukses menggunakan bentuk berikut:

```json
{
  "success": true,
  "message": "...",
  "data": {},
  "meta": {}
}
```

`meta` hanya digunakan pada endpoint list. Response validasi memiliki field `errors`:

```json
{
  "success": false,
  "message": "validation fail",
  "errors": {
    "field": "pesan error"
  }
}
```

## Autentikasi

Endpoint `/auth/register`, `/auth/login`, `/auth/refresh`, dan `/auth/logout` bersifat publik. Endpoint `/auth/me`, `/users`, dan `/students` memerlukan header:

```http
Authorization: Bearer <access_token>
```

Access token adalah JWT. Refresh token digunakan pada endpoint `/auth/refresh` untuk memperoleh pasangan token baru. Saat logout, refresh token dapat dikirim untuk dicabut.

Semua method `POST`, `PUT`, dan `PATCH` pada route API harus menggunakan:

```http
Content-Type: application/json
```

Login memiliki rate limit maksimum 5 request per IP dalam 1 menit. Request tanpa atau dengan format Bearer token yang tidak valid akan menerima status `401 Unauthorized`.

## API Endpoints

Base URL: `http://localhost:3000/api/v1`

### Health Check

```http
GET /health
```

Endpoint publik untuk memeriksa koneksi database.

Response berhasil:

```json
{
  "success": true,
  "message": "server and database is OK!",
  "data": null
}
```

Jika database tidak dapat dihubungi, response berstatus `503` dengan pesan `database can't be reached`.

### Authentication

#### Register

```http
POST /auth/register
Content-Type: application/json

{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "Password1"
}
```

Username minimal 3 karakter dan hanya boleh berisi huruf, angka, titik, serta underscore. Password minimal 8 karakter dan harus mengandung huruf serta angka. Username disimpan setelah trim spasi. Response berhasil berstatus `201 Created`.

#### Login

```http
POST /auth/login
Content-Type: application/json

{
  "username": "john_doe",
  "password": "Password1"
}
```

Response berhasil berisi `access_token`, `refresh_token`, `token_type`, dan `expired_in`:

```json
{
  "success": true,
  "message": "login berhasil",
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "<opaque-token>",
    "token_type": "Bearer",
    "expired_in": 900
  }
}
```

Username atau password yang salah menghasilkan `401`. User yang tidak aktif menghasilkan `403`.

#### Refresh Token

```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "<refresh-token>"
}
```

Endpoint ini mencabut refresh token lama dan mengembalikan pasangan token baru.

#### Logout

```http
POST /auth/logout
Content-Type: application/json

{
  "refresh_token": "<refresh-token>"
}
```

Logout tetap mengembalikan status `200`. Jika refresh token dikirim, token tersebut dicabut.

#### Current User

```http
GET /auth/me
Authorization: Bearer <access-token>
```

Mengembalikan data user yang terhubung dengan subject pada access token. Password tidak pernah dikirim karena field tersebut memiliki tag JSON `-`.

### Users

Semua endpoint users memerlukan Bearer access token.

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/users/` | List user dengan pagination, search, filter `is_active`, sort, dan order |
| `GET` | `/users/:id` | Ambil user berdasarkan ID |
| `POST` | `/users/` | Buat user baru |
| `PATCH` | `/users/:id` | Update sebagian data user |
| `DELETE` | `/users/:id` | Hapus user |

Contoh membuat user:

```http
POST /users/
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "username": "jane_doe",
  "email": "jane@example.com",
  "password": "Password1"
}
```

Route `PUT /users/:id` belum didaftarkan pada `route/route.go`, walaupun tipe `ReplaceUserRequest` dan method `Replace` tersedia di service.

### Students

Semua endpoint students memerlukan Bearer access token.

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/students/` | List mahasiswa dengan pagination dan filter |
| `GET` | `/students/:id` | Ambil mahasiswa berdasarkan ID |
| `POST` | `/students/` | Tambah mahasiswa |
| `PUT` | `/students/:id` | Ganti data nama, grade, dan status aktif |
| `PATCH` | `/students/:id` | Update sebagian data mahasiswa |
| `DELETE` | `/students/:id` | Hapus mahasiswa |

Contoh membuat mahasiswa:

```http
POST /students/
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "nim": "123456789",
  "name": "John Doe",
  "grade": 3.50
}
```

Query parameter list yang didukung:

| Parameter | Default | Keterangan |
|---|---|---|
| `page` | `1` | Nomor halaman |
| `limit` | `10` | Jumlah data per halaman, maksimum `100` |
| `search` | kosong | Mencari pada nama atau NIM |
| `sort` | `id` | `id`, `nim`, `name`, atau `grade` |
| `order` | `asc` | `asc` atau `desc` |
| `is_active` | kosong | `true` atau `false` |
| `grade_start` | `0.00` | Nilai minimum |
| `grade_end` | `4.00` | Nilai maksimum |

Contoh:

```http
GET /students/?page=1&limit=10&search=john&sort=name&order=asc&is_active=true&grade_start=0&grade_end=4
Authorization: Bearer <access-token>
```

Response list menggunakan `data` array dan `meta`:

```json
{
  "success": true,
  "message": "student list successfully retreived",
  "data": [],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 0,
    "total_pages": 0
  }
}
```

## Validasi dan Error

| Status | Kondisi umum |
|---|---|
| `200 OK` | Request berhasil |
| `201 Created` | Resource berhasil dibuat |
| `204 No Content` | Resource berhasil dihapus |
| `400 Bad Request` | JSON, ID, atau input request tidak valid |
| `401 Unauthorized` | Belum login atau token invalid/kedaluwarsa |
| `403 Forbidden` | Akun tidak aktif |
| `409 Conflict` | Username atau NIM sudah digunakan |
| `415 Unsupported Media Type` | Body method tidak memakai `application/json` |
| `422 Unprocessable Entity` | Validasi field gagal |
| `429 Too Many Requests` | Login melebihi rate limit |
| `500 Internal Server Error` | Kegagalan internal atau database |
| `503 Service Unavailable` | Health check gagal melakukan ping database |

Pesan error ditentukan oleh handler service dan dapat berbeda antar endpoint. Error repository `ErrNotFound` dan `ErrDuplicate` diterjemahkan oleh service menjadi response HTTP yang sesuai.

## Testing

Jalankan unit test:

```bash
go test ./...
```

Test yang tersedia saat ini mencakup business rules student. Untuk pengujian manual, gunakan urutan berikut:

```bash
# Health check
curl http://localhost:3000/api/v1/health

# Register
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"john_doe","email":"john@example.com","password":"Password1"}'

# Login, lalu simpan access_token dari response
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"john_doe","password":"Password1"}'

# Akses endpoint protected
curl http://localhost:3000/api/v1/students/ \
  -H "Authorization: Bearer <access-token>"
```

## Struktur Project

```text
mhs-mgg-tiga/
├── main.go
├── app/
│   ├── model/          # Model domain dan request/response types
│   ├── repository/     # Interface dan query PostgreSQL
│   └── service/        # Handler HTTP dan business rules
├── config/              # Fiber, environment, dan logger
├── database/            # Pembuatan connection pool PostgreSQL
├── helper/              # JWT, password, request, dan response helpers
├── middleware/          # Auth, CORS, JSON check, rate limiter, logger
├── migrations/          # SQL schema students, users, dan refresh tokens
├── route/               # Registrasi endpoint dan dependency wiring
├── AI-USAGE.md          # Catatan penggunaan AI pada pengembangan
├── go.mod
└── README.md
```

### Alur Request

```text
HTTP request
  -> Fiber middleware
  -> route dan authentication middleware
  -> service handler
  -> repository
  -> PostgreSQL
  -> WebResponse
```

Repository menggunakan query parameterized. Password disimpan menggunakan bcrypt, access token menggunakan JWT, dan refresh token disimpan sebagai hash.
