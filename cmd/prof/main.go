package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	fmt.Println("pprof server running at http://localhost:6060/debug/pprof")
	log.Fatal(http.ListenAndServe(":6060", nil))
}
