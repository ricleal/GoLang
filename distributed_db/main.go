// Command distributed_db is a hands-on exercise for testing distributed
// database concepts (fault tolerance, tunable consistency, CAP theorem
// trade-offs) against a 3-node Apache Cassandra cluster running in Docker.
//
// See README.md in this directory for the full step-by-step exercise.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gocql/gocql"
)

// ---------------------------------------------------------------------------
// EXPERIMENT KNOB: change this value between runs to compare behavior.
//
//	gocql.Quorum -> requires a majority of replicas (2 of 3) to ack. Tolerates
//	                the loss of a single node, but fails once 2 nodes are down.
//	gocql.One    -> requires only a single replica to ack. Keeps working even
//	                with 2 of 3 nodes down, at the cost of consistency
//	                guarantees (you may read stale or divergent data).
//
// Run the exercise once with gocql.Quorum, then again with gocql.One,
// per Experiment 1 and Experiment 2 in the README.
// ---------------------------------------------------------------------------
const consistencyLevel = gocql.Quorum // <-- CHANGE ME: gocql.Quorum or gocql.One

const (
	keyspace   = "test_ks"
	table      = "users"
	datacenter = "datacenter1"
)

// cassandraHosts are the contact points the driver uses to discover the
// cluster. Each node publishes its 9042 port under a distinct host port
// (cassandra-1 -> 9042, cassandra-2 -> 9043, cassandra-3 -> 9044), so the
// client has an independent path to every node, not just a single one.
// This matters for the failure experiments: if only cassandra-1's port were
// published and you killed cassandra-1 itself, the client would lose all
// connectivity regardless of quorum math, since it would have no other way
// to reach the cluster from outside the Docker network.
var cassandraHosts = []string{"127.0.0.1:9042", "127.0.0.1:9043", "127.0.0.1:9044"}

func main() {
	log.Printf("starting distributed_db exercise with consistency level: %s", consistencyLevel)

	initSession := newSession(gocql.Quorum) // schema setup always needs a quorum-safe level
	if err := initSchema(initSession); err != nil {
		log.Fatalf("failed to initialize schema: %v", err)
	}
	initSession.Close()

	session := newSession(consistencyLevel)
	defer session.Close()

	runLoop(session)
}

// newSession creates a new gocql session configured with the given
// consistency level for all subsequent queries.
func newSession(consistency gocql.Consistency) *gocql.Session {
	cluster := gocql.NewCluster(cassandraHosts...)
	cluster.Consistency = consistency
	cluster.Timeout = 5 * time.Second
	cluster.ConnectTimeout = 10 * time.Second
	cluster.ProtoVersion = 4
	// The nodes advertise their internal Docker network addresses (e.g.
	// 172.x.x.x:9042) via system.peers, which this host-side client cannot
	// reach. Disabling initial host lookup keeps the driver's connection
	// pool restricted to the host:port contact points given above, each of
	// which maps to a real, independently reachable node.
	cluster.DisableInitialHostLookup = true

	var session *gocql.Session
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		session, err = cluster.CreateSession()
		if err == nil {
			return session
		}
		log.Printf("waiting for cassandra to be reachable (attempt %d/30): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("could not connect to cassandra: %v", err)
	return nil
}

// initSchema creates the keyspace and table used by the exercise, if they
// do not already exist.
func initSchema(session *gocql.Session) error {
	createKeyspace := fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s
		WITH replication = {
			'class': 'NetworkTopologyStrategy',
			'%s': 3
		}`, keyspace, datacenter)
	if err := session.Query(createKeyspace).Exec(); err != nil {
		return fmt.Errorf("create keyspace: %w", err)
	}

	createTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			id INT PRIMARY KEY,
			name TEXT,
			updated_at TIMESTAMP
		)`, keyspace, table)
	if err := session.Query(createTable).Exec(); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	log.Printf("schema ready: keyspace=%s table=%s", keyspace, table)
	return nil
}

// runLoop repeatedly writes and reads a row, printing the outcome of each
// operation so failures/successes are easy to observe while nodes are
// stopped and started.
func runLoop(session *gocql.Session) {
	insertQuery := fmt.Sprintf(
		`INSERT INTO %s.%s (id, name, updated_at) VALUES (?, ?, ?)`,
		keyspace, table,
	)
	selectQuery := fmt.Sprintf(
		`SELECT id, name, updated_at FROM %s.%s WHERE id = ?`,
		keyspace, table,
	)

	id := 0
	for {
		id++
		name := fmt.Sprintf("user-%d", id)
		now := time.Now()

		if err := session.Query(insertQuery, id, name, now).Exec(); err != nil {
			fmt.Printf("[WRITE ERROR] %v\n", err)
		} else {
			fmt.Printf("[WRITE SUCCESS] id=%d name=%s\n", id, name)
		}

		var readID int
		var readName string
		var readUpdatedAt time.Time
		if err := session.Query(selectQuery, id).Scan(&readID, &readName, &readUpdatedAt); err != nil {
			fmt.Printf("[READ ERROR] %v\n", err)
		} else {
			fmt.Printf("[READ SUCCESS] id=%d name=%s updated_at=%s\n", readID, readName, readUpdatedAt.Format(time.RFC3339))
		}

		time.Sleep(1 * time.Second)
	}
}
