package main

import (
	"github.com/SakuraBurst/urlshortener/internal/app/api"
	"log"
)

func main() {
	router := api.InitAPI()
	log.Fatal(router.Run(":8080"))
}
