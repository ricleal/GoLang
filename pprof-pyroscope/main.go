package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
)

func lightHandler(w http.ResponseWriter, r *http.Request) {
	// light work
	list := []int{}
	for i := 0; i < 1000; i++ {
		list = append(list, i)
	}
	fmt.Fprintln(w, "light work done")
}

func mediumHandler(w http.ResponseWriter, r *http.Request) {
	// medium work
	list := []int{}
	for i := 0; i < 100000; i++ {
		list = append(list, i)
	}
	fmt.Fprintln(w, "medium work done")
}

func heavyHandler(w http.ResponseWriter, r *http.Request) {
	// heavy work
	list := []int{}
	for i := 0; i < 10000; i++ {
		for j := 0; j < 10; j++ {
			list = append(list, i)
		}
	}
	fmt.Fprintln(w, "heavy work done")
}

func main() {
	portFlag := flag.String("port", "8081", "Port to listen on")
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/{action}", pprof.Index)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))

	mux.HandleFunc("/light", lightHandler)
	mux.HandleFunc("/medium", mediumHandler)
	mux.HandleFunc("/heavy", heavyHandler)

	fmt.Printf("Listening on :%s...\n", *portFlag)
	if err := http.ListenAndServe(":"+*portFlag, mux); err != nil {
		panic(fmt.Errorf("error when starting or running http server: %w", err))
	}
}
