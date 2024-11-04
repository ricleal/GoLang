package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const N = 100000

var postgresURL = os.Getenv("POSTGRES_URL")

var (
	faker     *gofakeit.Faker
	fakerOnce sync.Once
)

func pgTest(ctx context.Context) {
	db, err := pgx.Connect(ctx, postgresURL)
	if err != nil {
		panic(err)
	}
	defer db.Close(ctx)

	err = db.Ping(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected!")
	// List existing tables
	rows, err := db.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema='public'")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	fmt.Println("Tables:")
	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			panic(err)
		}
		fmt.Println("-", tableName)
	}
}

func initFakeIt() *gofakeit.Faker {
	// once initialization
	fakerOnce.Do(func() {
		gofakeit.Seed(0)
		faker = gofakeit.New(0)
	})
	return faker
}

/*
id uuid PRIMARY KEY DEFAULT gen_random_uuid(),

	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	username VARCHAR(255) NOT NULL,
	email VARCHAR(255) NOT NULL,
	name VARCHAR(255) NOT NULL,
	bio TEXT,
	data jsonb
*/
func generateFakeData() []interface{} {
	faker := initFakeIt()
	return []interface{}{
		uuid.New(),
		faker.Username(),
		faker.Email(),
		faker.Name(),
		faker.Sentence(50),
		map[string]interface{}{
			"user": map[string]interface{}{
				"username":   faker.Username(),
				"first_name": faker.FirstName(),
				"last_name":  faker.LastName(),
				"email":      faker.Email(),
				"id":         uuid.New(),
			},
			"resource": map[string]interface{}{
				"id":     uuid.New(),
				"domain": faker.DomainName(),
				"host":   faker.IPv4Address(),
			},
			"timestamp": faker.Date(),
		},
	}
}

func pgPopulate(ctx context.Context) {
	db, err := pgx.Connect(ctx, postgresURL)
	if err != nil {
		panic(err)
	}
	defer db.Close(ctx)

	fakeData := make([][]interface{}, 0, N)

	start := time.Now()
	fmt.Println("Generating fake data...")
	for i := 0; i < N; i++ {
		d := generateFakeData()
		fakeData = append(fakeData, d)
	}
	fmt.Printf("Generated %d rows in %s\n", N, time.Since(start))
	_, err = db.CopyFrom(
		ctx,
		pgx.Identifier{"users"},
		[]string{"id", "username", "email", "name", "bio", "data"},
		pgx.CopyFromRows(fakeData),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Successfully inserted %d rows in %s\n", N, time.Since(start))
}

func main() {
	ctx := context.Background()
	pgTest(ctx)
	pgPopulate(ctx)
}
