package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
)

// Inspired by
// https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/#the-newserver-constructor
// How I write HTTP services in Go after 13 years

// mock the real implementations
type Config struct {
	Host string
	Port string
}

// DDD: Store is a repository of objects.
type Service struct{}

// Validator is an object that can be validated.
type Validator interface {
	// Valid checks the object and returns any
	// problems. If len(problems) == 0 then
	// the object is valid.
	Valid(ctx context.Context) (problems map[string]string)
}

type server struct {
	http.Handler
}

func (s *server) Use(middlewares ...func(http.Handler) http.Handler) {
	for _, middleware := range middlewares {
		s.Handler = middleware(s.Handler)
	}
}

func middlewareLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("request", slog.Any("method", r.Method), slog.Any("url", r.URL.String()))
			next.ServeHTTP(w, r)
			logger.Info("response", slog.Any("status", w.Header().Get("status")))
		})
	}
}

func NewServer(
	logger *slog.Logger,
	config *Config,
	service *Service,
) *server {
	mux := http.NewServeMux()
	addRoutes(
		mux,
		logger,
		config,
		service,
	)
	var handler http.Handler = mux
	server := &server{handler}
	server.Use(middlewareLog(logger))
	return server
}

func addRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	config *Config,
	service *Service,
) {
	mux.Handle("/api/v1", handleSomething(logger, service))
	mux.HandleFunc("/healthz", handleHealthz(logger))
	mux.Handle("/", http.NotFoundHandler())
}

func handleHealthz(logger *slog.Logger) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		},
	)
}

type something struct {
	Name string `json:"name"`
}

func (s *something) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)
	if s.Name == "" {
		problems["name"] = "required"
	}
	return problems
}

func handleSomething(logger *slog.Logger, service *Service) http.Handler {
	// do something with service
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// use thing to handle request
			logger.Info("handle something")
			switch r.Method {
			case http.MethodPost:
				thing, problems, err := decodeValid[*something](r)
				if err != nil {
					logger.ErrorContext(r.Context(), "decode valid", slog.Any("err", err))
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if len(problems) > 0 {
					if err := encode(w, r, http.StatusBadRequest, problems); err != nil {
						logger.ErrorContext(r.Context(), "encode", slog.Any("err", err))
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
					return
				}
				// do something with thing
				logger.Info("thing", slog.Any("thing", thing))
				w.WriteHeader(http.StatusCreated)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		},
	)
}

func encode[T any](w http.ResponseWriter, r *http.Request, status int, v T) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

func decodeValid[T Validator](r *http.Request) (T, map[string]string, error) {
	v, err := decode[T](r)
	if err != nil {
		return v, nil, fmt.Errorf("decode json: %w", err)
	}
	if problems := v.Valid(r.Context()); len(problems) > 0 {
		return v, problems, fmt.Errorf("invalid %T: %d problems", v, len(problems))
	}
	return v, nil, nil
}

func run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	config := &Config{
		Host: "localhost",
		Port: "8080",
	}
	logger := slog.New(slog.NewTextHandler(stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	service := &Service{}
	srv := NewServer(
		logger,
		config,
		service,
	)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.Host, config.Port),
		Handler: srv,
	}
	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if err := httpServer.Shutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()
	wg.Wait()
	return nil
}

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
