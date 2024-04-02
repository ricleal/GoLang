package main

// Test to make sre DB connection pool is working as expected
// if more connections are opened than the pool size, the query should wait until a connection is available

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupDB() (*gorm.DB, error) {
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

func main() {
	db, err := setupDB()
	if err != nil {
		log.Fatal(err)
	}
	stdDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer stdDB.Close()
	stdDB.SetMaxOpenConns(2)
	stdDB.SetMaxIdleConns(2)

	mux := http.NewServeMux()
	mux.HandleFunc("/", Handler(db, stdDB))

	// Wrap the servemux with the limit middleware.
	log.Print("Listening on :8887...")
	http.ListenAndServe("localhost:8887", mux)
}

func Handler(db *gorm.DB, stdDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tx := db.Begin()
		if tx.Error != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		if err := tx.Exec("SELECT 1").Error; err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// random sleep
		s := rand.Intn(10)
		log.Printf("Sleeping for %d seconds", s)
		time.Sleep(time.Duration(s) * time.Second)

		// convert dbstats to json
		stats := stdDB.Stats()
		log.Printf("MaxOpenConnections=%d OpenConnections=%d InUse=%d Idle=%d WaitCount=%d WaitDuration=%s MaxIdleClosed=%d MaxLifetimeClosed=%d",
			stats.MaxOpenConnections, stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount, stats.WaitDuration, stats.MaxIdleClosed, stats.MaxLifetimeClosed)

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write([]byte("OK"))
	}
}
