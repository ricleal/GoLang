# Prompt

In this folder I would like to do an experiment with Kafka. I want to see how it works, how to set it up, and how to use it in a simple application. I will be using Docker to set up Kafka (Apache Kafka Without Its ZooKeeper) , and then I will write a simple producer and consumer in Go to test it out.

The producer should have 5 go routines and produce `claims` until we press Ctrl+C. The `claims` should have the strucure:

```go
type Claim struct {
    ID        string `json:"id"` // unique identifier for the claim
    CustomerID string `json:"customer_id"` // identifier for the customer making the claim (uniformly distributed between 1 and 1000)
    ClaimType string `json:"claim_type"` // type of the claim (uniformly distributed between "auto", "home", "life")
    Amount    float64 `json:"amount"` // amount of the claim (uniformly distributed between 100 and 10000)
    Timestamp time.Time `json:"timestamp"` // time when the claim was created
}
```

The Kafka should have 5 partitions, and 3 topics: `claims_auto`, `claims_home`, and `claims_life`. The producer should produce claims to the appropriate topic based on the `ClaimType`. To avoid hot partitioning, the producer should use the `CustomerID` as the key when producing messages to Kafka. This way, claims from the same customer will go to the same partition, but claims from different customers will be distributed across partitions.

I want to to test fan-out by having 4 consumers, one of them reading from `claims_auto`, one reading from `claims_home`, one reading from `claims_life`, and one reading from all three topics. The consumer that reads from all three topics should be able to process claims of all types, while the other consumers should only process claims of their respective types.

I will also want to test the performance of the producer and consumer, and see how they handle a high volume of messages. I will use a simple in-memory data structure to store the claims that have been processed by the consumers, and I will print out some statistics about the number of claims processed, the average amount of the claims, and the distribution of claim types.

I will also want to test the fault tolerance of the system by simulating failures in the producer and consumer. For example, I can simulate a failure in the producer by stopping it while it is producing messages, and then restarting it to see if it can resume producing messages without losing any data. Similarly, I can simulate a failure in the consumer by stopping it while it is consuming messages, and then restarting it to see if it can resume consuming messages without losing any data.
Overall, this experiment will help me understand how Kafka works, how to set it up, and how to use it in a simple application. It will also help me understand the concepts of partitions, topics, producers, consumers, and fault tolerance in Kafka.