package main

import (
	"io"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, req *http.Request) {
	log.Printf("HTTP: Request %s: %s\n", req.Method, req.URL)
	io.WriteString(w, "nil\n")
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("HTTP: Started at 192.168.9.11:8080")
	log.Println(http.ListenAndServe("192.168.0.11:8080", nil))
}
