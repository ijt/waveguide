package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var addr = flag.String("a", ":4089", "server address")

func main() {
	http.HandleFunc("/", handleRoot)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "wave hunter")
}
