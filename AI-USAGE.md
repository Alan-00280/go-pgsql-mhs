# AI Usage Log

## Session 7 - README & RBAC Documentation Synchronization (2026-09-20)

### Objective
Menyelaraskan dokumentasi README dengan rute yang benar-benar aktif di `route/route.go` serta menambahkan matriks permission RBAC berdasarkan migration dan logika otorisasi.

### Activities Completed

- Membaca `route/route.go` untuk memastikan endpoint yang dijelaskan di README sesuai dengan route yang benar-benar terdaftar.
- Membaca migration `004_rbac.sql` dan `005_student_rbac.sql` untuk memastikan daftar permission role yang tercatat di dokumentasi sesuai dengan data yang benar di database.
- Menambahkan section baru `RBAC Permission Matrix` ke `README.md` mencakup role `admin`, `staff`, dan `user`, serta permission yang relevan untuk `/users` dan `/students`.
- Menjelaskan aturan akses owner-based pada `CanAccessUser()` dan `CanAccessStudent()` agar dokumentasi menggambarkan arah logika bisnis yang sebenarnya.
- Menyusun catatan ini untuk menjaga jejak aktivitas yang sudah dilakukan.

### Validation

- Pemeriksaan manual terhadap `route/route.go`, `004_rbac.sql`, `005_student_rbac.sql`, dan `README.md` menunjukkan konsistensi antara dokumentasi dan implementasi.
- Tidak ada perubahan kode program yang memerlukan build baru; fokus pada dokumentasi dan sinkronisasi informasi.

### Notes

- README sekarang mencerminkan bahwa permission untuk `users` dan `students` bersifat role-based, sementara akses data diri sendiri tetap diizinkan secara otomatis oleh service-level authz check.
- Dokumentasi ini membantu pengembangan di masa depan agar tidak salah mengasumsikan endpoint tanpa permission atau role yang tidak ada.

## Session 6 - Authorization Rules Test Coverage (2026-09-20)

### Objective
Membuat unit test untuk menguji fungsi otorisasi pada `app/service/student_authz_rules.go` dan `app/service/user_authz_rules.go`.

### Activities Completed

- Membaca fungsi `CanAccessStudent` dan `CanAccessUser` serta `ValidateAssignRole` untuk memastikan logika akses sesuai dengan kebutuhan bisnis.
- Membuat file baru `app/service/authz_rules_test.go` yang menguji:
  - owner dapat mengakses data miliknya sendiri
  - user dengan permission yang tepat dapat mengakses data orang lain
  - user tanpa permission ditolak
  - role valid dapat dipasang
  - role kosong atau tidak dikenal ditolak
  - user tidak dapat mengubah role dirinya sendiri
- Menjalankan test yang relevan dengan command `go test ./app/service -run 'TestCanAccess(Student|User)|TestValidateAssignRole'`.
- Menemukan bahwa suite test yang lebih luas di `./app/service` masih gagal karena file `files/common_password.txt` tidak ditemukan pada environment saat ini, namun test otorisasi yang baru dibuat berhasil.

### Validation

- Command: `cd 'd:\programs\unair\backend_lanjut\mhs-mgg-tiga'; go test ./app/service -run 'TestCanAccess(Student|User)|TestValidateAssignRole'`
- Hasil: `ok      github.com/Alan-00280/go-pgsql-mhs.git/app/service      0.749s`

### Notes

- Fokus utama aktivitas ini adalah pengujian perilaku nyata dari fungsi otorisasi, bukan asumsi atau mock.
- Tujuan akhir adalah memastikan aturan akses user/student konsisten dengan kebutuhan RBAC yang ada di project.

## Session 5 - Debugging Login Token Pair (2026-09-15)

### Objective
Memeriksa mengapa endpoint `Login()` mengembalikan JSON dengan struktur `model.TokenPair`, tetapi seluruh value token kosong.

### Activities Completed

- Menelusuri alur `Login()` ke `issueTokenPair()` dan `helper.JWTManager.GenerateAccessToken()`.
- Menemukan bahwa `issueTokenPair()` mengembalikan `model.TokenPair{}` bersama `nil` error ketika pembuatan JWT, random refresh token, atau penyimpanan token gagal.
- Menemukan ketidaksesuaian algoritma JWT: token dibuat dengan `ES256` menggunakan secret `[]byte`, sedangkan parser mengharapkan algoritma HMAC.
- Mengubah algoritma signing JWT menjadi `HS256` agar sesuai dengan secret string dan proses parsing.
- Memperbaiki `issueTokenPair()` agar semua error diteruskan ke `Login()` dan tidak lagi menghasilkan response sukses dengan token kosong.
- Memperbaiki `helper.RandomToken()` agar error dari `crypto/rand` tidak disembunyikan.
- Menyesuaikan repository refresh token dengan migration `003_auth.sql`: tabel `refresh_tokens`, kolom `expires_at`, query pencarian token aktif, serta query revoke.

### Validation

- Pemeriksaan diagnostik editor pada `auth_service.go`, `helper/jwt.go`, `helper/security.go`, dan `app/repository/token_repo.go` tidak menemukan error.
- Command `go test ./...` disiapkan untuk verifikasi, tetapi eksekusinya dilewati oleh environment.
- Verifikasi runtime melalui Postman masih perlu dilakukan setelah aplikasi dijalankan ulang dan migration auth dipastikan sudah diterapkan.

### Notes

- Sebelum perbaikan, error token tertutup oleh return `nil`, sehingga `Login()` mengirim HTTP sukses dengan object token kosong.
- Setelah perbaikan, kegagalan pembuatan atau penyimpanan token akan menghasilkan response error server sehingga penyebabnya dapat ditelusuri.

## Session 4 - Authentication Rules Test & Validation Debugging (2026-09-14)

### Objective
Membuat unit test untuk validasi auth (`ValidateRegister`, `ValidateLogin`, dan `checkPasswordStrength`) serta mendiagnosa bug yang muncul saat pengujian.

### Activities Completed

- Membaca `app/service/auth_rules.go` dan `app/model/auth.go` untuk memastikan format request dan aturan validasi yang benar.
- Mencocokkan pola test dengan project yang sudah ada di `app/service/student_rules_test.go`.
- Menulis file baru `app/service/auth_rules_test.go` untuk kasus valid, invalid, dan kombinasi error pada username, email, serta password.
- Menjalankan `go test ./app/service` untuk verifikasi cepat.
- Mendeteksi root cause pada `checkPasswordStrength`: nilai `hasLetter` dan `hasDigit` di-reset ke `false` setiap iterasi karakter, sehingga password valid seperti `Password1` selalu gagal validasi.
- Menetapkan fix yang benar dengan memeriksa apakah password minimal mengandung satu huruf dan satu angka selama iterasi.

### Validation

- Command: `cd 'd:\programs\unair\backend_lanjut\mhs-mgg-tiga'; go test ./app/service`
- Hasil yang teramati: exit code 1, karena bug validasi password yang sedang diperiksa pada saat itu.
- Output menunjukkan password valid `Password1` ditolak karena logika validasi salah, bukan karena file test yang salah.

### Notes
- Aktivitas ini fokus pada pengujian perilaku nyata validator, bukan sekadar mock atau asumsi.
- Tujuan akhir adalah memastikan test auth mencerminkan aturan bisnis yang sebenarnya di `auth_rules.go`.

## Session 3 - Student Business Rules Tests (2026-09-06)

### Objective
Membuat unit test untuk seluruh business rules pada `app/service/student_rules.go` berdasarkan pola test contoh yang diberikan.

### Activities Completed

- Membaca `student_rules.go` dan `example-test.go`.
- Menyesuaikan import serta tipe request dengan model proyek saat ini.
- Membuat `app/service/student_rules_test.go` tanpa menyalakan server atau mengakses PostgreSQL.
- Menguji validasi create untuk nama, NIM, dan batas grade.
- Menguji validasi replace untuk nama dan batas grade.
- Menguji PATCH untuk update field, trim nama, preservasi nilai ketika input invalid, dan empty patch.
- Menguji `IsEmptyPatch` serta perhitungan `CountTotalPages`, termasuk limit tidak valid.

### Validation

- Pemeriksaan diagnostik editor pada `student_rules_test.go` tidak menemukan error.
- Command `gofmt` dan `go test ./...` dicoba, tetapi eksekusinya dilewati oleh environment.
- File `example-test.go` dihapus karena masih memakai module `latihan-fiber` dan tipe `User` yang tidak ada di project ini; cakupan test contohnya sudah dipindahkan ke `student_rules_test.go`.

## Session 2 - Logger Configuration Refactoring (2026-09-04)

### Objective
Merapikan konfigurasi logger HTTP agar hanya menggunakan implementasi middleware yang aktif dan menghapus konfigurasi Fiber logger yang tidak terpasang.

### Activities Completed

#### 1. Logger Configuration Refactoring - `main.go`
**Task:** Menghubungkan logger `slog` yang dikonfigurasi secara eksplisit ke middleware aplikasi.

**Changes Applied:**
- Menghapus konfigurasi `github.com/gofiber/fiber/v2/middleware/logger` yang hanya dibuat sebagai variabel lokal dan tidak digunakan.
- Menghapus import middleware dan `strings` yang hanya terkait kode lama/commented-out.
- Membuat `slog.TextHandler` ke `os.Stdout` dengan level minimum `INFO`.
- Meneruskan instance `slog.Logger` ke `middleware.Register`, yang kemudian digunakan oleh `RequestLogger`.

**Rationale:**
- Mencegah konfigurasi logger yang membingungkan dan tidak pernah dipasang ke Fiber.
- Mempertahankan logging request terstruktur yang sudah mencatat request ID, method, path, status, durasi, dan IP.
- Menjadikan output logger dan level logging dapat dikonfigurasi dari satu titik di `main.go`.

### Validation
- Pemeriksaan diagnostik editor pada `main.go` tidak menemukan error.
- `gofmt` dan `go test ./...` belum dijalankan karena eksekusi terminal dilewati.

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
