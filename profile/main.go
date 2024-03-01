package main

import (
	"exp/profile/hw"
	"flag"
	"fmt"
	"net/http"
	"sync"

	_ "net/http/pprof"
)

func main() {
	var wg sync.WaitGroup

	// if flag `-debug` is passed, start the pprof server
	debugFlag := flag.Bool("debug", false, "Enable pprof server")
	flag.Parse()
	if *debugFlag {
		fmt.Println("Starting pprof server on localhost:6060")
		wg.Add(1) // pprof - so we won't exit prematurely
		go func() {
			fmt.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	wg.Add(1) // for the hardWork
	go func() {
		hw.HardWork()
		wg.Done()
	}()
	wg.Wait()
}
