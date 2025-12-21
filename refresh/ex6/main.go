package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Person struct {
	Name string
	Age  int
}

type AddressBook struct {
	Persons []Person
}

func (a *AddressBook) ListPersons() []Person {
	return a.Persons
}

func (a *AddressBook) AddPerson(p Person) {
	a.Persons = append(a.Persons, p)
}

func (a *AddressBook) AddPersons(ps []Person) {
	a.Persons = append(a.Persons, ps...)
}

// ----

type API struct {
	AddressBook
	router *mux.Router
}

func NewAPI() *API {
	api := &API{
		AddressBook: AddressBook{},
	}

	r := mux.NewRouter()
	r.HandleFunc("/persons", api.GetPersons).Methods("GET")

	api.router = r
	return api
}

// This is mandatory so API implements the http.Handler
//
//	type Handler interface {
//		ServeHTTP(ResponseWriter, *Request)
//	}
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func (a *API) GetPersons(w http.ResponseWriter, r *http.Request) {
	persons := a.AddressBook.ListPersons()

	if persons == nil {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	personsJSON, err := json.Marshal(persons)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "` + err.Error() + `"}`))
		return
	}

	w.Write(personsJSON)
	w.WriteHeader(http.StatusAccepted)
}

// ---

// Main function
func main() {
	log.Println("Starting server on :8080")
	api := NewAPI()
	err := http.ListenAndServe(":8080", api)
	if err != nil {
		panic(err)
	}
}
