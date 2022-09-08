package main

import (
	"flag"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/controllers"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/router"
	"github.com/caarlos0/env/v6"
	"log"
)

type config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080/"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "Адрес сервера, где будет работать приложение")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Базовый урл сокращенной ссылки")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "Путь до бекап файла")
	flag.Parse()
	urlRepo, err := repository.InitURLRepository(cfg.FileStoragePath)
	if err != nil {
		log.Fatal(err)
	}
	userRepo, err := repository.InitUserRepository()
	if err != nil {
		log.Fatal(err)
	}
	controller := controllers.InitController(cfg.BaseURL, urlRepo, userRepo)
	r := router.InitAPI(controller)
	log.Fatal(r.Run(cfg.ServerAddress))
}
