package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-super-secret-key-change-in-production")

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ValidateRequest struct {
	Token string `json:"token"`
}

type ValidateResponse struct {
	Valid    bool   `json:"valid"`
	Username string `json:"username,omitempty"`
	Role     string `json:"role,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Simple in-memory user store (in production, use a database)
var users = map[string]string{
	"alice": "password123",
	"bob":   "password456",
	"admin": "admin123",
}

// User roles
var userRoles = map[string]string{
	"alice": "user",
	"bob":   "user",
	"admin": "admin",
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate credentials
	expectedPassword, exists := users[creds.Username]
	if !exists || expectedPassword != creds.Password {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Create JWT token
	expirationTime := time.Now().Add(15 * time.Minute)
	claims := &Claims{
		Username: creds.Username,
		Role:     userRoles[creds.Username],
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token:     tokenString,
		ExpiresAt: expirationTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("Token issued for user: %s", creds.Username)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse and validate token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(req.Token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	response := ValidateResponse{}

	if err != nil || !token.Valid {
		response.Valid = false
		response.Error = "Invalid or expired token"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	response.Valid = true
	response.Username = claims.Username
	response.Role = claims.Role

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("Token validated for user: %s (role: %s)", claims.Username, claims.Role)
}

// Validate token from Authorization header
func validateHeaderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ValidateResponse{Valid: false, Error: "No authorization header"})
		return
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ValidateResponse{Valid: false, Error: "Invalid authorization header format"})
		return
	}

	tokenString := parts[1]

	// Parse and validate token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	response := ValidateResponse{}

	if err != nil || !token.Valid {
		response.Valid = false
		response.Error = "Invalid or expired token"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	response.Valid = true
	response.Username = claims.Username
	response.Role = claims.Role

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("Token validated for user: %s (role: %s)", claims.Username, claims.Role)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func main() {
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/validate", validateHandler)
	http.HandleFunc("/validate-header", validateHeaderHandler)
	http.HandleFunc("/health", healthHandler)

	port := ":8080"
	log.Printf("Auth server starting on port %s", port)
	log.Printf("Available users: alice, bob, admin")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
