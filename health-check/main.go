package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type Message struct {
	ServiceName string `json:"service_name"`
	Alive       bool   `json:"alive"`
	Code        int    `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}

type registry struct {
	sync.Mutex
	services map[string]time.Time
}

func newRegistry() *registry {
	return &registry{
		services: make(map[string]time.Time),
	}
}

func (r *registry) register(serviceName string) {
	r.Lock()
	r.services[serviceName] = time.Now()
	r.Unlock()
}

func (r *registry) heartbeat(serviceName string) (bool, error) {
	r.Lock()
	_, ok := r.services[serviceName]
	r.Unlock()

	if !ok {
		return false, fmt.Errorf("no service with name %s found", serviceName)
	}
	r.Lock()
	r.services[serviceName] = time.Now()
	r.Unlock()
	return true, nil

}

func (r *registry) checkStatus(serviceName string) bool {
	r.Lock()
	lastHeartbeat, ok := r.services[serviceName]
	r.Unlock()

	if !ok {
		return false
	}
	return time.Since(lastHeartbeat) < 5*time.Second // Consider service alive for 5 seconds after last heartbeat
}

func main() {
	r := newRegistry()

	router := mux.NewRouter()
	router.HandleFunc("/register/{name}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		serviceName := vars["name"]
		r.register(serviceName)
		m := Message{
			ServiceName: serviceName,
			Alive:       true,
			Code:        http.StatusOK,
		}
		// send response in JSON format
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(m)
	}).Methods("POST")

	router.HandleFunc("/heartbeat/{name}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		serviceName := vars["name"]
		r.heartbeat(serviceName)
		m := Message{
			ServiceName: serviceName,
			Alive:       true,
			Code:        http.StatusOK,
		}
		// send response in JSON format
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(m)
	}).Methods("POST")

	router.HandleFunc("/status/{name}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		serviceName := vars["name"]
		alive := r.checkStatus(serviceName)
		m := Message{
			ServiceName: serviceName,
			Alive:       alive,
		}
		if alive {
			m.Code = http.StatusOK
		} else {
			m.Code = http.StatusNotFound
		}
		// send response in JSON format
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(m.Code)
		_ = json.NewEncoder(w).Encode(m)
	}).Methods("GET")

	router.HandleFunc("/status", func(w http.ResponseWriter, req *http.Request) {
		r.Lock()
		defer r.Unlock()
		var messages []Message
		for serviceName, lastHeartbeat := range r.services {
			alive := time.Since(lastHeartbeat) < 5*time.Second
			m := Message{
				ServiceName: serviceName,
				Alive:       alive,
				Code:        http.StatusOK,
			}
			messages = append(messages, m)
		}
		// send response in JSON format
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(messages)

	}).Methods("GET")

	fmt.Println("Registry listening on port 8080")
	http.ListenAndServe(":8080", router)
}
