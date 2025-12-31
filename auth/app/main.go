package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	appName  = getEnv("APP_NAME", "app")
	hostname = getHostname()
)

type CowsayRequest struct {
	Message string `json:"message"`
}

type CowsayResponse struct {
	Cow      string `json:"cow"`
	Message  string `json:"message"`
	Service  string `json:"service"`
	Instance string `json:"instance"`
	User     string `json:"user"`
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// Middleware to verify gateway headers (authentication already done by gateway)
func verifyGatewayHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.Header.Get("X-Username")
		role := r.Header.Get("X-Role")

		if username == "" || role == "" {
			log.Printf("[%s/%s] Request denied: missing gateway headers", appName, hostname)
			http.Error(w, "Unauthorized: missing authentication headers", http.StatusUnauthorized)
			return
		}

		log.Printf("[%s/%s] Request from user: %s (role: %s)", appName, hostname, username, role)
		next(w, r)
	}
}

// Middleware to require specific role (AUTHORIZATION - app's responsibility)
func requireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return verifyGatewayHeaders(func(w http.ResponseWriter, r *http.Request) {
		userRole := r.Header.Get("X-Role")
		if userRole != role {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			log.Printf("[%s/%s] Access denied for user with role: %s (required: %s)", appName, hostname, userRole, role)
			return
		}
		next(w, r)
	})
}

func cowsayHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CowsayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		req.Message = "Hello from " + hostname + "!"
	}

	username := r.Header.Get("X-Username")

	cow := generateCowsay(req.Message)

	response := CowsayResponse{
		Cow:      cow,
		Message:  req.Message,
		Service:  appName,
		Instance: hostname,
		User:     username,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("[%s/%s] Cowsay request from user: %s", appName, hostname, username)
}

func generateCowsay(message string) string {
	messageLen := len(message)
	border := strings.Repeat("-", messageLen+2)

	var cow bytes.Buffer
	cow.WriteString(fmt.Sprintf(" %s \n", border))
	cow.WriteString(fmt.Sprintf("< %s >\n", message))
	cow.WriteString(fmt.Sprintf(" %s \n", border))
	cow.WriteString("        \\   ^__^\n")
	cow.WriteString("         \\  (oo)\\_______\n")
	cow.WriteString("            (__)\\       )\\/\\\n")
	cow.WriteString("                ||----w |\n")
	cow.WriteString("                ||     ||\n")

	return cow.String()
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.Header.Get("X-Username")
	role := r.Header.Get("X-Role")

	response := map[string]interface{}{
		"message":  "Welcome to the admin panel!",
		"user":     username,
		"role":     role,
		"service":  appName,
		"instance": hostname,
		"admin_features": []string{
			"View all users",
			"Manage permissions",
			"System configuration",
			"Audit logs",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("[%s/%s] Admin access granted to user: %s", appName, hostname, username)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "healthy",
		"service":  appName,
		"instance": hostname,
	})
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"service":  appName,
		"instance": hostname,
		"version":  "1.0.0",
		"note":     "Authentication handled by API Gateway",
	})
}

func main() {
	http.HandleFunc("/api/v1/cowsay", verifyGatewayHeaders(cowsayHandler))
	http.HandleFunc("/api/v1/admin", requireRole("admin", adminHandler))
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/info", infoHandler)

	port := getEnv("PORT", "8081")
	log.Printf("%s/%s starting on port %s", appName, hostname, port)
	log.Printf("Authentication: Trusting API Gateway headers")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
