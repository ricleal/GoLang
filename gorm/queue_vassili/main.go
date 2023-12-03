package main

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	batchSize       = 10 // Define the size of each batch
	numberOfWorkers = 5  // Total number of workers
)

// Task struct with GORM annotations
type Task struct {
	ID         uint `gorm:"primaryKey"`
	TaskData   []byte
	Status     string    `gorm:"index:idx_task, priority:1"`
	InstanceID string    `gorm:"index:idx_task, priority:2"`
	CreatedAt  time.Time `gorm:"index:idx_task, priority:3"`
	UpdatedAt  time.Time
}

func main() {
	// Initialize database connection with GORM
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	// Automigrate to ensure the schema is up to date
	db.AutoMigrate(&Task{})

	// Recovery logic: Reset tasks that were in_progress to pending
	if err := recoverTasks(db); err != nil {
		log.Fatal("failed to recover tasks:", err)
	}

	// Starting workers
	for i := 0; i < numberOfWorkers; i++ {
		go worker(db, i)
	}

	// Block main from exiting
	select {}
}

func recoverTasks(db *gorm.DB) error {
	// Reset tasks that are stuck in 'in_progress' state
	return db.Model(&Task{}).Where("status = ?", "in_progress").Update("status", "pending").Error
}

func worker(db *gorm.DB, workerID int) {
	for {
		tasks, err := dequeueTasks(db, workerID)
		if err != nil {
			log.Printf("Error dequeueing tasks: %v\n", err)
			time.Sleep(10 * time.Second) // Backoff strategy
			continue
		}

		if len(tasks) == 0 {
			time.Sleep(10 * time.Second) // Backoff strategy
			continue
		}

		for _, task := range tasks {
			processTask(task)
			// Update task status after processing
		}
	}
}

func dequeueTasks(db *gorm.DB, workerID int) ([]Task, error) {
	var tasks []Task
	query := `UPDATE work_queue SET status='in_progress' WHERE status='pending' AND id % ? = ? ORDER BY created_at LIMIT ?`
	result := db.Raw(query, numberOfWorkers, workerID, batchSize).Scan(&tasks)

	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

func processTask(task Task) {
	// Process the task
	// ...
}
