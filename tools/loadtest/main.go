package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/config"
	"github.com/Alan-00280/go-pgsql-mhs.git/database"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// fire menjalankan n pekerjaan secara serentak dan menghitung hasilnya.
//
// Setiap goroutine menunggu di depan "gerbang" start yang sama. Gerbang
// baru dibuka setelah SELURUH goroutine siap, sehingga request benar-benar
// berangkat hampir bersamaan — bukan bergiliran sesuai urutan dibuatnya.
func fire(n int, job func(i int) string) map[string]int {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)

		go func(i int, out chan<- string) {
			defer wg.Done()
			<-start
			out <- job(i)
		}(i, results)
	}

	close(start) // buka gerbang
	wg.Wait()    // tunggu SEMUA goroutine selesai
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}

	return tally
}

func main() {
	// Muat berkas .env baik dari root maupun direktori saat ini
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	mode := flag.String("mode", "supervisor", "Mode pengujian: supervisor, supervisor-switch, crowd, credits, swap, reset")
	baseURL := flag.String("url", "http://localhost:3000", "Base URL server aplikasi")
	supervisorID := flag.Int("supervisor", 9, "ID supervisor target (bawaan 9, kuota 3)")
	concurrency := flag.Int("concurrency", 30, "Jumlah request serentak")
	flag.Parse()

	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < 32 {
		log.Fatalf("JWT_SECRET di .env belum diatur atau kurang dari 32 karakter")
	}

	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "backend-practice"),
		15*time.Minute,
	)

	// Hubungkan ke database untuk seed dan verifikasi konsistensi
	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("Gagal terhubung ke PostgreSQL: %v", err)
	}
	defer pool.Close()

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
	}

	switch *mode {
	case "supervisor", "supervisor-assign", "supervisor-crowd":
		runSupervisorCrowdTest(ctx, pool, client, jwtManager, *baseURL, *supervisorID, *concurrency)
	case "supervisor-switch":
		runSupervisorSwitchTest(ctx, pool, client, jwtManager, *baseURL)
	case "crowd":
		runCourseCrowdTest(ctx, pool, client, jwtManager, *baseURL)
	case "credits":
		runCourseCreditsTest(ctx, pool, client, jwtManager, *baseURL)
	case "swap":
		runCourseSwapTest(ctx, pool, client, jwtManager, *baseURL)
	case "reset":
		resetSupervisorData(ctx, pool)
		fmt.Println("Data supervisions berhasil di-reset.")
	default:
		fmt.Printf("Mode %q tidak dikenali. Pilihan mode: supervisor, supervisor-switch, crowd, credits, swap, reset\n", *mode)
		os.Exit(1)
	}
}

// ============================================================================
// UJI BEBAN SUPERVISOR (D.3)
// ============================================================================

func runSupervisorCrowdTest(
	ctx context.Context,
	pool *pgxpool.Pool,
	client *http.Client,
	jwtMgr *helper.JWTManager,
	baseURL string,
	supervisorID int,
	n int,
) {
	// Pastikan minimal n mahasiswa tersedia di tabel students
	studentIDs := ensureTestStudents(ctx, pool, n)

	// Dapatkan info supervisor sebelum uji
	var supName string
	var quota, supervisedCount int
	err := pool.QueryRow(ctx, "SELECT name, quota, supervised_count FROM supervisors WHERE id = $1", supervisorID).
		Scan(&supName, &quota, &supervisedCount)
	if err != nil {
		log.Fatalf("Supervisor ID %d tidak ditemukan di database: %v", supervisorID, err)
	}

	// Cetak token admin (karena endpoint POST /supervisors/:id/assign memerlukan permission supervisions:manage)
	adminToken, err := jwtMgr.GenerateAccessToken(model.User{
		ID:       5,
		Username: "admin_test",
		Role:     "admin",
	})
	if err != nil {
		log.Fatalf("Gagal membuat admin token: %v", err)
	}

	fmt.Printf("%d mahasiswa serentak mendaftar ke supervisor ID %d (%s, kuota %d)...\n", n, supervisorID, supName, quota)

	assignURL := fmt.Sprintf("%s/api/v1/supervisors/%d/assign", strings.TrimRight(baseURL, "/"), supervisorID)

	t0 := time.Now()
	tally := fire(n, func(i int) string {
		studentID := studentIDs[i]
		payload := map[string]int{"student_id": studentID}
		bodyBytes, _ := json.Marshal(payload)

		req, reqErr := http.NewRequestWithContext(context.Background(), "POST", assignURL, bytes.NewReader(bodyBytes))
		if reqErr != nil {
			return fmt.Sprintf("ERR %v", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, doErr := client.Do(req)
		if doErr != nil {
			return fmt.Sprintf("ERR %v", doErr)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusCreated {
			return "201 OK"
		}

		// Parse error code dari format JSON helper {"success":false,"code":"...","message":"..."}
		var errResp struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &errResp)

		if errResp.Code != "" {
			return fmt.Sprintf("%d %s", resp.StatusCode, errResp.Code)
		}
		return fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	})
	totalTime := time.Since(t0)

	// Cetak hasil request
	printTally(tally)
	fmt.Printf("waktu total : %dms\n", totalTime.Milliseconds())

	// Verifikasi konsistensi database
	var endSupervisedCount int
	var rowCount int
	_ = pool.QueryRow(ctx, "SELECT supervised_count FROM supervisors WHERE id = $1", supervisorID).Scan(&endSupervisedCount)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM supervisions WHERE supervisor_id = $1", supervisorID).Scan(&rowCount)

	status := "KONSISTEN"
	if endSupervisedCount != rowCount || endSupervisedCount > quota {
		status = "TIDAK KONSISTEN"
	}

	fmt.Printf("Supervisor ID %d: kuota=%d supervised_count=%d baris supervisions=%d -> %s\n",
		supervisorID, quota, endSupervisedCount, rowCount, status)
}

func runSupervisorSwitchTest(
	ctx context.Context,
	pool *pgxpool.Pool,
	client *http.Client,
	jwtMgr *helper.JWTManager,
	baseURL string,
) {
	pairs := 5
	n := pairs * 2
	studentIDs := ensureTestStudents(ctx, pool, n)

	supA := 1 // Dr. Budi Santoso (kuota 10)
	supB := 6 // Dr. Rina Kusumawati (kuota 10)

	// Pastikan 5 mahasiswa pertama terdaftar di supA, 5 mahasiswa berikutnya di supB
	resetSupervisorData(ctx, pool)
	for i := 0; i < pairs; i++ {
		_, _ = pool.Exec(ctx, "INSERT INTO supervisions (supervisor_id, student_id) VALUES ($1, $2)", supA, studentIDs[i])
	}
	for i := pairs; i < n; i++ {
		_, _ = pool.Exec(ctx, "INSERT INTO supervisions (supervisor_id, student_id) VALUES ($1, $2)", supB, studentIDs[i])
	}
	_, _ = pool.Exec(ctx, "UPDATE supervisors SET supervised_count = (SELECT COUNT(*) FROM supervisions WHERE supervisor_id = supervisors.id)")

	// Buat token admin atau user
	token, _ := jwtMgr.GenerateAccessToken(model.User{
		ID:       5,
		Username: "admin_test",
		Role:     "admin",
	})

	switchURL := fmt.Sprintf("%s/api/v1/supervisors/switch", strings.TrimRight(baseURL, "/"))

	fmt.Printf("%d pasang mahasiswa pindah supervisor berlawanan arah secara serentak (ID %d <-> ID %d)...\n", pairs, supA, supB)

	t0 := time.Now()
	tally := fire(n, func(i int) string {
		fromID, toID := supA, supB
		if i >= pairs {
			fromID, toID = supB, supA
		}

		payload := map[string]int{
			"student_id":         studentIDs[i],
			"from_supervisor_id": fromID,
			"to_supervisor_id":   toID,
		}
		bodyBytes, _ := json.Marshal(payload)

		req, err := http.NewRequestWithContext(context.Background(), "POST", switchURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Sprintf("ERR %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Sprintf("ERR %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			return "200 OK"
		}

		var errResp struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Code != "" {
			return fmt.Sprintf("%d %s", resp.StatusCode, errResp.Code)
		}
		return fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	})
	totalTime := time.Since(t0)

	printTally(tally)
	fmt.Printf("waktu total : %dms\n", totalTime.Milliseconds())

	// Verifikasi konsistensi
	var countA, countB, rowsA, rowsB int
	_ = pool.QueryRow(ctx, "SELECT supervised_count FROM supervisors WHERE id = $1", supA).Scan(&countA)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM supervisions WHERE supervisor_id = $1", supA).Scan(&rowsA)
	_ = pool.QueryRow(ctx, "SELECT supervised_count FROM supervisors WHERE id = $1", supB).Scan(&countB)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM supervisions WHERE supervisor_id = $1", supB).Scan(&rowsB)

	fmt.Printf("Supervisor ID %d: supervised_count=%d baris supervisions=%d -> %s\n",
		supA, countA, rowsA, konsistenStr(countA == rowsA))
	fmt.Printf("Supervisor ID %d: supervised_count=%d baris supervisions=%d -> %s\n",
		supB, countB, rowsB, konsistenStr(countB == rowsB))
}

// ============================================================================
// UJI BEBAN KELAS & KRS (MODUL 9 BAGIAN C)
// ============================================================================

func runCourseCrowdTest(ctx context.Context, pool *pgxpool.Pool, client *http.Client, jwtMgr *helper.JWTManager, baseURL string) {
	n := 60
	courseCode := "IF399"
	var courseID, quota int
	err := pool.QueryRow(ctx, "SELECT id, quota FROM courses WHERE code = $1", courseCode).Scan(&courseID, &quota)
	if err != nil {
		log.Fatalf("Course %s tidak ditemukan: %v", courseCode, err)
	}

	tokens := make([]string, n)
	for i := 0; i < n; i++ {
		username := fmt.Sprintf("mhs%03d", i+1)
		var uid int
		_ = pool.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&uid)
		if uid == 0 {
			uid = 100 + i
		}
		tokens[i], _ = jwtMgr.GenerateAccessToken(model.User{
			ID:       uid,
			Username: username,
			Role:     "user",
		})
	}

	fmt.Printf("%d mahasiswa berebut kelas %s (kuota %d) secara serentak...\n", n, courseCode, quota)

	enrollURL := fmt.Sprintf("%s/api/v1/courses/%d/enroll", strings.TrimRight(baseURL, "/"), courseID)

	t0 := time.Now()
	tally := fire(n, func(i int) string {
		req, _ := http.NewRequestWithContext(context.Background(), "POST", enrollURL, nil)
		req.Header.Set("Authorization", "Bearer "+tokens[i])

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Sprintf("ERR %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusCreated {
			return "201 OK"
		}
		var errResp struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Code != "" {
			return fmt.Sprintf("%d %s", resp.StatusCode, errResp.Code)
		}
		return fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	})
	totalTime := time.Since(t0)

	printTally(tally)
	fmt.Printf("waktu total : %dms\n", totalTime.Milliseconds())

	var enrolledCount, rowCount int
	_ = pool.QueryRow(ctx, "SELECT enrolled_count FROM courses WHERE id = $1", courseID).Scan(&enrolledCount)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM enrollments WHERE course_id = $1", courseID).Scan(&rowCount)

	status := "KONSISTEN"
	if enrolledCount != rowCount || enrolledCount > quota {
		status = "TIDAK KONSISTEN"
	}
	fmt.Printf("%s kuota=%d enrolled_count=%d baris enrollments=%d -> %s\n",
		courseCode, quota, enrolledCount, rowCount, status)
}

func runCourseCreditsTest(ctx context.Context, pool *pgxpool.Pool, client *http.Client, jwtMgr *helper.JWTManager, baseURL string) {
	// 10 mahasiswa mendaftar ke 12 kelas sekaligus
	fmt.Println("Menjalankan uji batas SKS (10 mahasiswa x 12 kelas serentak)...")
	// Ambil 12 course ID
	rows, err := pool.Query(ctx, "SELECT id FROM courses ORDER BY id LIMIT 12")
	if err != nil {
		log.Fatalf("Gagal mengambil courses: %v", err)
	}
	defer rows.Close()

	var courseIDs []int
	for rows.Next() {
		var cid int
		_ = rows.Scan(&cid)
		courseIDs = append(courseIDs, cid)
	}

	numStudents := 10
	totalRequests := numStudents * len(courseIDs)

	tokens := make([]string, numStudents)
	for i := 0; i < numStudents; i++ {
		username := fmt.Sprintf("mhs%03d", i+1)
		var uid int
		_ = pool.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&uid)
		if uid == 0 {
			uid = 100 + i
		}
		tokens[i], _ = jwtMgr.GenerateAccessToken(model.User{ID: uid, Username: username, Role: "user"})
	}

	t0 := time.Now()
	tally := fire(totalRequests, func(i int) string {
		studentIdx := i / len(courseIDs)
		courseIdx := i % len(courseIDs)

		cid := courseIDs[courseIdx]
		url := fmt.Sprintf("%s/api/v1/courses/%d/enroll", strings.TrimRight(baseURL, "/"), cid)

		req, _ := http.NewRequestWithContext(context.Background(), "POST", url, nil)
		req.Header.Set("Authorization", "Bearer "+tokens[studentIdx])

		resp, doErr := client.Do(req)
		if doErr != nil {
			return fmt.Sprintf("ERR %v", doErr)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusCreated {
			return "201 OK"
		}
		var errResp struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Code != "" {
			return fmt.Sprintf("%d %s", resp.StatusCode, errResp.Code)
		}
		return fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	})
	totalTime := time.Since(t0)

	printTally(tally)
	fmt.Printf("waktu total : %dms\n", totalTime.Milliseconds())

	// Verifikasi SKS setiap mahasiswa
	allValid := true
	for i := 0; i < numStudents; i++ {
		username := fmt.Sprintf("mhs%03d", i+1)
		var totalCredits int
		_ = pool.QueryRow(ctx, `SELECT COALESCE(SUM(c.credits), 0)
			FROM enrollments e
			JOIN courses c ON c.id = e.course_id
			JOIN users u ON u.id = e.user_id
			WHERE u.username = $1`, username).Scan(&totalCredits)
		if totalCredits > 24 {
			allValid = false
			fmt.Printf("PERINGATAN: %s memiliki %d SKS (> 24 SKS)!\n", username, totalCredits)
		}
	}
	if allValid {
		fmt.Println("seluruh 10 mahasiswa <= 24 SKS -> KONSISTEN")
	} else {
		fmt.Println("Ditemukan mahasiswa melebihi 24 SKS -> TIDAK KONSISTEN")
	}
}

func runCourseSwapTest(ctx context.Context, pool *pgxpool.Pool, client *http.Client, jwtMgr *helper.JWTManager, baseURL string) {
	pairs := 20
	n := pairs * 2

	var id101, id102 int
	_ = pool.QueryRow(ctx, "SELECT id FROM courses WHERE code = 'IF101'").Scan(&id101)
	_ = pool.QueryRow(ctx, "SELECT id FROM courses WHERE code = 'IF102'").Scan(&id102)

	// Pastikan 20 mahasiswa terdaftar di IF101, 20 di IF102
	tokens := make([]string, n)
	for i := 0; i < n; i++ {
		username := fmt.Sprintf("mhs%03d", i+1)
		var uid int
		_ = pool.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&uid)
		if uid == 0 {
			uid = 100 + i
		}
		tokens[i], _ = jwtMgr.GenerateAccessToken(model.User{ID: uid, Username: username, Role: "user"})

		// Bersihkan dan masukkan
		_, _ = pool.Exec(ctx, "DELETE FROM enrollments WHERE user_id = $1", uid)
		if i < pairs {
			_, _ = pool.Exec(ctx, "INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2)", id101, uid)
		} else {
			_, _ = pool.Exec(ctx, "INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2)", id102, uid)
		}
	}
	_, _ = pool.Exec(ctx, "UPDATE courses SET enrolled_count = (SELECT COUNT(*) FROM enrollments WHERE course_id = courses.id)")

	fmt.Printf("%d pasang mahasiswa pindah kelas berlawanan arah secara serentak...\n", pairs)
	swapURL := fmt.Sprintf("%s/api/v1/enrollments/swap", strings.TrimRight(baseURL, "/"))

	t0 := time.Now()
	tally := fire(n, func(i int) string {
		fromID, toID := id101, id102
		if i >= pairs {
			fromID, toID = id102, id101
		}
		payload := map[string]int{"from_course_id": fromID, "to_course_id": toID}
		bodyBytes, _ := json.Marshal(payload)

		req, _ := http.NewRequestWithContext(context.Background(), "POST", swapURL, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokens[i])

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Sprintf("ERR %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusOK {
			return "200 OK"
		}
		var errResp struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Code != "" {
			return fmt.Sprintf("%d %s", resp.StatusCode, errResp.Code)
		}
		return fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	})
	totalTime := time.Since(t0)

	printTally(tally)
	fmt.Printf("waktu total : %dms\n", totalTime.Milliseconds())

	var count101, rows101, count102, rows102 int
	_ = pool.QueryRow(ctx, "SELECT enrolled_count FROM courses WHERE id = $1", id101).Scan(&count101)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM enrollments WHERE course_id = $1", id101).Scan(&rows101)
	_ = pool.QueryRow(ctx, "SELECT enrolled_count FROM courses WHERE id = $1", id102).Scan(&count102)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM enrollments WHERE course_id = $1", id102).Scan(&rows102)

	fmt.Printf("IF101: enrolled_count=%d baris enrollments=%d -> %s\n", count101, rows101, konsistenStr(count101 == rows101))
	fmt.Printf("IF102: enrolled_count=%d baris enrollments=%d -> %s\n", count102, rows102, konsistenStr(count102 == rows102))
}

// ============================================================================
// HELPER & UTILITIES
// ============================================================================

func ensureTestStudents(ctx context.Context, pool *pgxpool.Pool, count int) []int {
	// Ambil students yang ada
	rows, err := pool.Query(ctx, "SELECT id FROM students ORDER BY id LIMIT $1", count)
	if err != nil {
		log.Fatalf("Gagal mengambil data students: %v", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}

	// Jika kurang dari count, buat test students
	if len(ids) < count {
		need := count - len(ids)
		for i := 1; i <= need+10; i++ {
			nim := fmt.Sprintf("LOAD%05d", i)
			name := fmt.Sprintf("LoadTest Student %d", i)
			var insertedID int
			err := pool.QueryRow(ctx, `INSERT INTO students (nim, name, grade, is_active)
				VALUES ($1, $2, 3.50, true)
				ON CONFLICT (nim) DO UPDATE SET is_active = true
				RETURNING id`, nim, name).Scan(&insertedID)
			if err == nil {
				ids = append(ids, insertedID)
			}
			if len(ids) >= count {
				break
			}
		}
	}

	if len(ids) < count {
		log.Fatalf("Tidak cukup mahasiswa di database (tersedia %d, dibutuhkan %d)", len(ids), count)
	}

	return ids[:count]
}

func resetSupervisorData(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, "DELETE FROM supervisions")
	_, _ = pool.Exec(ctx, "UPDATE supervisors SET supervised_count = (SELECT COUNT(*) FROM supervisions WHERE supervisor_id = supervisors.id)")
}

func printTally(tally map[string]int) {
	fmt.Println("hasil request:")
	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-24s %d\n", k, tally[k])
	}
}

func konsistenStr(ok bool) string {
	if ok {
		return "KONSISTEN"
	}
	return "TIDAK KONSISTEN"
}
