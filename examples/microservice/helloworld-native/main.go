package main

import (
	"fmt"
	"log"
	"net/http"
)

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"status":"UP"}`)
}

func handleRequests() {
	http.HandleFunc("/health", health)
	log.Fatal(http.ListenAndServe(":80", nil))
}

func main() {
	handleRequests()
}
