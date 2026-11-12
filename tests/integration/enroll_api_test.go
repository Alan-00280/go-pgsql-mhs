//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/service"
	"github.com/Alan-00280/go-pgsql-mhs.git/config"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/Alan-00280/go-pgsql-mhs.git/route"
)

// api adalah aplikasi lengkap yang memakai PostgreSQL sungguhan. Bedanya
// dengan production hanya satu: database-nya praktikum_backend_test.
type api struct {
	app *fiber.App
	jwt *helper.JWTManager
}

func newAPI(t *testing.T) api {
	t.Helper()

	raw, err := repository.NewRoleRepository(pool).LoadPermissions(context.Background())
	if err != nil {
		t.Fatalf("memuat permission: %v", err)
	}
	perms := helper.NewPermissionSet(raw)
	jwtManager := helper.NewJWTManager("secret-khusus-test-minimal-32-karakter", "test", time.Minute)
	appValidator := helper.NewValidator(&helper.PasswordCommonSet{})

	courseService := service.NewCourseService(
		repository.NewCourseRepository(pool),
		repository.NewOutboxRepository(pool),
		repository.NewTxManager(pool),
		appValidator,
	)

	app := config.NewApp(slog.New(slog.NewTextHandler(io.Discard, nil)), route.Dependencies{
		Pool:          pool,
		JWT:           jwtManager,
		Permission:    perms,
		CourseHandler: courseService,
	})
	return api{app: app, jwt: jwtManager}
}

// request mengirim permintaan HTTP terautentikasi atas nama student dan
// mengembalikan status code beserta field "code" pada response.
func (a api) request(t *testing.T, student model.User, method, path string, payload any) (int, string) {
	t.Helper()

	token, err := a.jwt.GenerateAccessToken(model.User{ID: student.ID, Username: student.Username, Role: "user"})
	if err != nil {
		t.Errorf("membuat token: %v", err)
		return 0, ""
	}

	var bodyReader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Errorf("marshal body: %v", err)
			return 0, ""
		}
		bodyReader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		// t.Errorf, bukan t.Fatalf: fungsi ini juga dipanggil dari
		// goroutine, dan t.Fatalf hanya boleh dipanggil dari goroutine test.
		t.Errorf("request %s %s: %v", method, path, err)
		return 0, ""
	}
	defer resp.Body.Close()

	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.Code
}

// enroll mengirim POST /courses/:id/enroll atas nama student dan
// mengembalikan status code beserta field "code" pada response.
func (a api) enroll(t *testing.T, student model.User, courseID int) (int, string) {
	t.Helper()
	return a.request(t, student, http.MethodPost, "/api/v1/courses/"+strconv.Itoa(courseID)+"/enroll", nil)
}

// drop mengirim DELETE /courses/:id/enroll atas nama student.
func (a api) drop(t *testing.T, student model.User, courseID int) (int, string) {
	t.Helper()
	return a.request(t, student, http.MethodDelete, "/api/v1/courses/"+strconv.Itoa(courseID)+"/enroll", nil)
}

// swap mengirim POST /enrollments/swap atas nama student.
func (a api) swap(t *testing.T, student model.User, fromCourseID, toCourseID int) (int, string) {
	t.Helper()
	return a.request(t, student, http.MethodPost, "/api/v1/enrollments/swap", model.SwapRequest{
		FromCourseID: fromCourseID,
		ToCourseID:   toCourseID,
	})
}

func countRows(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	return n
}

func TestEnrollEndpoint_WritesOutboxOnlyOnSuccess(t *testing.T) {
	a := newAPI(t)
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran pertama: status = %d, seharusnya 201", status)
	}
	if status, code := a.enroll(t, student, course.ID); status != http.StatusConflict || code != "ALREADY_ENROLLED" {
		t.Fatalf("pendaftaran kedua: dapat %d %s, seharusnya 409 ALREADY_ENROLLED", status, code)
	}

	events := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentCreated)
	if events != 1 {
		t.Errorf("event enrollment.created = %d, seharusnya 1 (pendaftaran yang gagal tidak boleh meninggalkan jejak)", events)
	}
}

// Satu mahasiswa mengirim sepuluh pendaftaran SERENTAK ke sepuluh kelas
// 3 SKS. Batasnya 24 SKS, maka tepat delapan yang boleh berhasil.
// Test ini adalah versi otomatis dari uji beban mode credits pada Modul 9.
func TestEnrollEndpoint_ConcurrentCreditLimit(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)

	const attempts = 10
	courses := make([]model.Course, attempts)
	for i := range courses {
		courses[i] = seedCourse(t, 3, 40)
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = map[int]int{}
		start   = make(chan struct{})
	)
	for _, c := range courses {
		wg.Add(1)
		go func(courseID int) {
			defer wg.Done()
			<-start // tunggu aba-aba agar semua request berangkat bersamaan
			status, _ := a.enroll(t, student, courseID)
			mu.Lock()
			results[status]++
			mu.Unlock()
		}(c.ID)
	}
	close(start) // aba-aba: semua goroutine dilepas pada saat yang sama
	wg.Wait()

	maxCourses := service.MaxCreditsPerSemester / 3
	if results[http.StatusCreated] != maxCourses {
		t.Errorf("berhasil = %d, seharusnya %d (hasil lengkap: %v)", results[http.StatusCreated], maxCourses, results)
	}
	enrolled := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1`, student.ID)
	if enrolled != maxCourses {
		t.Errorf("baris enrollments = %d, seharusnya %d", enrolled, maxCourses)
	}
}

// TestSwapEndpoint_DestinationFull_RollsBack memastikan pindah kelas ke
// kelas tujuan yang penuh ditolak 409 COURSE_FULL tanpa menghapus keikutsertaan
// di kelas asal, tanpa mengubah enrolled_count, dan tanpa menulis event swapped.
func TestSwapEndpoint_DestinationFull_RollsBack(t *testing.T) {
	a := newAPI(t)
	fromCourse := seedCourse(t, 3, 40)
	toCourse := seedCourse(t, 3, 1)
	student := seedStudent(t)
	otherStudent := seedStudent(t)

	if status, code := a.enroll(t, student, fromCourse.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran kelas asal: dapat %d %s, seharusnya 201", status, code)
	}
	if status, code := a.enroll(t, otherStudent, toCourse.ID); status != http.StatusCreated {
		t.Fatalf("pengisian kuota kelas tujuan: dapat %d %s, seharusnya 201", status, code)
	}

	status, code := a.swap(t, student, fromCourse.ID, toCourse.ID)
	if status != http.StatusConflict || code != "COURSE_FULL" {
		t.Fatalf("swap ke kelas penuh: dapat %d %s, seharusnya 409 COURSE_FULL", status, code)
	}

	inFrom := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student.ID, fromCourse.ID)
	if inFrom != 1 {
		t.Errorf("keikutsertaan di kelas asal = %d, seharusnya tetap 1", inFrom)
	}
	inTo := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student.ID, toCourse.ID)
	if inTo != 0 {
		t.Errorf("keikutsertaan di kelas tujuan = %d, seharusnya 0", inTo)
	}

	fromEnrolled := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, fromCourse.ID)
	if fromEnrolled != 1 {
		t.Errorf("enrolled_count kelas asal = %d, seharusnya tetap 1", fromEnrolled)
	}
	toEnrolled := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, toCourse.ID)
	if toEnrolled != 1 {
		t.Errorf("enrolled_count kelas tujuan = %d, seharusnya tetap 1", toEnrolled)
	}

	swappedEvents := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentSwapped)
	if swappedEvents != 0 {
		t.Errorf("event enrollment.swapped = %d, seharusnya 0", swappedEvents)
	}
}

// TestDropEndpoint_Success_UpdatesCountAndOutbox memastikan DELETE /courses/:id/enroll
// mengembalikan 204, menghapus baris enrollment, mengurangi enrolled_count satu,
// dan mencatat tepat satu event enrollment.dropped di outbox.
func TestDropEndpoint_Success_UpdatesCountAndOutbox(t *testing.T) {
	a := newAPI(t)
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	if status, code := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran awal: dapat %d %s, seharusnya 201", status, code)
	}

	status, code := a.drop(t, student, course.ID)
	if status != http.StatusNoContent {
		t.Fatalf("drop kelas: dapat %d %s, seharusnya 204", status, code)
	}

	remaining := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student.ID, course.ID)
	if remaining != 0 {
		t.Errorf("sisa baris enrollments = %d, seharusnya 0", remaining)
	}

	enrolledCount := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, course.ID)
	if enrolledCount != 0 {
		t.Errorf("enrolled_count = %d, seharusnya 0", enrolledCount)
	}

	droppedEvents := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND (payload->>'course_id')::INT = $2 AND event_type = $3`,
		student.ID, course.ID, model.EventEnrollmentDropped)
	if droppedEvents != 1 {
		t.Errorf("event enrollment.dropped = %d, seharusnya 1", droppedEvents)
	}
}
