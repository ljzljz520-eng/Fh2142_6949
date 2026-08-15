package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"example.com/spellingchallenge/internal/application"
)

func main() {
	address := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()
	fmt.Printf("Spelling Challenge listening on http://%s\n", *address)
	if err := http.ListenAndServe(*address, application.NewHandler(nil)); err != nil {
		log.Fatal(err)
	}
}
