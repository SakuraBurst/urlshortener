package main

import (
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/api"
	"github.com/caarlos0/env/v6"
	"log"
)

type config struct {
	ServerAddress string `env:"SERVER_ADDRESS" envDefault:"http://localhost:8080/"`
	BaseURL       string `env:"BASE_URL" envDefault:"http://localhost:8080/"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}
	router := api.InitAPI(cfg.BaseURL)
	log.Fatal(router.Run(":8080"))
}
