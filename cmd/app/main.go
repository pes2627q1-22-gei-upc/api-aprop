package main

import (
	"log"
	"net/http"
	. "api-aprop/internal/router"
	. "api-aprop/internal/repositories"
)

func main() {
	LoadOfficialEvents()

	log.Fatal(http.ListenAndServe(":8080", MakeRouter()))
}