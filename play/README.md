
### Category 1: Go Concurrency & Synchronization

*Why:* Because it handles real-time streams (LiveKit/WebRTC) and event-driven AI inputs, you will almost certainly face a question that tests how cleanly you manage asynchronous data without data races or resource leaks.

#### 1. The Concurrency-Limited Worker Pool

* **The Problem:** You are given an array of jobs (e.g., URLs to process or video frames to analyze). Implement a pool that processes these jobs using exactly `N` concurrent worker goroutines. The function should return when all jobs are processed, cleanly closing all channels and aggregating any errors.
* **What it tests:** Proper use of `sync.WaitGroup`, channel closing semantics, avoiding goroutine leaks, and managing context cancellation.

#### 2. Thread-Safe Event Fan-In (Multiplexing)

* **The Problem:** Write a function that takes an arbitrary number of read-only channels (`<-chan Message`) and merges them into a single output channel. The output channel must close cleanly when all input channels are exhausted.
* **What it tests:** Dynamic orchestration of channel reading, understanding how to prevent writing to closed channels, and managing dynamic `reflect.Select` or coordination via wait groups.

---

### Category 2: Streaming & Rate Limiting

*Why:* Processing real-time computer vision/audio packets means dealing with data spikes, backpressure, and sliding windows.

#### 3. Token Bucket Rate Limiter

* **The Problem:** Implement a basic thread-safe rate limiter in Go (or object-oriented pseudocode if you prefer, but Go is highly recommended here) with a single method `Allow() bool`. It should allow up to `B` (burst) requests and replenish tokens at a rate of `R` tokens per second.
* **What it tests:** Mutex locks (`sync.Mutex`), tracking delta times, and resource control.

#### 4. Moving Average over a Slidng Time Window

* **The Problem:** You receive a constant stream of processing durations from an NLP inference server. Implement a tracker that calculates the rolling average of the execution time over the *last 10 seconds* or the *last N items*.
* **What it tests:** Fast data structures (like a circular buffer or a queue), timestamps, and optimizing for quick updates ($O(1)$ insertions).

---

### Category 3: Access Control & Data Filtering

*Why:* The job description heavily features RBAC (Role-Based Access Control), tenant isolation, and microservice boundaries.

#### 5. Efficient Permission Evaluator

* **The Problem:** Given a list of nested role permissions (e.g., `"admin" -> inherits "editor" -> inherits "viewer"`), write an evaluation engine that determines if a user with a specific role has access to a resource action (like `document:delete`).
* **What it tests:** Graph/Tree traversal (DFS/BFS), cycle detection, and optimizing lookups via maps for fast evaluation.

#### 6. Log / Event Stream Parser

* **The Problem:** You are reading a massive, stream-like line-by-line log format (or JSON payloads) from a simulated Kafka broker containing metadata about AI pipelines. Parse out metrics, count anomalies, or group messages by type using minimal memory allocations.
* **What it tests:** Using `bufio.Scanner`, handling errors gracefully, strings/bytes manipulation, and memory performance.

---

### Your 30-Minute Execution Strategy

Because 30 minutes flies by, follow this rigorous timeline during the session to guarantee you cross the finish line:

1. **Minutes 0-5 (Clarify):** Do not write a line of code. State assumptions explicitly. *“Is this stream bounded or infinite?” “Should I optimize for low latency or memory footprint?” “How do you want errors handled?”*
2. **Minutes 5-20 (The Blueprint & Code):** Write down your structural approach in comments first, then code. If you’re writing Go, lean into idioms: check your errors immediately, make your struct fields private unless needed, and utilize `defer` for resource cleanup.
3. **Minutes 20-25 (Edge Cases):** Verbally step through the code with empty inputs, negative numbers, or channel closures.
4. **Minutes 25-30 (Lead Level Review):** End your presentation by stating how you would scale this if it were a production system (e.g., adding metrics hooks, using `sync.Pool` to avoid garbage collection pressure, or graceful shutdowns using a `context.Context`). This elevates you from a "coder" to a "Lead Candidate."