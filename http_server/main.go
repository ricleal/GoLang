package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"exp/http_server/store"

	"github.com/lmittmann/tint"
)

// Inspired by
// https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/#the-newserver-constructor
// How I write HTTP services in Go after 13 years

// mock the real implementations.
type Config struct {
	Host string
	Port string
}

var config = &Config{
	Host: "localhost",
	Port: "8080",
}

// Validator is an object that can be validated.
type Validator interface {
	// Valid checks the object and returns any
	// problems. If len(problems) == 0 then
	// the object is valid.
	Valid(ctx context.Context) (problems map[string]string)
}

func Logger(w io.Writer, levelAsString string) *slog.Logger {
	var level slog.Level

	switch strings.ToLower(levelAsString) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "Error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      level,
			TimeFormat: time.TimeOnly,
		}),
	)
	return logger
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
	st store.Store,
) *server {
	mux := http.NewServeMux()
	addRoutes(
		mux,
		logger,
		config,
		st,
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
	st store.Store,
) {
	// unused args
	_ = config
	mux.Handle("GET /api/v1", handleSomething(logger, st))
	mux.HandleFunc("GET /healthz", handleHealthz(logger))
	mux.Handle("GET /", http.NotFoundHandler())
}

func handleHealthz(log *slog.Logger) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Debug("server is healthy")
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

func handleSomething(logger *slog.Logger, st store.Store) http.Handler {
	// do something with store
	logger.Debug("handleSomething", slog.Any("store", st))
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
	getEnvFunc func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	// unused args
	_ = args
	_ = stdin
	_ = stdout

	log := Logger(stderr, getEnvFunc("LOG_LEVEL"))
	st := store.NewMemStore()
	srv := NewServer(
		log,
		config,
		st,
	)

	httpServer := &http.Server{
		Addr:              net.JoinHostPort(config.Host, config.Port),
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("starting server", slog.String("addr", httpServer.Addr))
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen and serve", tint.Err(err))
		}
	}()

	<-ctx.Done()
	stop()

	log.Info("shutting down gracefully, press Ctrl+C again to force")

	// Perform application shutdown with a maximum timeout of 5 seconds.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownCh := make(chan struct{})
	go func() {
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Error("shutdown", tint.Err(err))
			os.Exit(2)
		}
		shutdownCh <- struct{}{}
	}()

	select {
	case <-timeoutCtx.Done():
		// canceled or timed out
		if timeoutCtx.Err() == context.DeadlineExceeded {
			log.Info("graceful shutdown timed out", tint.Err(timeoutCtx.Err()))
			return timeoutCtx.Err()
		}
		log.Info("graceful shutdown canceled")
	case <-shutdownCh:
		log.Info("gracefully shutdown")
	}

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
