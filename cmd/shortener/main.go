package main

import (
	"context"
	"flag"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/controllers"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/router"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/token"
	"github.com/caarlos0/env/v6"
	"github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
	"log"
	"time"
)

type config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080/"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	SecretSignKey   string `env:"SECRET_KEY" envDefault:"secret key"`
	// envDefault:"postgres://postgres:password@localhost:5433/postgres"
	DataBaseDsn string `env:"DATABASE_DSN"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "Адрес сервера, где будет работать приложение")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Базовый урл сокращенной ссылки")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "Путь до бекап файла")
	flag.StringVar(&cfg.SecretSignKey, "k", cfg.SecretSignKey, "Секретный ключ для создания подписи")
	flag.StringVar(&cfg.DataBaseDsn, "d", cfg.DataBaseDsn, "Ссылка для подключения к базе данных")
	flag.Parse()
	var db *pgx.Conn
	var err error
	c, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	if len(cfg.DataBaseDsn) != 0 {
		db, err = pgx.Connect(c, cfg.DataBaseDsn)
		if err != nil {
			log.Fatal(err)
		}
		err = db.Ping(c)
		if err != nil {
			log.Fatal(err)
		}
	}
	urlRepo, userRepo, err := repository.InitRepositories(c, cfg.FileStoragePath, db)
	if err != nil {
		log.Fatal(err)
	}
	tb := token.InitTokenBuilder(cfg.SecretSignKey)
	controller := controllers.InitController(cfg.BaseURL, db, tb, urlRepo, userRepo)
	r := router.InitAPI(controller, tb)
	log.Fatal(r.Run(cfg.ServerAddress))
}
