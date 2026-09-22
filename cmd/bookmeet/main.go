package main

import (
	"log"
	"net/http"

	"github.com/yzh44yzh/bookmeet/internal/hello"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", hello.Handler)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
