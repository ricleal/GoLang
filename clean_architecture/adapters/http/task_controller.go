package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"exp/clean_architecture/application"
	"exp/clean_architecture/domain"
)

type TaskController struct {
	createTask application.CreateTask
	getTask    application.GetTask
}

func NewTaskController(createTask application.CreateTask, getTask application.GetTask) TaskController {
	return TaskController{createTask: createTask, getTask: getTask}
}

func (controller TaskController) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", controller.create)
	mux.HandleFunc("GET /tasks/{id}", controller.get)
	return mux
}

func (controller TaskController) create(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid JSON body")
		return
	}

	task, err := controller.createTask.Execute(request.Context(), application.CreateTaskInput{Title: input.Title})
	if errors.Is(err, domain.ErrEmptyTaskTitle) {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "could not create task")
		return
	}

	writeJSON(writer, http.StatusCreated, task)
}

func (controller TaskController) get(writer http.ResponseWriter, request *http.Request) {
	id, err := strconv.ParseInt(request.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(writer, http.StatusBadRequest, "task ID must be a positive integer")
		return
	}

	task, err := controller.getTask.Execute(request.Context(), id)
	if errors.Is(err, domain.ErrTaskNotFound) {
		writeError(writer, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "could not get task")
		return
	}

	writeJSON(writer, http.StatusOK, task)
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": strings.TrimSpace(message)})
}
