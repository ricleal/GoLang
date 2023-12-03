package main

import (
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func main() {

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	l := gormLogger.New(
		log.New(os.Stdout, "\n", log.LstdFlags),
		gormLogger.Config{
			SlowThreshold: time.Microsecond, // Slow SQL threshold :)
			LogLevel:      gormLogger.Warn,  // Log level
		},
	)

	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: l,
	})
	if err != nil {
		panic(err)
	}

	type Result struct {
		ID       int
		Name     string
		UniqueID int
	}

	var result []Result
	r := db.Raw("SELECT id, name, unique_id FROM t1").Scan(&result)
	if r.Error != nil {
		panic(r.Error)
	}

	for i, v := range result {
		logger.Debug("result", zap.Int("index", i), zap.Any("value", v))
	}
}
