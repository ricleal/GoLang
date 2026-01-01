# Go Experiments

A collection of Go experiments, proof-of-concepts, and learning projects exploring various aspects of the Go programming language and its ecosystem.

## 🎯 Featured Projects

### [Authentication & Authorization PoC](auth/)
A production-ready microservices architecture demonstrating:
- JWT-based authentication with role-based access control (RBAC)
- Traefik load balancer with rate limiting (DDoS protection)
- Docker Compose with multiple replicas (horizontal scaling)
- Decoupled authentication/authorization pattern
- Health checks and monitoring

**Tech Stack**: Traefik, JWT, Docker Compose, microservices architecture

See [auth/README.md](auth/README.md) for detailed documentation.

---

## 📂 Project Categories

### 🔐 Authentication & Authorization
- **[auth/](auth/)** - Complete microservices auth system with JWT, RBAC, and load balancing

### 🌐 HTTP & APIs
- **[http_server/](http_server/)** - HTTP server implementations and patterns
- **[api/](api/)** - API development and benchmarking
- **[api_book/](api_book/)** - API examples from books/tutorials
- **[proxy/](proxy/)** - Reverse proxy implementations

### ⚡ Concurrency & Performance
- **[benchmarks/](benchmarks/)** - Performance benchmarks and comparisons
- **[n-producers_n-consumers/](n-producers_n-consumers/)** - Concurrency patterns
- **[crawler/](crawler/)** - Parallel vs serial web crawling
- **[thread-pool/](thread-pool/)** - Thread pool implementations
- **[ch_block/](ch_block/)** - Channel blocking examples

### 🏗️ Design Patterns & Architecture
- **[load_balancer/](load_balancer/)** - Load balancing algorithms (round-robin)
- **[load-shedding/](load-shedding/)** - Load shedding patterns
- **[rate_limiter/](rate_limiter/)** - Rate limiting implementations
- **[event_bus/](event_bus/)** - Event bus pattern
- **[pub-sub/](pub-sub/)** - Publish-subscribe pattern

### 🗄️ Database & Data
- **[gorm/](gorm/)** - GORM ORM examples
- **[bigquery/](bigquery/)** - Google BigQuery integration
- **[db-vector/](db-vector/)** - Vector database experiments
- **[lru/](lru/)** - LRU cache implementation

### 🔧 System Programming
- **[ctx/](ctx/)** - Context package examples (cancellation, signals)
- **[shut-down/](shut-down/)** - Graceful shutdown patterns
- **[wait_ready/](wait_ready/)** - Ready/health check implementations
- **[file/](file/)** - File parsing and processing
- **[split_csv/](split_csv/)** - CSV file splitting utilities

### 🧪 Testing & Debugging
- **[profile/](profile/)** - Profiling and performance analysis
- **[pprof-pyroscope/](pprof-pyroscope/)** - Continuous profiling with Pyroscope
- **[err/](err/)** - Error handling patterns

### 📚 Data Structures & Algorithms
- **[stack/](stack/)** - Stack implementation
- **[iterator/](iterator/)** - Iterator patterns
- **[matrix/](matrix/)** - Matrix operations
- **[codility/](codility/)** - Coding challenges
- **[search/](search/)** - Search algorithms

### 🎲 Miscellaneous
- **[strings/](strings/)** - String manipulation
- **[bitwise/](bitwise/)** - Bitwise operations
- **[binary-protocol/](binary-protocol/)** - Binary protocol implementations
- **[funcs/](funcs/), [funcs2/](funcs2/), [funcs3/](funcs3/)** - Function patterns
- **[options/](options/)** - Options pattern
- **[viper_config/](viper_config/)** - Configuration with Viper
- **[log/](log/)** - Logging patterns
- **[ticker/](ticker/)** - Ticker examples
- **[scheduler/](scheduler/)** - Task scheduling
- **[notifications/](notifications/)** - Notification systems
- **[queue_send_emails/](queue_send_emails/)** - Email queue implementation
- **[health-check/](health-check/)** - Health check endpoints
- **[obrc/](obrc/)** - One Billion Row Challenge
- **[crdt/](crdt/)** - Conflict-free Replicated Data Types
- **[clone/](clone/)** - Deep cloning patterns
- **[refresh/](refresh/)** - Auto-refresh patterns
- **[assemble-file/](assemble-file/)** - File assembly utilities

### 🎯 Fun & Challenges
- **[fun/](fun/)** - Various fun projects and challenges

## 🚀 Getting Started

Most projects are self-contained and can be run individually:

```bash
# Navigate to a project directory
cd <project-name>

# Run the project
go run main.go

# Or run tests
go test ./...
```

For projects with specific requirements, check the project's directory for a README or documentation.

## 📋 Prerequisites

- Go 1.20 or higher
- Docker and Docker Compose (for containerized projects)

## 🤝 Contributing

These are personal experiments and learning projects. Feel free to explore and learn from them!

## 📝 License

This is a personal learning repository.

