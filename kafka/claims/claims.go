// Package claims defines the shared data types and Kafka topic constants used by
// the producer, consumer, and inspector executables.
package claims

import "time"

const (
	TopicAuto = "claims-auto" // topic for automobile insurance claims
	TopicHome = "claims-home" // topic for home insurance claims
	TopicLife = "claims-life" // topic for life insurance claims

	// NumProducerWorkers is the default number of concurrent producer goroutines.
	NumProducerWorkers = 5
	// BrokerAddr is the default Kafka broker address for local development.
	BrokerAddr = "localhost:9092"
)

//nolint:gochecknoglobals // used as a constant list across packages
var Topics = []string{
	TopicAuto,
	TopicHome,
	TopicLife,
}

// Claim represents an insurance claim.
type Claim struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	ClaimType  string    `json:"claim_type"`
	Amount     float64   `json:"amount"`
	Timestamp  time.Time `json:"timestamp"`
}

// TopicFor returns the Kafka topic for the given claim type.
func TopicFor(claimType string) string {
	switch claimType {
	case "auto":
		return TopicAuto
	case "home":
		return TopicHome
	case "life":
		return TopicLife
	default:
		return TopicAuto
	}
}
