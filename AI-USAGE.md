# AI Usage Log

## Session 1 - Database Migration & Model Refactoring (2026-09-01)

### Objective
Membuat migration SQL untuk tabel `students` dengan requirement spesifik dan melakukan refactoring handler untuk menggunakan package model dengan proper import.

### Activities Completed

#### 1. Migration File Creation - `migrations/001_create_students.sql`
**Task:** Membuat schema tabel `students` dengan kolom, constraint, dan index yang sesuai requirement.

**Schema Design:**
- **Tabel**: `students`
- **Kolom**:
  - `id` (SERIAL PRIMARY KEY) - Auto-increment unique identifier
  - `nim` (CHAR(9) UNIQUE NOT NULL) - Nomor Induk Mahasiswa, wajib unik
  - `name` (VARCHAR(255) NOT NULL) - Nama mahasiswa
  - `grade` (NUMERIC(3,2) NOT NULL) - IPK range 0.00-4.00 dengan CHECK constraint
  - `is_active` (BOOLEAN DEFAULT true) - Status aktif/tidak aktif
  - `created_at` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP) - Audit timestamp

**Constraints:**
- PRIMARY KEY pada `id`
- UNIQUE constraint pada `nim` (menjamin keunikan di database level)
- CHECK constraint pada `grade` (memvalidasi range 0.00-4.00)
- DEFAULT values untuk `is_active` dan `created_at`

**Indexes:**
1. `idx_students_nim` - B-tree index untuk mempercepat pencarian/filter berdasarkan NIM
2. `idx_students_is_active` - B-tree index untuk filter student aktif/tidak aktif
3. `idx_students_is_active_created_at` - Composite B-tree index untuk query dengan filter is_active dan sorting by created_at

**Rationale:**
- NIM dijaga dengan UNIQUE constraint di database level (bukan hanya di kode) untuk mencegah race condition saat concurrent requests
- Index strategy mendukung query patterns yang sering digunakan aplikasi
- Composite index menghindari multiple separate scans

#### 2. Handler Package Import & Model Refactoring - `handler.go`
**Task:** Update handler.go untuk menggunakan `model.` prefix pada semua types dan constants.

**Changes Applied:**
- Added import: `"github.com/Alan-00280/go-pgsql-mhs.git/app/model"`
- Updated type references:
  - `Student` → `model.Student` (11 occurrences)
  - `CreateStudentReq` → `model.CreateStudentReq` (1 occurrence)
  - `ReplaceStudentReq` → `model.ReplaceStudentReq` (1 occurrence)
  - `PatchStudentReq` → `model.PatchStudentReq` (1 occurrence)
- Updated constants:
  - `MAX_GRADE` → `model.MAX_GRADE` (4 occurrences)
  - `NIM_LENGTH` → `model.NIM_LENGTH` (1 occurrence)

**Implementation Details:**
- Handler menggunakan `StudentRepository` interface dari package repository
- Error translation dengan `translateErr()` untuk convert repository errors ke HTTP responses
- Sentinel errors: `ErrNotFound` dan `ErrDuplicate` dari repository
- Validasi input tetap dilakukan di handler layer sebelum repository call

#### 3. Comprehensive README Documentation - `README.md`
**Task:** Membuat dokumentasi lengkap project termasuk schema, setup, API endpoints, dan penjelasan desain.

**Documentation Includes:**
- Prasyarat (Go, PostgreSQL, Git)
- **Skema Tabel Database**: Deskripsi lengkap kolom, tipe data, constraint
- **Setup Database**: Step-by-step untuk membuat database dan menjalankan migrasi
- **Instalasi & Konfigurasi**: Clone, install dependencies
- **Environment Variables**: Daftar lengkap variables dengan default values
- **API Endpoints**: 7 endpoint dengan request/response examples (GET /health, GET/POST /students, GET/PUT/PATCH/DELETE /students/:id)
- **Error Handling**: HTTP status codes dan sentinel errors
- **Penjelasan Desain**: Struktur project, alur data, pemisahan concerns, keamanan, performance
- **Testing**: Contoh manual testing dengan cURL
- **Troubleshooting**: Common issues dan solutions

### Notes:
- Handler.go sudah menggunakan StudentRepository interface (dependency injection)
- Error handling sudah proper dengan sentinel errors
- Semua dokumentasi sudah lengkap di README.md
- Implementasi repository tinggal menulis SQL queries dengan parameter binding

# Ringkasan Percakapan CHAT-GPT

Percakapan ini digunakan sebagai bantuan dalam pengembangan repository PostgreSQL menggunakan **Go (Golang) dan pgx/pgxpool**, khususnya implementasi operasi CRUD pada `StudentPGRepository`.

### 1. Multiple Return Value pada Go

Dijelaskan bahwa Go mendukung function dengan lebih dari satu return value. Contohnya:

```go
func ParseConfig(connString string) (*pgxpool.Config, error)
```

Penggunaan umumnya:

```go
config, err := pgxpool.ParseConfig(connString)
if err != nil {
    return nil, err
}
```

Pola `value, err` merupakan idiom umum di Go untuk menangani hasil function sekaligus error.

### 2. Penggunaan `CreatedAt`

Model `Student` menggunakan field:

```go
CreatedAt *time.Time `json:"created_at,omitempty"`
```

Field `created_at` digunakan ketika aplikasi membutuhkan informasi waktu pembuatan data, misalnya untuk sorting:

```sql
ORDER BY created_at DESC
```

`CreatedAt` juga perlu tersedia sebagai kolom pada database dan dapat menggunakan default seperti `CURRENT_TIMESTAMP`.

### 3. Dynamic Query dan Pagination

Pada `FindAll`, query dibangun menggunakan `fmt.Sprintf()` untuk bagian yang memang dinamis, seperti kolom sorting, arah sorting, `LIMIT`, dan `OFFSET`.

Contoh:

```go
sqlText := fmt.Sprintf(
    `SELECT id, nim, name, grade, is_active, created_at
     FROM students %s
     ORDER BY %s %s
     LIMIT $%d OFFSET $%d`,
    where,
    sortColumn[q.Sort],
    direction,
    len(args)+1,
    len(args)+2,
)
```

Dijelaskan perbedaan antara:

* `%s` / `%d` → placeholder milik `fmt.Sprintf()` untuk membangun string SQL.
* `$1`, `$2`, `$3`, dst. → parameter placeholder PostgreSQL.

Nilai parameter kemudian dimasukkan melalui:

```go
args = append(args, q.Limit, q.Offset())
```

Mapping parameter harus konsisten dengan posisi `$1`, `$2`, dan seterusnya.

### 4. Menghitung Total Data

Ditemukan kesalahan pada query:

```go
SELECT (*) FROM students
```

Sintaks tersebut tidak valid untuk menghitung jumlah row.

Query yang benar:

```sql
SELECT COUNT(*) FROM students
```

Digunakan pada `FindAll` untuk memperoleh jumlah total data sebelum pagination:

```go
var total int
if err := r.pool.QueryRow(
    ctx,
    "SELECT COUNT(*) FROM students"+where,
    args...,
).Scan(&total); err != nil {
    return nil, 0, fmt.Errorf("[ERROR] count total students: %w", err)
}
```

### 5. `QueryRow().Scan()` dan Struct

Dijelaskan bahwa `pgx` tidak secara langsung melakukan:

```go
.Scan(&s)
```

untuk memetakan hasil query ke struct.

Setiap kolom hasil query perlu dipetakan ke field masing-masing:

```go
.Scan(
    &s.ID,
    &s.NIM,
    &s.Name,
    &s.Grade,
    &s.IsActive,
    &s.CreatedAt,
)
```

Urutan field harus sesuai dengan urutan kolom pada `SELECT`.

### 6. `FindById`

Ditemukan dua kesalahan pada implementasi `FindById`:

```go
"SELECT ... FROM students %s WHERE id = $1"
```

`%s` tidak diperlukan karena query tidak menggunakan `fmt.Sprintf()`.

Selain itu, parameter `id` harus diberikan ke `QueryRow()` karena query menggunakan `$1`.

Bentuk yang benar:

```go
r.pool.QueryRow(
    ctx,
    `SELECT id, nim, name, grade, is_active, created_at
     FROM students
     WHERE id = $1`,
    id,
)
```

### 7. Update dengan `RETURNING`

Pada method `Update`, query menggunakan:

```sql
UPDATE students
SET nim = $1,
    name = $2,
    grade = $3,
    is_active = $4
WHERE id = $5
RETURNING id, nim, name, grade, is_active, created_at
```

Ditemukan kesalahan syntax karena kurang koma:

```sql
grade = $3 is_active = $4
```

seharusnya:

```sql
grade = $3, is_active = $4
```

`RETURNING` digunakan agar PostgreSQL mengembalikan row yang berhasil di-update sehingga hasilnya dapat langsung di-`Scan()` kembali ke struct `Student`.

Jika ID tidak ditemukan, `RETURNING` tidak menghasilkan row sehingga `pgx.ErrNoRows` dapat digunakan untuk mengembalikan `ErrNotFound`.

### Kesimpulan

Bantuan AI dalam percakapan ini digunakan untuk:

* memahami idiom Go seperti `value, error`;
* memahami multiple return value;
* memahami penggunaan `QueryRow`, `Query`, dan `Scan` pada pgx;
* memperbaiki syntax SQL PostgreSQL;
* membangun dynamic query dengan `fmt.Sprintf`;
* memahami parameterized query `$1`, `$2`, dan seterusnya;
* mengimplementasikan sorting dan pagination;
* menghitung total data menggunakan `COUNT(*)`;
* menggunakan `created_at` untuk kebutuhan sorting;
* menggunakan `RETURNING` pada operasi `UPDATE`;
* menangani `pgx.ErrNoRows`, `ErrNotFound`, dan duplicate/unique violation.
