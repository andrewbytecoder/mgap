package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"
)

func main() {
	fmt.Println("pprof server running at http://localhost:6060/debug/pprof")

	go func() {
		n := 10
		for i := 1; i <= 100000; i++ {
			fmt.Printf("fib(%d)=%d\n", n, fib(n))
			n += 3 * i
		}
		time.Sleep(time.Second * 10)
	}()

	log.Fatal(http.ListenAndServe(":6060", nil))
}

func fib(n int) int {
	if n <= 1 {
		return 1
	}

	return fib(n-1) + fib(n-2)
}
