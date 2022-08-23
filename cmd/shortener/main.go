package main

import (
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/api"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/controlers"
	"github.com/caarlos0/env/v6"
	"log"
)

type config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080/"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"backUp.log"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}
	controlers.InitRepository(nil, cfg.FileStoragePath)
	router := api.InitAPI(cfg.BaseURL)
	log.Fatal(router.Run(cfg.ServerAddress))
}
