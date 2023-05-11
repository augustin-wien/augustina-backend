package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var (
	port = flag.String("port", ":3000", "Port to listen on")
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/hello" {
        http.Error(w, "404 not found.", http.StatusNotFound)
        return
    }

    if r.Method != "GET" {
        http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
    }

    fmt.Fprintf(w, "Hello World!")
}

func main() {
	http.HandleFunc("/hello", helloHandler)

	fmt.Printf("Starting server at port %s\n", *port)
	if err := http.ListenAndServe(*port, nil); err != nil {
        log.Fatal(err)
    }
}
