package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/Alan-00280/go-pgsql-mhs.git/config"
	"github.com/Alan-00280/go-pgsql-mhs.git/database"
)

func main() {
	config.LoadEnv()

	outboxID := flag.Int64("outbox", 0, "ID baris outbox yang akan dikirim ulang")
	copies := flag.Int("copies", 5, "Jumlah salinan task yang dikirim ke Redis")
	flag.Parse()

	if *outboxID <= 0 {
		log.Fatal("flag -outbox wajib diisi dengan ID valid (> 0)")
	}
	if *copies <= 0 {
		log.Fatal("flag -copies harus minimal 1")
	}

	ctx := context.Background()

	// 1. Ambil data event dari outbox di PostgreSQL
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("gagal koneksi ke database: %v", err)
	}
	defer pool.Close()

	var event model.OutboxEvent
	err = pool.QueryRow(ctx, `
		SELECT id, event_type, payload, created_at
		FROM outbox
		WHERE id = $1
	`, *outboxID).Scan(&event.ID, &event.EventType, &event.Payload, &event.CreatedAt)
	if err != nil {
		log.Fatalf("gagal mengambil outbox ID %d: %v", *outboxID, err)
	}

	// 2. Siapkan task payload
	job := model.NotificationJob{
		OutboxID:  event.ID,
		EventType: event.EventType,
		Payload:   event.Payload,
	}

	payload, err := json.Marshal(job)
	if err != nil {
		log.Fatalf("gagal marshal job: %v", err)
	}

	// 3. Hubungkan ke Redis
	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	// 4. Enqueue sejumlah copies tanpa batasan TaskID bentrok
	fmt.Printf("Mengirim %d salinan task untuk outbox ID %d ke queue %q...\n", *copies, event.ID, model.QueueNotifications)

	for i := 1; i <= *copies; i++ {
		task := asynq.NewTask(
			model.TaskSendNotification,
			payload,
			asynq.Queue(model.QueueNotifications),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
			asynq.Retention(24*time.Hour),
		)

		info, err := client.EnqueueContext(ctx, task)
		if err != nil {
			log.Fatalf("gagal enqueue copy ke-%d: %v", i, err)
		}
		fmt.Printf("  [%d/%d] Enqueued task ID: %s\n", i, *copies, info.ID)
	}

	fmt.Println("Selesai. Seluruh salinan berhasil dimasukkan ke Redis.")
}
