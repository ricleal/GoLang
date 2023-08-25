package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

type EventBus struct {
	subscribers map[string][]chan interface{}
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		// map of topic -> list of channels (each channel is a subscriber)
		subscribers: make(map[string][]chan interface{}),
	}
}

func (eb *EventBus) Subscribe(topic string) chan interface{} {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan interface{})
	eb.subscribers[topic] = append(eb.subscribers[topic], ch)
	return ch
}

func (eb *EventBus) Publish(topic string, data interface{}) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subscribers, found := eb.subscribers[topic]
	if !found {
		log.Println("No subscribers for event type:", topic)
		return
	}

	for _, ch := range subscribers {
		go func(ch chan interface{}) {
			ch <- data
		}(ch)
	}
}

func main() {
	eventBus := NewEventBus()

	log.Printf("-- Serial subscriptions --")
	// Subscribe to "user_created" events
	userCreatedCh1 := eventBus.Subscribe("user_created")
	userCreatedCh2 := eventBus.Subscribe("user_created")

	// Publish a "user_created" event
	eventBus.Publish("user_created", `User created: {"id": 1, "name": "John"}`)

	// Read from the subscription channel
	log.Println("Received 1:", <-userCreatedCh1)
	log.Println("Received 2:", <-userCreatedCh2)

	log.Printf("-- Parallel subscriptions --")
	// Parallel subscriptions
	// simulate topics and subscribers in a multi-threaded environment
	topics := []string{"user_created", "user_updated", "user_deleted"}
	// create 2 subscribers per topic
	subscribers := make([]chan interface{}, len(topics)*2)
	for i := 0; i < len(topics)*2; i++ {
		subscribers[i] = eventBus.Subscribe(topics[i%len(topics)])
	}
	wg := sync.WaitGroup{}
	wg.Add(len(topics))
	for i := 0; i < len(topics); i++ {
		go func(topic string) {
			defer wg.Done()
			// simulate some random delay
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
			eventBus.Publish(topic, fmt.Sprintf("Event {.....} for topic: %s", topic))
		}(topics[i])
	}

	wg2 := sync.WaitGroup{}
	wg2.Add(len(subscribers))
	// Read from the subscription channels
	for _, ch := range subscribers {
		go func(ch chan interface{}) {
			defer wg2.Done()
			log.Println("Received:", <-ch)
		}(ch)
	}

	log.Println("Waiting for events...")
	wg.Wait()
	wg2.Wait()
}
