// Command datagen generates continuous CRUD traffic for the inventory schema
// (customers, products, products_on_hand, orders) so the CDC pipeline always
// has fresh change events. It is the Go port of datagen/gen.py.
package main

import (
	"context"
	"flag"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/jackc/pgx/v5"
)

const defaultDSN = "postgresql://postgres:postgres@postgres-source:5432/postgres"

func main() {
	once := flag.Bool("once", false, "run a single change and exit")
	flag.Parse()

	dsn := os.Getenv("DATAGEN_DSN")
	if dsn == "" {
		dsn = defaultDSN
	}
	interval := envFloat("DATAGEN_INTERVAL_SECONDS", 2)
	seedFaker()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("datagen up (interval %.0fs, once=%v)", interval, *once)
	for {
		conn, err := pgx.Connect(ctx, dsn)
		if err != nil {
			log.Printf("db unavailable, retrying in 5s: %v", err)
			if *once {
				log.Fatal(err)
			}
			time.Sleep(5 * time.Second)
			continue
		}
		err = run(ctx, conn, *once, interval)
		conn.Close(ctx)
		if err != nil {
			log.Printf("db unavailable, retrying in 5s: %v", err)
			if *once {
				log.Fatal(err)
			}
			time.Sleep(5 * time.Second)
			continue
		}
		if *once {
			return
		}
	}
}

// run executes the weighted op pool until ctx is done (or once after one op).
func run(ctx context.Context, conn *pgx.Conn, once bool, interval float64) error {
	for {
		op := pool[gofakeit.IntRange(0, len(pool)-1)]
		if err := op(ctx, conn); err != nil {
			return err
		}
		if once {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(interval * float64(time.Second))):
		}
	}
}

// ---- ops ----

func insertCustomer(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx,
		"INSERT INTO inventory.customers (first_name, last_name, email) VALUES ($1, $2, $3)",
		gofakeit.FirstName(), gofakeit.LastName(), gofakeit.Email())
	return err
}

func updateCustomer(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx,
		"UPDATE inventory.customers SET email = $1 WHERE id = "+
			"(SELECT id FROM inventory.customers ORDER BY random() LIMIT 1)",
		gofakeit.Email())
	return err
}

func deleteCustomer(ctx context.Context, conn *pgx.Conn) error {
	// only generated customers never referenced by orders — keeps FK intact
	_, err := conn.Exec(ctx,
		"DELETE FROM inventory.customers WHERE id = ("+
			" SELECT id FROM inventory.customers"+
			" WHERE id > 1004 AND id NOT IN (SELECT purchaser FROM inventory.orders)"+
			" ORDER BY random() LIMIT 1)")
	return err
}

func insertOrder(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx,
		"INSERT INTO inventory.orders (order_date, purchaser, quantity, product_id) "+
			"VALUES (CURRENT_DATE, $1, $2, $3)",
		gofakeit.IntRange(1001, 1004), gofakeit.IntRange(1, 5), gofakeit.IntRange(101, 109))
	return err
}

func updateStock(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx,
		"UPDATE inventory.products_on_hand SET quantity = $1 WHERE product_id = $2",
		gofakeit.IntRange(0, 100), gofakeit.IntRange(101, 109))
	return err
}

func updateProduct(ctx context.Context, conn *pgx.Conn) error {
	// keeps the products table's iceberg snapshots advancing so its
	// freshness DQ check stays meaningful
	weight := math.Round(gofakeit.Float64Range(0.1, 10.1)*100) / 100
	_, err := conn.Exec(ctx,
		"UPDATE inventory.products SET weight = round($1::numeric, 2) WHERE id = $2",
		weight, gofakeit.IntRange(101, 109))
	return err
}

// pool is the weighted op list: insert_customer x4, update_customer x3,
// delete_customer x1, insert_order x1, update_stock x1, update_product x1.
var pool = []func(context.Context, *pgx.Conn) error{
	insertCustomer, insertCustomer, insertCustomer, insertCustomer,
	updateCustomer, updateCustomer, updateCustomer,
	deleteCustomer,
	insertOrder,
	updateStock,
	updateProduct,
}

func envFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// seedFaker makes runs reproducible via DATAGEN_SEED (an integer), defaulting
// to a time-based seed so each run looks different.
func seedFaker() {
	if s := os.Getenv("DATAGEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			_ = gofakeit.Seed(n)
			return
		}
		log.Printf("DATAGEN_SEED=%q is not an integer, ignoring", s)
	}
	_ = gofakeit.Seed(time.Now().UnixNano())
}
