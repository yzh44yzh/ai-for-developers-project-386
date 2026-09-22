package main

import (
	"log"
	"net/http"

	"github.com/yzh44yzh/bookmeet/internal/hello"
	"github.com/yzh44yzh/bookmeet/internal/home"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", home.Handler)
	mux.HandleFunc("GET /hello", hello.Handler)
	mux.HandleFunc("POST /hello", hello.BodyLogger)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
