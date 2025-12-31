package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

var (
	authServerURL = getEnv("AUTH_SERVER_URL", "http://auth-server:8080")
	appURL        = getEnv("APP_URL", "http://app:8081")
)

type ValidateResponse struct {
	Valid    bool   `json:"valid"`
	Username string `json:"username,omitempty"`
	Role     string `json:"role,omitempty"`
	Error    string `json:"error,omitempty"`
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Auth middleware - validates token before proxying
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			log.Printf("Request denied: missing authorization header")
			return
		}

		// Validate token with auth server
		req, err := http.NewRequest("GET", authServerURL+"/validate-header", nil)
		if err != nil {
			log.Printf("Error creating validation request: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", authHeader)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error validating token: %v", err)
			http.Error(w, "Failed to validate token", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var validateResp ValidateResponse
		if err := json.NewDecoder(resp.Body).Decode(&validateResp); err != nil {
			log.Printf("Error decoding validation response: %v", err)
			http.Error(w, "Failed to validate token", http.StatusInternalServerError)
			return
		}

		if !validateResp.Valid {
			log.Printf("Request denied: invalid token")
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		log.Printf("Request authorized for user: %s (role: %s)", validateResp.Username, validateResp.Role)
		r.Header.Set("X-Username", validateResp.Username)
		r.Header.Set("X-Role", validateResp.Role)
		next(w, r)
	}
}

// Proxy handler - Docker DNS round-robin handles load balancing
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	backendURL, err := url.Parse(appURL)
	if err != nil {
		log.Printf("Error parsing backend URL: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Proxying request to: %s%s", appURL, r.URL.Path)

	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = backendURL.Host
		req.URL.Scheme = backendURL.Scheme
		req.URL.Host = backendURL.Host
	}

	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

// Login handler - proxy to auth server
func loginHandler(w http.ResponseWriter, r *http.Request) {
	authURL, err := url.Parse(authServerURL)
	if err != nil {
		log.Printf("Error parsing auth URL: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Proxying login request to auth server")

	proxy := httputil.NewSingleHostReverseProxy(authURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = authURL.Host
		req.URL.Scheme = authURL.Scheme
		req.URL.Host = authURL.Host
		req.URL.Path = "/login"
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Auth proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"role":   "api-gateway",
	})
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":        "api-gateway",
		"version":        "1.0.0",
		"auth_server":    authServerURL,
		"backend":        appURL,
		"load_balancing": "docker-dns-round-robin",
	})
}

func main() {
	// Public endpoints (no auth required)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/info", infoHandler)

	// Protected endpoints (auth required)
	http.HandleFunc("/api/v1/cowsay", authMiddleware(proxyHandler))
	http.HandleFunc("/api/v1/admin", authMiddleware(proxyHandler))

	port := getEnv("PORT", "8000")
	log.Printf("API Gateway starting on port %s", port)
	log.Printf("Auth Server: %s", authServerURL)
	log.Printf("Backend Service: %s", appURL)
	log.Printf("Load balancing: Docker DNS round-robin")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
