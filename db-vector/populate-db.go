package main

import (
	"context"
	"fmt"
	"math/rand"
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
	faker                 *gofakeit.Faker
	fakerOnce             sync.Once
	possibleOrganizations = []uuid.UUID{
		uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(),
	}
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

func generateFakeData() []interface{} {
	faker := initFakeIt()
	return []interface{}{
		uuid.New(),
		possibleOrganizations[rand.Intn(len(possibleOrganizations))], //nolint:G404
		faker.FirstName(),
		faker.LastName(),
		faker.Email(),
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
		[]string{"id", "organization_id", "firstname", "lastname", "email", "data"},
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
