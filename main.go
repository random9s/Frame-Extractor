package main

import (
	"log"
	"net/http"
)

func main() {
	router := NewRouter()
	router.PathPrefix("/temps/").Handler(
		http.StripPrefix("/temps/", http.FileServer(http.Dir("temps/"))),
	)

	log.Fatal(http.ListenAndServe(":8080", router))
}
