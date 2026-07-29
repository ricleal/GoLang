# Task Management Clean Architecture PoC

This PoC implements one full task slice: create a task, retrieve it through HTTP or Cobra CLI, and store it in DuckDB. The root module is `exp`, so run all commands from the repository root.

## Layers and Diagram Mapping

The project has **four architectural layers**. Dependencies always point toward the center. The `cmd/taskapp` folder is not a fifth layer: it is the outermost composition root that creates concrete infrastructure objects and injects them into the inner layers.

| Diagram level | Project folder | Responsibility | Concrete examples |
| --- | --- | --- | --- |
| 1. Entities / Enterprise Business Rules | `domain/` | Core business concepts, enterprise rules, and abstractions. It has no framework or database imports. | `Task`, `NewTask`, `TaskRepository`, domain errors |
| 2. Use Cases / Application Business Rules | `application/` | Application workflows that coordinate entities through domain abstractions. | `CreateTask.Execute`, `GetTask.Execute` |
| 3. Interface Adapters | `adapters/` | Converts HTTP and CLI input/output to and from use-case input/output. | `adapters/http/TaskController`, `adapters/cli/TaskPresenter` |
| 4. Frameworks and Drivers | `infrastructure/` | Framework-specific details and implementations of inner-layer abstractions. | DuckDB `TaskRepository`, `http.Server` |

### Where the Use Cases Are

The use cases in the original diagram are implemented in `application/`:

- `CreateTask` in `application/create_task.go` validates and persists a new `Task` through the `domain.TaskRepository` abstraction.
- `GetTask` in `application/get_task.go` retrieves a `Task` through the same abstraction.

`adapters/http/TaskController` is the diagram's **Controller**: it converts an HTTP request into `application.CreateTaskInput` or a task ID, then calls a use case. `adapters/cli/TaskPresenter` is the **Presenter**: it turns use-case results into CLI text. `infrastructure/duckdb/TaskRepository` is the database **Gateway** implementation. The repository interface itself remains in `domain/`, as required by this PoC, so the persistence dependency points inward.

```text
HTTP request or Cobra command
            |
            v
adapters/ (Controller or Presenter)
            |
            v
application/ (CreateTask or GetTask use case)
            |
            v
domain/ (Task entity and TaskRepository abstraction)
            ^
            |
infrastructure/ (DuckDB TaskRepository implementation)
```

## Directory Tree

```text
clean_architecture/
├── domain/                         # Entities, enterprise rules, repository abstractions
│   ├── task.go
│   └── task_repository.go
├── application/                    # Use cases; depends only on domain
│   ├── create_task.go
│   ├── create_task_test.go
│   └── get_task.go
├── adapters/                       # Input/output conversion for delivery mechanisms
│   ├── cli/task_presenter.go
│   └── http/task_controller.go
├── infrastructure/                 # DuckDB implementation and HTTP server driver
│   ├── duckdb/task_repository.go
│   └── server/server.go
└── cmd/taskapp/main.go              # Composition root: pure dependency injection
```

Dependencies point inward. The composition root is allowed to reference every layer only to wire them together:

```text
infrastructure -> adapters -> application -> domain
cmd/taskapp -> all layers (composition only)
```

The domain has no framework or database imports. `domain.TaskRepository` is the persistence abstraction; the DuckDB adapter implements it in the infrastructure layer.

## Run

Build the executable:

```bash
go build -o taskapp ./clean_architecture/cmd/taskapp
```

Create and retrieve a task through Cobra:

```bash
./taskapp --database /tmp/tasks.duckdb create "Write architecture proposal"
./taskapp --database /tmp/tasks.duckdb get 1
```

Run the API against the same database:

```bash
./taskapp --database /tmp/tasks.duckdb serve --address :8080
curl -i -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Write architecture proposal"}'
curl -i http://localhost:8080/tasks/1
```

The API exposes `POST /tasks` and `GET /tasks/{id}`. A blank task title returns `400`; an unknown task ID returns `404`.

## Verify

```bash
go test ./clean_architecture/...
go vet ./clean_architecture/...
```