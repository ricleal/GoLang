package claims

import "time"

const (
	TopicAuto = "claims-auto"
	TopicHome = "claims-home"
	TopicLife = "claims-life"

	NumProducerWorkers = 5
	BrokerAddr         = "localhost:9092"
)

var Topics = []string{TopicAuto, TopicHome, TopicLife}

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
