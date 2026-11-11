# Berita Acara Perbaikan — Modul 12 (Bagian B1)

Dokumen ini mencatat kesalahan yang ditemukan pada Bagian B1 (*Testing Puzzle*) beserta bukti pemeriksaan dan perbaikannya untuk kebutuhan laporan **D.1 Berita Acara Perbaikan**.

## Ringkasan Tabel Perbaikan

| No | Lokasi (file, fungsi) | Gejala yang diamati | Pemeriksaan yang mengungkapnya | Akar masalah | Perbaikan |
| --- | --- | --- | --- | --- | --- |
| 1 | `app/service/course_rules_test.go`, `TestWithinCreditLimit` | Seluruh test hijau (`PASS`) dengan coverage `WithinCreditLimit` `100.0%`, tetapi saat `<=` pada `WithinCreditLimit` diubah menjadi `<` (uji mutation `T5`), `TestWithinCreditLimit` tetap lolos. | `T5` (`go test -count=1 ./app/service/` pada mutation `<=` → `<`) | Tabel skenario di `TestWithinCreditLimit` tidak memiliki kasus tepat di batas maksimum (`taken + adding == 24`), hanya menguji `3`, `15`, dan `25` SKS. | Menambahkan kasus batas `{name: "tepat di batas 24 SKS", taken: 21, adding: 3, want: true}` pada `TestWithinCreditLimit`, lalu memverifikasi ulang uji mutation `T5` hingga gagal di subtest tersebut. |
| 2 | `app/service/notification_rules_rest.go` → `app/service/notification_rules_test.go`, `TestBuildMessage` | `TestBuildMessage` tidak ikut dijalankan oleh `go test`; pada laporan coverage (`T3`), `TestBuildMessage` tercatat `0.0%` dan `BuildMessage` hanya `33.3%`. | `T3` (`go test -coverprofile=coverage.out ./app/service/` & `go tool cover -func=coverage.out`) | Nama file salah ketik berakhiran `_rest.go` alih-alih `_test.go`, sehingga Go tidak mengenalinya sebagai file test dan paket `testing` ikut terkompilasi ke package produksi. | Mengganti nama file menjadi `app/service/notification_rules_test.go` (`git mv`), sehingga `TestBuildMessage` dieksekusi dan coverage `BuildMessage` mencapai `100.0%`. |
| 3 | `app/service/fakes_test.go`, `(*fakeReviewRepo).FindByCourse` | `go test ./app/service/` gagal compile pada assertion `var _ repository.ReviewRepository = (*fakeReviewRepo)(nil)` dengan error `*fakeReviewRepo does not implement repository.ReviewRepository (wrong type for method FindByCourse): have FindByCourse(context.Context, int), want FindByCourse(context.Context, model.ReviewQuery)` | `compiler` | Signature `FindByCourse` pada `fakeReviewRepo` masih memakai parameter kedua `courseID int`, tidak sesuai dengan interface `repository.ReviewRepository` yang menerima `q model.ReviewQuery` (untuk filter `CourseID`, `Tag`, cursor `Before`, dan `Limit`). | Mengubah signature menjadi `FindByCourse(_ context.Context, q model.ReviewQuery) ([]model.Review, error)` serta mengimplementasikan filter `q.CourseID`, `q.Tag`, validasi & cursor `q.Before` (mengembalikan `repository.ErrInvalidID` bila format cursor tidak valid), dan batas `q.Limit` dengan urutan terbaru lebih dulu. |
| 4 | `app/service/review_service_test.go`, `sharedReviews` / `newTestEnv` | `go test -count=1 -v ./app/service/` lolos, tetapi `go test -shuffle=on -count=5 ./app/service/` gagal pada `TestListReviews_NewCourseIsEmpty`, `TestCreateReview_Success`, dan `TestCreateReview_Twice` (`409 ALREADY_REVIEWED` atau jumlah ulasan tidak 0). | `T2` (`go test -shuffle=on -count=5 ./app/service/`) | `sharedReviews` dideklarasikan sebagai variabel global tingkat package dan dipakai bersama oleh seluruh test melalui `newTestEnv`, sehingga state ulasan antar-test dan antar-iterasi saling bocor ketika urutan diacak atau diulang. | Menghapus variabel global `sharedReviews` dan membuat instance baru `reviews := newFakeReviewRepo()` di dalam `newTestEnv(t)` agar setiap test terisolasi. |
| 5 | `app/service/review_errors_test.go`, `TestTranslateReviewError` | `go test -count=1 ./app/service/` gagal compile dengan error `undefined: service.translateReviewError`. | `compiler` | File memakai `package service_test` (black-box test), padahal fungsi `translateReviewError` tidak di-export oleh package `service` sehingga tidak dapat diakses dari luar package. | Mengubah deklarasi package menjadi `package service` (white-box test), menghapus import `app/service`, dan memanggil `translateReviewError(tt.err)` secara langsung. |
| 6 | `app/service/notification_worker_test.go`, `TestHandleSend_PermanentFailures` | Ketiga subtest gagal dengan pesan `error = skip retry for the task: ..., seharusnya membungkus asynq.SkipRetry`. | `T1` (`go test -count=1 ./...`) | `NotificationWorker.HandleSend` membungkus `asynq.SkipRetry` menggunakan `fmt.Errorf("%w: ...", asynq.SkipRetry, ...)`, tetapi test memeriksa dengan perbandingan langsung `err != asynq.SkipRetry` alih-alih `errors.Is`. | Menambahkan import `"errors"` dan mengganti kondisi menjadi `!errors.Is(err, asynq.SkipRetry)`. |

---

## Detail & Bukti

### 1. `app/service/course_rules_test.go` — `TestWithinCreditLimit` (Uji Mutation T5)

- **Uji mutation `T5` (`<=` → `<` di `app/service/course_rules.go`) sebelum perbaikan**:
  ```text
  === RUN   TestWithinCreditLimit
  === RUN   TestWithinCreditLimit/KRS_masih_kosong
  === RUN   TestWithinCreditLimit/masih_jauh_dari_batas
  === RUN   TestWithinCreditLimit/melewati_batas_satu_SKS
  === RUN   TestWithinCreditLimit/KRS_sudah_penuh
  --- PASS: TestWithinCreditLimit (0.00s)
  PASS
  ok  	github.com/Alan-00280/go-pgsql-mhs.git/app/service	0.277s
  ```
- **Uji mutation `T5` (`<=` → `<` di `app/service/course_rules.go`) setelah perbaikan**:
  ```text
  --- FAIL: TestWithinCreditLimit (0.00s)
      --- FAIL: TestWithinCreditLimit/tepat_di_batas_24_SKS (0.00s)
          course_rules_test.go:28: WithinCreditLimit(21, 3) = false, seharusnya true
  FAIL
  FAIL	github.com/Alan-00280/go-pgsql-mhs.git/app/service	0.295s
  ```
  (`app/service/course_rules.go` dikembalikan dengan `git checkout -- app/service/course_rules.go`.)

### 2. `app/service/notification_rules_test.go` — `TestBuildMessage`

- **Output sebelum perbaikan (`go tool cover -func=coverage.out`)**:
  ```text
  github.com/Alan-00280/go-pgsql-mhs.git/app/service/notification_rules.go:14:		BuildMessage		33.3%
  github.com/Alan-00280/go-pgsql-mhs.git/app/service/notification_rules_rest.go:11:	TestBuildMessage	0.0%
  ```
- **Output setelah perbaikan (`go tool cover -func=coverage.out`)**:
  ```text
  github.com/Alan-00280/go-pgsql-mhs.git/app/service/course_rules.go:13:		WithinCreditLimit		100.0%
  github.com/Alan-00280/go-pgsql-mhs.git/app/service/course_rules.go:18:		CreditsAfterSwap		100.0%
  github.com/Alan-00280/go-pgsql-mhs.git/app/service/notification_rules.go:14:	BuildMessage			100.0%
  github.com/Alan-00280/go-pgsql-mhs.git/app/service/review_rules.go:12:		NormalizeTags			100.0%
  github.com/Alan-00280/go-pgsql-mhs.git/app/service/review_service.go:282:	translateReviewError		100.0%
  ```

### 3. `app/service/fakes_test.go` — `(*fakeReviewRepo).FindByCourse`

- **Output sebelum perbaikan (`go test ./app/service/`)**:
  ```text
  FAIL	github.com/Alan-00280/go-pgsql-mhs.git/app/service [build failed]
  app\service\fakes_test.go:17:37: cannot use (*fakeReviewRepo)(nil) (value of type *fakeReviewRepo) as repository.ReviewRepository value in variable declaration: *fakeReviewRepo does not implement repository.ReviewRepository (wrong type for method FindByCourse)
  		have FindByCourse(context.Context, int) ([]model.Review, error)
  		want FindByCourse(context.Context, model.ReviewQuery) ([]model.Review, error)
  ```
- **Output setelah perbaikan (`go test -count=1 -v ./app/service/`)**:
  ```text
  PASS
  ok  	github.com/Alan-00280/go-pgsql-mhs.git/app/service	0.280s
  ```

### 4. `app/service/review_service_test.go` — `sharedReviews` / `newTestEnv`

- **Output sebelum perbaikan (`go test -shuffle=on -count=5 ./app/service/`)**:
  ```text
  -test.shuffle 1794407931948887500
  --- FAIL: TestListReviews_NewCourseIsEmpty (0.00s)
      review_service_test.go:108: jumlah ulasan = 1, seharusnya 0
  --- FAIL: TestCreateReview_Success (0.00s)
      review_service_test.go:126: jumlah ulasan setelah dibuat = 2, seharusnya 1
  --- FAIL: TestCreateReview_Twice (0.00s)
      review_service_test.go:159: ulasan pertama: status = 409, seharusnya 201
  ...
  FAIL	github.com/Alan-00280/go-pgsql-mhs.git/app/service	0.293s
  ```
- **Output setelah perbaikan (`go test -shuffle=on -count=5 ./app/service/`)**:
  ```text
  ok  	github.com/Alan-00280/go-pgsql-mhs.git/app/service	0.316s
  ```

### 5. `app/service/review_errors_test.go` — `TestTranslateReviewError`

- **Output sebelum perbaikan (`go test -count=1 ./app/service/`)**:
  ```text
  FAIL	github.com/Alan-00280/go-pgsql-mhs.git/app/service [build failed]
  app\service\review_errors_test.go:36:26: undefined: service.translateReviewError
  ```
- **Output setelah perbaikan (`go test -count=1 -v -run TestTranslateReviewError ./app/service/`)**:
  ```text
  --- PASS: TestTranslateReviewError (0.00s)
  PASS
  ok  	github.com/Alan-00280/go-pgsql-mhs.git/app/service	0.224s
  ```

### 6. `app/service/notification_worker_test.go` — `TestHandleSend_PermanentFailures`

- **Output sebelum perbaikan (`go test -count=1 -v -run TestHandleSend_PermanentFailures ./app/service/`)**:
  ```text
  === RUN   TestHandleSend_PermanentFailures
  === RUN   TestHandleSend_PermanentFailures/payload_task_bukan_JSON
      notification_worker_test.go:72: error = skip retry for the task: payload task rusak: invalid character 'b' looking for beginning of value, seharusnya membungkus asynq.SkipRetry
  === RUN   TestHandleSend_PermanentFailures/user_sudah_dihapus
      notification_worker_test.go:72: error = skip retry for the task: user 99 tidak ditemukan, seharusnya membungkus asynq.SkipRetry
  === RUN   TestHandleSend_PermanentFailures/jenis_event_tidak_dikenal
      notification_worker_test.go:72: error = skip retry for the task: jenis event tidak dikenal: "enrollment.unknown", seharusnya membungkus asynq.SkipRetry
  --- FAIL: TestHandleSend_PermanentFailures (0.00s)
  ```
- **Output setelah perbaikan (`go test -count=1 ./...`)**:
  ```text
  ok  	github.com/Alan-00280/go-pgsql-mhs.git/app/service	0.283s
  ok  	github.com/Alan-00280/go-pgsql-mhs.git/helper	0.873s
  ```
