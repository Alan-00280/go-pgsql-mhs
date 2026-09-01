# API Students - PostgreSQL Backend

API REST untuk manajemen data mahasiswa (students) dengan backend PostgreSQL. Aplikasi ini memindahkan penyimpanan data dari memori lokal ke basis data PostgreSQL dengan mempertahankan seluruh perilaku HTTP.

## Daftar Isi

- [Prasyarat](#prasyarat)
- [Skema Tabel Database](#skema-tabel-database)
- [Setup Database](#setup-database)
- [Instalasi & Konfigurasi](#instalasi--konfigurasi)
- [Environment Variables](#environment-variables)
- [Menjalankan Aplikasi](#menjalankan-aplikasi)
- [API Endpoints](#api-endpoints)
- [Error Handling](#error-handling)
- [Penjelasan Desain](#penjelasan-desain)

---

## Prasyarat

- **Go** 1.21 atau lebih baru
- **PostgreSQL** 12 atau lebih baru
- **Git** (untuk clone repository)

---

## Skema Tabel Database

### Tabel: `students`

```sql
CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    nim CHAR(9) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    grade NUMERIC(3, 2) NOT NULL CHECK (grade >= 0 AND grade <= 4.00),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Deskripsi Kolom

| Kolom | Tipe | Batasan | Deskripsi |
|-------|------|---------|-----------|
| `id` | SERIAL | PRIMARY KEY | Unique identifier, auto-increment |
| `nim` | CHAR(9) | NOT NULL, UNIQUE | Nomor Induk Mahasiswa (wajib unik) |
| `name` | VARCHAR(255) | NOT NULL | Nama mahasiswa |
| `grade` | NUMERIC(3,2) | 0.00 - 4.00 | IPK (Indeks Prestasi Kumulatif) |
| `is_active` | BOOLEAN | DEFAULT true | Status aktivitas mahasiswa |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Waktu pembuatan record |

### Indeks

| Indeks | Tipe | Kolom | Tujuan |
|--------|------|-------|--------|
| `PRIMARY KEY` | Clustered | `id` | Identifikasi unik setiap record |
| `idx_students_nim` | B-tree | `nim` | Mempercepat pencarian berdasarkan NIM |
| `idx_students_is_active` | B-tree | `is_active` | Optimasi filter mahasiswa aktif/tidak aktif |
| `idx_students_is_active_created_at` | B-tree | `is_active, created_at DESC` | Mendukung query filtering + sorting by created_at |

### Penjelasan Keunikan NIM (UNIQUE Constraint)

**Mengapa dijaga di database daripada di kode Go?**

1. **Data Integrity**: Constraint UNIQUE di database menjamin integritas data secara mutlak, tidak peduli aplikasi mana yang mengakses database.
2. **Mencegah Race Condition**: Jika dua request POST datang bersamaan dengan NIM yang sama:
   - Tanpa constraint: kedua request mungkin lolos validasi Go, lalu keduanya disimpan
   - Dengan constraint: database akan menolak yang kedua dengan `ErrDuplicate`
3. **Source of Truth**: Database adalah satu-satunya sumber kebenaran untuk integritas data
4. **Konsistensi**: Mencegah inkonsistensi jika ada aplikasi lain yang juga mengakses database

---

## Setup Database

### 1. Membuat Database Baru

```bash
# Login ke PostgreSQL
psql -U postgres

# Di dalam psql shell:
CREATE DATABASE mhs_mgg_tiga;
```

### 2. Menjalankan Migrasi

```bash
# Masuk ke database
psql -U postgres -d mhs_mgg_tiga -f migrations/001_create_students.sql
```

Atau dari aplikasi Go, skema akan otomatis dibuat saat startup jika belum ada:

```sql
-- Migrasi akan membuat:
-- - Tabel students dengan semua kolom dan batasan
-- - Indeks untuk optimasi query
```

### 3. Verifikasi Struktur

```bash
# Cek tabel
\dt

# Cek kolom tabel
\d students

# Cek indeks
\di
```

---

## Instalasi & Konfigurasi

### 1. Clone Repository

```bash
cd d:\programs\unair\backend_lanjut
git clone <repository-url> mhs-mgg-tiga
cd mhs-mgg-tiga
```

### 2. Install Dependencies

```bash
go mod download
go mod tidy
```

Dependencies utama:
- `github.com/gofiber/fiber/v2` - Framework HTTP
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/joho/godotenv` - Environment variable loader

### 3. Setup Environment Variables

Copy `.env.example` ke `.env`:

```bash
cp .env.example .env
```

Edit `.env` dengan konfigurasi database Anda.

---

## Environment Variables

Semua kredensial database disimpan di file `.env`. **Pastikan `.env` tidak ter-commit ke git** dengan menambahkannya ke `.gitignore`:

```
.env
```

### Daftar Environment Variables

| Variable | Default | Deskripsi | Contoh |
|----------|---------|-----------|--------|
| `DB_USER` | postgres | Username PostgreSQL | postgres |
| `DB_PASSWORD` | (kosong) | Password PostgreSQL | rahasia123 |
| `DB_HOST` | 127.0.0.1 | Host database | localhost |
| `DB_PORT` | 5432 | Port PostgreSQL | 5432 |
| `DB_NAME` | database_name | Nama database | mhs_mgg_tiga |
| `DB_SSLMODE` | disable | Mode SSL | disable, require, prefer |
| `DB_MAX_CONNS` | 10 | Max connection pool | 10 |

### File `.env.example`

```env
DB_USER=postgres
DB_PASSWORD=
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=mhs_mgg_tiga
DB_SSLMODE=disable
DB_MAX_CONNS=10
```

### File `.gitignore`

```
.env
.env.local
*.log
*.out
/tmp
```

---

## Menjalankan Aplikasi

### Development

```bash
go run main.go handler.go helper.go
```

### Build Release

```bash
go build -o api-students.exe
.\api-students.exe
```

### Esperansi Output

```
Server running in 3000...
```

Server akan melisten di `http://localhost:3000`

---

## API Endpoints

### Base URL: `http://localhost:3000/api/v1`

### 1. Health Check

```http
GET /health
```

**Deskripsi**: Memeriksa status server dan koneksi database

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "server and database is OK!",
  "data": null
}
```

**Response (503 Service Unavailable)** - Jika database tidak terhubung:
```json
{
  "success": false,
  "message": "database can't be reached"
}
```

---

### 2. Daftar Mahasiswa (dengan filter & pagination)

```http
GET /students?page=1&limit=10&search=john&sort=name&order=asc&is_active=true&start_grade=0&end_grade=4.00
```

**Query Parameters**:
| Param | Tipe | Default | Deskripsi |
|-------|------|---------|-----------|
| `page` | int | 1 | Nomor halaman |
| `limit` | int | 10 (max 100) | Jumlah per halaman |
| `search` | string | - | Cari di nama atau NIM |
| `sort` | string | id | Sortir by: `id`, `nim`, `name`, `grade` |
| `order` | string | asc | Urutan: `asc`, `desc` |
| `is_active` | bool | - | Filter: `true`, `false` |
| `start_grade` | float | 0.00 | Grade minimal |
| `end_grade` | float | 4.00 | Grade maksimal |

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "student list successfully retreived",
  "data": [
    {
      "id": 1,
      "nim": "123456789",
      "name": "John Doe",
      "grade": 3.50,
      "is_active": true
    }
  ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 50,
    "total_pages": 5
  }
}
```

---

### 3. Ambil Mahasiswa by ID

```http
GET /students/:id
```

**Path Parameters**:
- `id` (int) - Mahasiswa ID

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "student found",
  "data": {
    "id": 1,
    "nim": "123456789",
    "name": "John Doe",
    "grade": 3.50,
    "is_active": true
  }
}
```

**Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "student can't be found"
}
```

---

### 4. Tambah Mahasiswa Baru

```http
POST /students
Content-Type: application/json

{
  "nim": "123456789",
  "name": "John Doe",
  "grade": 3.50
}
```

**Request Body**:
| Field | Tipe | Validasi |
|-------|------|----------|
| `nim` | string | Exactly 9 characters, must be unique |
| `name` | string | Min 3 characters |
| `grade` | float | 0.00 - 4.00 |

**Response (201 Created)**:
```json
{
  "success": true,
  "message": "user berhasil dibuat",
  "data": {
    "id": 1,
    "nim": "123456789",
    "name": "John Doe",
    "grade": 3.50,
    "is_active": true
  }
}
```

**Headers**:
```
Location: /api/v1/students/1
```

**Response (409 Conflict)** - NIM sudah ada:
```json
{
  "success": false,
  "message": "nim already used"
}
```

**Response (422 Unprocessable Entity)** - Validasi gagal:
```json
{
  "success": true,
  "message": "validation fail",
  "errors": {
    "nim": "NIM harus memiliki panjang 9 karakter",
    "grade": "Nilai melebihi rentang 0.00 - 4.00"
  }
}
```

---

### 5. Update Seluruh Data Mahasiswa (PUT)

```http
PUT /students/:id
Content-Type: application/json

{
  "name": "Jane Doe",
  "grade": 3.75,
  "is_active": true
}
```

**Request Body** (semua field wajib):
| Field | Tipe | Validasi |
|-------|------|----------|
| `name` | string | Min 3 characters |
| `grade` | float | 0.00 - 4.00 |
| `is_active` | bool | - |

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "student successfully changed entirely",
  "data": {
    "id": 1,
    "nim": "123456789",
    "name": "Jane Doe",
    "grade": 3.75,
    "is_active": true
  }
}
```

---

### 6. Update Sebagian Data Mahasiswa (PATCH)

```http
PATCH /students/:id
Content-Type: application/json

{
  "name": "Jane Doe",
  "grade": 3.75
}
```

**Request Body** (hanya field yang ingin diubah):
```json
{
  "name": "Jane Doe",        // optional
  "grade": 3.75,              // optional
  "is_active": false          // optional
}
```

Minimal satu field harus dikirim.

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "user berhasil diperbarui sebagian",
  "data": {
    "id": 1,
    "nim": "123456789",
    "name": "Jane Doe",
    "grade": 3.75,
    "is_active": true
  }
}
```

---

### 7. Hapus Mahasiswa

```http
DELETE /students/:id
```

**Response (204 No Content)** - Sukses:
```
(body kosong)
```

**Response (404 Not Found)** - Mahasiswa tidak ditemukan:
```json
{
  "success": false,
  "message": "student can't be found"
}
```

---

## Error Handling

### HTTP Status Codes

| Status | Kondisi |
|--------|---------|
| 200 OK | Request berhasil |
| 201 Created | Resource berhasil dibuat |
| 204 No Content | Request berhasil, tidak ada response body |
| 400 Bad Request | Request tidak valid (parameter salah) |
| 404 Not Found | Resource tidak ditemukan |
| 409 Conflict | NIM sudah terdaftar (duplicate) |
| 415 Unsupported Media Type | Content-Type bukan application/json |
| 422 Unprocessable Entity | Validasi data gagal |
| 500 Internal Server Error | Error di server |
| 503 Service Unavailable | Database tidak terhubung |

### Sentinel Errors (dari Repository)

```go
// app/repository/errors.go
var (
    ErrNotFound = errors.New("record not found")
    ErrDuplicate = errors.New("duplicate value")
)
```

---

## Penjelasan Desain

### 1. Struktur Proyek

```
mhs-mgg-tiga/
├── main.go                          # Entry point
├── handler.go                       # HTTP handlers
├── helper.go                        # Helper functions
├── migrations/
│   └── 001_create_students.sql     # Database schema
├── config/
│   └── env.go                      # Environment configuration
├── database/
│   └── postgres.go                 # PostgreSQL connection pool
├── app/
│   ├── model/
│   │   └── mahasiswa.go            # Data models & structs
│   └── repository/
│       └── student_repository.go   # Data access layer
├── .env.example                     # Environment template
├── go.mod                          # Go dependencies
└── README.md                       # Dokumentasi ini
```

### 2. Alur Data

```
HTTP Request
    ↓
Fiber Router
    ↓
Handler (validation & business logic)
    ↓
Repository (data access)
    ↓
PostgreSQL Database
    ↓
(response sebaliknya)
```

### 3. Pemisahan Concerns

- **Handler**: Validasi HTTP, parsing request, error handling
- **Repository**: Query database, implementasi BusinessLogic
- **Model**: Struct data, request/response types
- **Config**: Environment variable management
- **Database**: Connection pool & utility

### 4. Keamanan

- **Parameterized Queries**: Semua query ke database memakai parameter binding (tidak ada string concatenation)
- **Unique Constraint**: NIM dijaga dengan UNIQUE index di database
- **Environment Variables**: Kredensial disimpan di `.env` (tidak di-commit)
- **Input Validation**: Semua input divalidasi sebelum disimpan

### 5. Performance

- **Connection Pool**: Memakai pgxpool untuk reuse connection
- **Indexes**: B-tree indexes untuk mempercepat pencarian
- **Pagination**: Limit hasil query untuk mengurangi memory & network
- **Prepared Statements**: Query reusable dan teroptimasi

---

## Testing

### Test Manual dengan cURL

```bash
# Health check
curl http://localhost:3000/api/v1/health

# Daftar semua mahasiswa
curl http://localhost:3000/api/v1/students

# Tambah mahasiswa
curl -X POST http://localhost:3000/api/v1/students \
  -H "Content-Type: application/json" \
  -d '{"nim":"123456789","name":"John Doe","grade":3.50}'

# Ambil mahasiswa spesifik
curl http://localhost:3000/api/v1/students/1

# Update seluruh data (PUT)
curl -X PUT http://localhost:3000/api/v1/students/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Jane Doe","grade":3.75,"is_active":true}'

# Update sebagian (PATCH)
curl -X PATCH http://localhost:3000/api/v1/students/1 \
  -H "Content-Type: application/json" \
  -d '{"grade":3.80}'

# Hapus mahasiswa
curl -X DELETE http://localhost:3000/api/v1/students/1
```

### Test dengan Postman

1. Import endpoints ke Postman
2. Set `{{base_url}}` = `http://localhost:3000/api/v1`
3. Test semua endpoints

---

## Troubleshooting

### "database can't be reached"

```bash
# Cek koneksi PostgreSQL
psql -U postgres -h localhost -d mhs_mgg_tiga -c "SELECT 1"

# Cek konfigurasi .env
cat .env

# Pastikan PostgreSQL berjalan
systemctl status postgresql
```

### "NIM telah terdaftar"

```bash
# Cek data existing
psql -U postgres -d mhs_mgg_tiga -c "SELECT nim FROM students"
```

### "berkas .env tidak ditemukan"

```bash
# Copy template
cp .env.example .env

# Edit dengan credential database
nano .env
```

---

## Lisensi

Universitas Airlangga - Backend Assignment

---

## Kontak

Untuk pertanyaan atau issues: hubungi instructor
