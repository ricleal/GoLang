package server

import (
	"net/http"

	httpadapter "exp/clean_architecture/adapters/http"
)

func New(address string, controller httpadapter.TaskController) *http.Server {
	return &http.Server{Addr: address, Handler: controller.Routes()}
}
