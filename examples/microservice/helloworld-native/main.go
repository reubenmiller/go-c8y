package main

import (
	"fmt"
	"net/http"
)

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"status":"UP"}`)
}

func handleRequests() {
	http.HandleFunc("/health", health)
	panic(http.ListenAndServe(":80", nil))
}

func main() {
	handleRequests()
}
